package main

import (
	"net/http/httptest"
	"testing"
)

func TestSessionsAreCappedPerPrincipal(t *testing.T) {
	cfg := testConfig()
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	user := principal{Role: "player", PlayerID: cfg.Players[0].ID, Name: cfg.Players[0].Name}
	for range 12 {
		if err := setSessionCookie(httptest.NewRecorder(), cfg, db, user, cfg.Players[0].Token); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE player_id = ?", user.PlayerID).Scan(&count); err != nil || count != 10 {
		t.Fatalf("stored %d sessions, want 10: %v", count, err)
	}
}
