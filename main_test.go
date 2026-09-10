package main

import (
	"testing"
)

func TestConfigRejectsDuplicateTokens(t *testing.T) {
	cfg := config{
		SessionSecret: "12345678901234567890123456789012",
		OwnerToken:    "1234567890123456",
		Players:       []playerConfig{{ID: "alice", Name: "Alice", Token: "1234567890123456"}},
		Games:         []gameConfig{{ID: "krillion", Name: "Krillion", URL: "https://krillion.io/"}},
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected duplicate token validation error")
	}
}

func TestOpenDatabaseAppliesMigrations(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("got %d migrations, want 1", count)
	}
}
