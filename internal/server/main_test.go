package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRejectsDuplicateTokens(t *testing.T) {
	cfg := config{
		SessionSecret: "12345678901234567890123456789012",
		OwnerToken:    "1234567890123456",
		Players:       []playerConfig{{ID: "alice", Name: "Alice", Token: "1234567890123456"}},
		Games:         []gameConfig{{ID: "krillion", Name: "Krillion", URL: "https://krillion.io/", MaxScore: 700}},
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected duplicate token validation error")
	}
}

func TestConfigRejectsPlaceholderSecrets(t *testing.T) {
	cfg := testConfig()
	cfg.OwnerToken = "replace-owner-token"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected placeholder secret validation error")
	}
}

func TestConfigRequiresCompleteWebPushSettings(t *testing.T) {
	cfg := testConfig()
	cfg.WebPush.PublicKey = "public"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected incomplete webPush validation error")
	}
	cfg.WebPush = webPushConfig{PublicKey: "public", PrivateKey: "private", Subject: "not-a-uri"}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected invalid webPush subject error")
	}
	cfg.WebPush.Subject = "mailto:test@example.com"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected invalid VAPID key error")
	}
}

func TestDeploymentRequiresSecureCookies(t *testing.T) {
	cfg := testConfig()
	cfg.SecureCookies = false
	data, _ := json.Marshal(cfg)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REQUIRE_SECURE_COOKIES", "true")
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected secure cookie deployment error")
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
	if count != 8 {
		t.Fatalf("got %d migrations, want 8", count)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE checksum IS NOT NULL").Scan(&count); err != nil || count != 8 {
		t.Fatalf("got %d migration checksums, want 8: %v", count, err)
	}
}
