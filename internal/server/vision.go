package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type visionResult struct {
	Valid  bool   `json:"valid"`
	Score  *int   `json:"score"`
	Reason string `json:"reason"`
}

type visionClient struct {
	url    string
	model  string
	apiKey string
	http   *http.Client
	slots  chan struct{}
}

func newVisionClient(cfg serviceConfig) visionClient {
	return visionClient{url: cfg.URL, model: cfg.Model, apiKey: cfg.APIKey, http: &http.Client{Timeout: 60 * time.Second}, slots: make(chan struct{}, 2)}
}

func (client visionClient) analyze(ctx context.Context, game gameConfig, mediaType string, image []byte) (visionResult, error) {
	if client.url == "" || client.model == "" || client.apiKey == "" {
		return visionResult{}, fmt.Errorf("vision service is not configured")
	}
	select {
	case client.slots <- struct{}{}:
		defer func() { <-client.slots }()
	case <-ctx.Done():
		return visionResult{}, ctx.Err()
	}
	prompt := fmt.Sprintf(`Analyze this screenshot from %s. It is valid only if it clearly shows the completed final results page for that game. Extract only the final total score, not a round score, percentile, rank, timer, date, or other number. For Krillion, return the total in points; if only depth in metres is displayed, divide it by 10. If the screenshot is valid but the total cannot be read, return a null score. Give a short reason when invalid.`, game.Name)
	requestBody := map[string]any{
		"model": client.model,
		"messages": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "text", "text": prompt},
				map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(image)}},
			},
		}},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "score_result",
				"strict": true,
				"schema": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"valid":  map[string]string{"type": "boolean"},
						"score":  map[string]any{"type": []string{"integer", "null"}},
						"reason": map[string]string{"type": "string"},
					},
					"required": []string{"valid", "score", "reason"},
				},
			},
		},
		"provider": map[string]bool{"require_parameters": true},
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return visionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url, bytes.NewReader(encoded))
	if err != nil {
		return visionResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "Daily Game Leaderboard")
	response, err := client.http.Do(req)
	if err != nil {
		return visionResult{}, fmt.Errorf("call vision service: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return visionResult{}, fmt.Errorf("vision service returned %s", response.Status)
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&completion); err != nil || len(completion.Choices) == 0 {
		return visionResult{}, fmt.Errorf("invalid vision service response")
	}
	var result visionResult
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &result); err != nil {
		return visionResult{}, fmt.Errorf("invalid structured vision result: %w", err)
	}
	return result, nil
}
