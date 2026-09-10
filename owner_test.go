package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func ownerTestHandler(t *testing.T) (http.Handler, config, *eventBroker) {
	t.Helper()
	cfg := testConfig()
	cfg.DataDir = t.TempDir()
	db, err := openDatabase(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := os.WriteFile(filepath.Join(cfg.DataDir, "screenshots", "today.png"), testPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.DataDir, "screenshots", "past.png"), testPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type, raw_score, normalized_score, confirmed_at)
		VALUES ('today', 'alice', 'geopolitix', '2026-09-10', 'confirmed', 'today.png', 'image/png', 450, 50, CURRENT_TIMESTAMP),
		       ('past', 'alice', 'geopolitix', '2026-09-09', 'confirmed', 'past.png', 'image/png', 300, 33.333, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	broker := newEventBroker()
	handler, err := newHandler(cfg, db, func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }, broker)
	if err != nil {
		t.Fatal(err)
	}
	return handler, cfg, broker
}

func ownerRequest(t *testing.T, handler http.Handler, cookie *http.Cookie, method, path, id string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var encoded []byte
	if body != nil {
		encoded, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.SetPathValue("id", id)
	req.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestOwnerListsAndCorrectsTodaysSubmission(t *testing.T) {
	handler, cfg, broker := ownerTestHandler(t)
	owner, _ := login(t, handler, cfg.OwnerToken)
	player, _ := login(t, handler, cfg.Players[0].Token)

	if response := ownerRequest(t, handler, player, http.MethodGet, "/api/owner/submissions", "", nil); response.Code != http.StatusForbidden {
		t.Fatalf("player owner-list returned %d", response.Code)
	}
	response := ownerRequest(t, handler, owner, http.MethodGet, "/api/owner/submissions", "", nil)
	var submissions []ownerSubmission
	if response.Code != http.StatusOK || json.NewDecoder(response.Body).Decode(&submissions) != nil || len(submissions) != 1 || submissions[0].ID != "today" {
		t.Fatalf("unexpected owner list: %d %+v", response.Code, submissions)
	}

	events, unsubscribe := broker.subscribe()
	defer unsubscribe()
	response = ownerRequest(t, handler, owner, http.MethodPost, "/api/owner/submissions/today/score", "today", map[string]any{"score": 900})
	if response.Code != http.StatusOK {
		t.Fatalf("correction returned %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-events:
	default:
		t.Fatal("correction did not publish leaderboard event")
	}

	dashboard := request(t, handler, http.MethodGet, "/api/dashboard", nil, player)
	var result dashboardResponse
	json.NewDecoder(dashboard.Body).Decode(&result)
	if result.Players[0].CombinedScore != 100 || result.Players[0].Scores[0].RawScore != 900 {
		t.Fatalf("corrected dashboard score: %+v", result.Players[0])
	}
	if screenshot := screenshotRequest(handler, player, "today"); screenshot.Code != http.StatusOK {
		t.Fatalf("correction removed screenshot: %d", screenshot.Code)
	}

	for _, tc := range []struct {
		id    string
		score int
		want  int
	}{
		{"today", 901, http.StatusUnprocessableEntity},
		{"past", 400, http.StatusNotFound},
	} {
		if response := ownerRequest(t, handler, owner, http.MethodPost, "/api/owner/submissions/"+tc.id+"/score", tc.id, map[string]any{"score": tc.score}); response.Code != tc.want {
			t.Fatalf("correction %s returned %d, want %d", tc.id, response.Code, tc.want)
		}
	}
}

func TestOwnerReopensTodaysSubmissionOnly(t *testing.T) {
	handler, cfg, broker := ownerTestHandler(t)
	owner, _ := login(t, handler, cfg.OwnerToken)
	player, _ := login(t, handler, cfg.Players[0].Token)

	if response := ownerRequest(t, handler, player, http.MethodDelete, "/api/owner/submissions/today", "today", nil); response.Code != http.StatusForbidden {
		t.Fatalf("player reopen returned %d", response.Code)
	}
	if response := ownerRequest(t, handler, owner, http.MethodDelete, "/api/owner/submissions/past", "past", nil); response.Code != http.StatusNotFound {
		t.Fatalf("past reopen returned %d", response.Code)
	}
	events, unsubscribe := broker.subscribe()
	defer unsubscribe()
	if response := ownerRequest(t, handler, owner, http.MethodDelete, "/api/owner/submissions/today", "today", nil); response.Code != http.StatusNoContent {
		t.Fatalf("reopen returned %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-events:
	default:
		t.Fatal("reopen did not publish leaderboard event")
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "screenshots", "today.png")); !os.IsNotExist(err) {
		t.Fatalf("reopened screenshot remains: %v", err)
	}
	if screenshot := screenshotRequest(handler, player, "today"); screenshot.Code != http.StatusNotFound {
		t.Fatalf("reopened screenshot returned %d", screenshot.Code)
	}
}
