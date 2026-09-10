package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCompletionNotificationIsSentOnce(t *testing.T) {
	var calls atomic.Int32
	var message string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		message = string(body)
		if r.URL.Path != "/team-topic" || r.Header.Get("Authorization") != "Bearer ntfy-token" || r.Header.Get("Title") != "Training complete" {
			t.Errorf("unexpected ntfy request: %s %v", r.URL.Path, r.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Ntfy = serviceConfig{URL: server.URL, Topic: "team-topic", Token: "ntfy-token"}
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insertNotificationScore(t, db, "one", "alice", "geopolitix", 50)
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil || calls.Load() != 0 {
		t.Fatalf("incomplete notification: calls=%d err=%v", calls.Load(), err)
	}
	insertNotificationScore(t, db, "two", "alice", "krillion", 75)
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil {
		t.Fatal(err)
	}
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM submissions WHERE id = 'two'"); err != nil {
		t.Fatal(err)
	}
	insertNotificationScore(t, db, "two-again", "alice", "krillion", 75)
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || !strings.Contains(message, "Alice finished today's training with 125.0 points.") {
		t.Fatalf("calls=%d message=%q", calls.Load(), message)
	}
	var sent bool
	if err := db.QueryRow("SELECT sent_at IS NOT NULL FROM completion_notifications WHERE player_id = 'alice'").Scan(&sent); err != nil || !sent {
		t.Fatalf("notification not recorded: sent=%v err=%v", sent, err)
	}
}

func insertNotificationScore(t *testing.T, db *sql.DB, id, player, game string, normalized float64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type, raw_score, normalized_score, confirmed_at)
		VALUES (?, ?, ?, '2026-09-10', 'confirmed', ?, 'image/png', 1, ?, CURRENT_TIMESTAMP)`, id, player, game, id+".png", normalized)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFailedNotificationCanRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	cfg := testConfig()
	cfg.Ntfy = serviceConfig{URL: server.URL, Topic: "team"}
	db, _ := openDatabase(t.TempDir())
	defer db.Close()
	insertNotificationScore(t, db, "bob-one", "bob", "geopolitix", 50)
	insertNotificationScore(t, db, "bob-two", "bob", "krillion", 50)

	if err := notifyCompletion(context.Background(), cfg, db, "bob", "2026-09-10"); err == nil {
		t.Fatal("expected ntfy failure")
	}
	if err := notifyCompletion(context.Background(), cfg, db, "bob", "2026-09-10"); err != nil || calls.Load() != 2 {
		t.Fatalf("retry calls=%d err=%v", calls.Load(), err)
	}
}

func TestNotificationFailureDoesNotRollbackFinalConfirmation(t *testing.T) {
	var ntfyCalls atomic.Int32
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ntfyCalls.Add(1)
		http.Error(w, "offline", http.StatusServiceUnavailable)
	}))
	defer ntfy.Close()
	var visionCalls atomic.Int32
	score := 300
	vision := fakeVision(t, visionResult{Valid: true, Score: &score}, &visionCalls)
	cfg := testConfig()
	cfg.DataDir = t.TempDir()
	cfg.Vision = serviceConfig{URL: vision.URL, Model: "test-model", APIKey: "test-api-key"}
	cfg.Ntfy = serviceConfig{URL: ntfy.URL, Topic: "team"}
	db, _ := openDatabase(cfg.DataDir)
	defer db.Close()
	handler, _ := newHandler(cfg, db, func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) })
	alice, _ := login(t, handler, cfg.Players[0].Token)

	for index, game := range []string{"geopolitix", "krillion"} {
		response := upload(t, handler, alice, game, "score.png", testPNG)
		var draft draftResponse
		if response.Code != http.StatusCreated || json.NewDecoder(response.Body).Decode(&draft) != nil {
			t.Fatalf("%s upload: %d %s", game, response.Code, response.Body.String())
		}
		if response := confirm(t, handler, alice, draft.ID, score); response.Code != http.StatusOK {
			t.Fatalf("%s confirmation: %d %s", game, response.Code, response.Body.String())
		}
		if got := int(ntfyCalls.Load()); got != index {
			t.Fatalf("after %s ntfy calls=%d, want %d", game, got, index)
		}
	}
	if ntfyCalls.Load() != 1 {
		t.Fatalf("ntfy calls=%d", ntfyCalls.Load())
	}
	var confirmed int
	if err := db.QueryRow("SELECT COUNT(*) FROM submissions WHERE player_id = 'alice' AND status = 'confirmed'").Scan(&confirmed); err != nil || confirmed != 2 {
		t.Fatalf("confirmed submissions=%d err=%v", confirmed, err)
	}
}
