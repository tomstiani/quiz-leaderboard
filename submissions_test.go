package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var testPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")

func fakeVision(t *testing.T, result visionResult, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("missing authorization header")
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode vision request: %v", err)
		}
		encoded, _ := json.Marshal(request)
		if request["model"] != "test-model" || request["response_format"] == nil || !bytes.Contains(encoded, []byte("data:image/png;base64,")) || !bytes.Contains(encoded, []byte("completed final results page")) {
			t.Errorf("unexpected vision request: %s", encoded)
		}
		content, _ := json.Marshal(result)
		writeJSON(w, http.StatusOK, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}}})
	}))
	t.Cleanup(server.Close)
	return server
}

func submissionHandler(t *testing.T, result visionResult, calls *atomic.Int32) (http.Handler, *sql.DB, config, *eventBroker) {
	t.Helper()
	server := fakeVision(t, result, calls)
	cfg := testConfig()
	cfg.DataDir = t.TempDir()
	cfg.Vision = serviceConfig{URL: server.URL, Model: "test-model", APIKey: "test-api-key"}
	db, err := openDatabase(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	broker := newEventBroker()
	handler, err := newHandler(cfg, db, func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }, broker)
	if err != nil {
		t.Fatal(err)
	}
	return handler, db, cfg, broker
}

func upload(t *testing.T, handler http.Handler, cookie *http.Cookie, gameID, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("screenshot", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/games/"+gameID+"/draft", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("gameID", gameID)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func confirm(t *testing.T, handler http.Handler, cookie *http.Cookie, id string, score any) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"score": score})
	req := httptest.NewRequest(http.MethodPost, "/api/drafts/"+id+"/confirm", bytes.NewReader(body))
	req.SetPathValue("id", id)
	req.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestSubmissionFlowAndPrivateScreenshots(t *testing.T) {
	var calls atomic.Int32
	score := 321
	handler, _, cfg, broker := submissionHandler(t, visionResult{Valid: true, Score: &score}, &calls)
	alice, _ := login(t, handler, cfg.Players[0].Token)
	bob, _ := login(t, handler, cfg.Players[1].Token)

	response := upload(t, handler, alice, "geopolitix", "score.png", testPNG)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload returned %d: %s", response.Code, response.Body.String())
	}
	var draft draftResponse
	if err := json.NewDecoder(response.Body).Decode(&draft); err != nil || draft.Score == nil || *draft.Score != score {
		t.Fatalf("unexpected draft: %+v, %v", draft, err)
	}

	if response := screenshotRequest(handler, nil, draft.ID); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated draft returned %d", response.Code)
	}
	if response := screenshotRequest(handler, bob, draft.ID); response.Code != http.StatusNotFound {
		t.Fatalf("other player draft returned %d", response.Code)
	}
	if response := screenshotRequest(handler, alice, draft.ID); response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("owner draft screenshot returned %d", response.Code)
	}

	if response := confirm(t, handler, bob, draft.ID, score); response.Code != http.StatusNotFound {
		t.Fatalf("other player confirmation returned %d", response.Code)
	}
	events, unsubscribe := broker.subscribe()
	defer unsubscribe()
	if response := confirm(t, handler, alice, draft.ID, score); response.Code != http.StatusOK {
		t.Fatalf("confirmation returned %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-events:
	default:
		t.Fatal("confirmation did not publish leaderboard event")
	}
	if response := screenshotRequest(handler, bob, draft.ID); response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), testPNG) {
		t.Fatalf("confirmed screenshot returned %d or wrong bytes", response.Code)
	}
	if response := upload(t, handler, alice, "geopolitix", "again.png", testPNG); response.Code != http.StatusConflict {
		t.Fatalf("duplicate upload returned %d", response.Code)
	}
	if calls.Load() != 1 {
		t.Fatalf("vision called %d times, want 1", calls.Load())
	}

	dashboard := request(t, handler, http.MethodGet, "/api/dashboard", nil, bob)
	var result dashboardResponse
	if err := json.NewDecoder(dashboard.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	normalized := normalizeScore(score, 900)
	if result.Players[0].Rank != 1 || result.Players[0].Completed != 1 || result.Players[0].CombinedScore != normalized || len(result.Players[0].Scores) != 1 || result.Players[0].Scores[0].RawScore != score || result.Players[0].Scores[0].NormalizedScore != normalized {
		t.Fatalf("dashboard missing normalized score: %+v", result.Players[0])
	}
	if result.Players[1].Rank != 2 || result.Players[1].CombinedScore != 0 {
		t.Fatalf("unfinished player ranked incorrectly: %+v", result.Players[1])
	}
}

func screenshotRequest(handler http.Handler, cookie *http.Cookie, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/screenshots/"+id, nil)
	req.SetPathValue("id", id)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestUnreadableScoreCanBeEnteredManually(t *testing.T) {
	var calls atomic.Int32
	handler, _, cfg, _ := submissionHandler(t, visionResult{Valid: true, Score: nil}, &calls)
	alice, _ := login(t, handler, cfg.Players[0].Token)
	response := upload(t, handler, alice, "geopolitix", "score.png", testPNG)
	var draft draftResponse
	if response.Code != http.StatusCreated || json.NewDecoder(response.Body).Decode(&draft) != nil || draft.Score != nil {
		t.Fatalf("unexpected unreadable-score draft: %d %+v", response.Code, draft)
	}
	if response := confirm(t, handler, alice, draft.ID, 42); response.Code != http.StatusOK {
		t.Fatalf("manual score returned %d: %s", response.Code, response.Body.String())
	}
}

func TestUploadValidationAndModelRejection(t *testing.T) {
	var calls atomic.Int32
	handler, db, cfg, _ := submissionHandler(t, visionResult{Valid: false, Reason: "not final results"}, &calls)
	alice, _ := login(t, handler, cfg.Players[0].Token)
	owner, _ := login(t, handler, cfg.OwnerToken)

	cases := []struct {
		name    string
		game    string
		content []byte
		want    int
	}{
		{"unknown game", "unknown", testPNG, http.StatusNotFound},
		{"unsupported file", "geopolitix", []byte("not an image"), http.StatusUnsupportedMediaType},
		{"oversized file", "geopolitix", bytes.Repeat([]byte("x"), maxScreenshotSize+1), http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if response := upload(t, handler, alice, tc.game, "file.bin", tc.content); response.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", response.Code, tc.want, response.Body.String())
			}
		})
	}
	if response := upload(t, handler, owner, "geopolitix", "score.png", testPNG); response.Code != http.StatusForbidden {
		t.Fatalf("owner upload returned %d", response.Code)
	}
	if response := upload(t, handler, alice, "geopolitix", "score.png", testPNG); response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "not final results") {
		t.Fatalf("model rejection returned %d: %s", response.Code, response.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("vision called %d times, want only the valid image", calls.Load())
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM submissions").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid upload persisted: count=%d err=%v", count, err)
	}
}

func TestScoreValidationAndDraftReplacement(t *testing.T) {
	var calls atomic.Int32
	score := 700
	handler, db, cfg, _ := submissionHandler(t, visionResult{Valid: true, Score: &score}, &calls)
	alice, _ := login(t, handler, cfg.Players[0].Token)

	first := upload(t, handler, alice, "krillion", "first.png", testPNG)
	var oldDraft draftResponse
	json.NewDecoder(first.Body).Decode(&oldDraft)
	second := upload(t, handler, alice, "krillion", "second.png", testPNG)
	var draft draftResponse
	json.NewDecoder(second.Body).Decode(&draft)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated || draft.ID == oldDraft.ID {
		t.Fatalf("draft replacement failed: %d %d %+v %+v", first.Code, second.Code, oldDraft, draft)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "screenshots", oldDraft.ID+".png")); !os.IsNotExist(err) {
		t.Fatalf("old draft file still exists: %v", err)
	}

	for _, value := range []any{nil, -1, 701} {
		if response := confirm(t, handler, alice, draft.ID, value); response.Code != map[bool]int{true: http.StatusBadRequest, false: http.StatusUnprocessableEntity}[value == nil] {
			t.Fatalf("score %v returned %d", value, response.Code)
		}
	}
	if response := confirm(t, handler, alice, draft.ID, 700); response.Code != http.StatusOK {
		t.Fatalf("valid score returned %d: %s", response.Code, response.Body.String())
	}
	if response := confirm(t, handler, alice, draft.ID, 700); response.Code != http.StatusNotFound {
		t.Fatalf("second confirmation returned %d", response.Code)
	}
	var status string
	var normalized float64
	if err := db.QueryRow("SELECT status, normalized_score FROM submissions WHERE id = ?", draft.ID).Scan(&status, &normalized); err != nil || status != "confirmed" || normalized != 100 {
		t.Fatalf("submission status=%q normalized=%v err=%v", status, normalized, err)
	}
}

func TestDraftCleanup(t *testing.T) {
	var calls atomic.Int32
	score := 1
	_, db, cfg, _ := submissionHandler(t, visionResult{Valid: true, Score: &score}, &calls)
	path := filepath.Join(cfg.DataDir, "screenshots", "old.png")
	if err := os.WriteFile(path, testPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type, created_at)
		VALUES ('old', 'alice', 'geopolitix', '2026-09-09', 'draft', 'old.png', 'image/png', '2026-09-09 00:00:00')`)
	if err != nil {
		t.Fatal(err)
	}
	cleanupDrafts(db, cfg.DataDir, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("old file still exists: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM submissions WHERE id = 'old'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("old draft remains: count=%d err=%v", count, err)
	}
}

func TestDailyVisionAttemptLimit(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for attempt := 0; attempt < 10; attempt++ {
		allowed, err := claimVisionAttempt(db, "alice", "geopolitix", "2026-09-10")
		if err != nil || !allowed {
			t.Fatalf("attempt %d: allowed=%v err=%v", attempt+1, allowed, err)
		}
	}
	allowed, err := claimVisionAttempt(db, "alice", "geopolitix", "2026-09-10")
	if err != nil || allowed {
		t.Fatalf("eleventh attempt: allowed=%v err=%v", allowed, err)
	}
}

func TestVisionServiceErrorsDoNotPersist(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"provider error", http.StatusTooManyRequests, `{}`},
		{"malformed response", http.StatusOK, `{"choices":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			cfg := testConfig()
			cfg.DataDir = t.TempDir()
			cfg.Vision = serviceConfig{URL: server.URL, Model: "test", APIKey: "key"}
			db, _ := openDatabase(cfg.DataDir)
			defer db.Close()
			handler, _ := newHandler(cfg, db, time.Now)
			cookie, _ := login(t, handler, cfg.Players[0].Token)
			if response := upload(t, handler, cookie, "geopolitix", "score.png", testPNG); response.Code != http.StatusBadGateway {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}
