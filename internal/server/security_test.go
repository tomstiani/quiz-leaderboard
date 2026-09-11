package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeadersAndCrossSiteMutationProtection(t *testing.T) {
	cfg := testConfig()
	handler := testHandler(t, cfg, time.Now())

	response := request(t, handler, http.MethodGet, "/", nil, nil)
	for _, header := range []string{"Content-Security-Policy", "Cross-Origin-Resource-Policy", "Permissions-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if response.Header().Get(header) == "" {
			t.Fatalf("missing %s", header)
		}
	}

	body, _ := json.Marshal(map[string]string{"token": cfg.Players[0].Token})
	for _, headers := range []map[string]string{
		{"Origin": "https://attacker.example"},
		{"Sec-Fetch-Site": "cross-site"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		req.Body = http.NoBody
		for name, value := range headers {
			req.Header.Set(name, value)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusForbidden {
			t.Fatalf("cross-site headers %v returned %d", headers, response.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "https://leaderboard.example/api/login", bytes.NewReader(body))
	req.Header.Set("Origin", "https://leaderboard.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("same-origin login returned %d: %s", response.Code, response.Body.String())
	}
}
