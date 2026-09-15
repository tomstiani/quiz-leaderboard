package server

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func testPushKeys(t *testing.T) (string, string, webpush.Keys) {
	t.Helper()
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	return privateKey, publicKey, webpush.Keys{
		P256dh: base64.RawURLEncoding.EncodeToString(receiver.PublicKey().Bytes()),
		Auth:   base64.RawURLEncoding.EncodeToString(auth),
	}
}

func pushTestHandler(t *testing.T) (http.Handler, *sql.DB, config) {
	t.Helper()
	privateKey, publicKey, _ := testPushKeys(t)
	cfg := testConfig()
	cfg.WebPush = webPushConfig{PrivateKey: privateKey, PublicKey: publicKey, Subject: "mailto:test@example.com"}
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	handler, err := newHandler(cfg, db, func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	return handler, db, cfg
}

func TestPushSubscriptionEndpoints(t *testing.T) {
	handler, db, cfg := pushTestHandler(t)
	alice, _ := login(t, handler, cfg.Players[0].Token)
	bob, _ := login(t, handler, cfg.Players[1].Token)
	_, _, keys := testPushKeys(t)
	body, _ := json.Marshal(pushSubscriptionRequest{Endpoint: "https://push.example/subscription", Keys: keys})

	if response := request(t, handler, http.MethodGet, "/api/push/config", nil, nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated config returned %d", response.Code)
	}
	response := request(t, handler, http.MethodGet, "/api/push/config", nil, alice)
	var pushConfig struct {
		Enabled   bool   `json:"enabled"`
		PublicKey string `json:"publicKey"`
	}
	json.NewDecoder(response.Body).Decode(&pushConfig)
	if response.Code != http.StatusOK || !pushConfig.Enabled || pushConfig.PublicKey != cfg.WebPush.PublicKey {
		t.Fatalf("push config: %d %+v", response.Code, pushConfig)
	}
	if response := request(t, handler, http.MethodPost, "/api/push/subscriptions", body, alice); response.Code != http.StatusNoContent {
		t.Fatalf("save subscription returned %d: %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodPost, "/api/push/subscriptions", []byte(`{"endpoint":"http://insecure","keys":{"p256dh":"x","auth":"y"}}`), alice); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid subscription returned %d", response.Code)
	}
	for device := 1; device <= 5; device++ {
		body, _ := json.Marshal(pushSubscriptionRequest{Endpoint: fmt.Sprintf("https://push.example/device-%d", device), Keys: keys})
		if response := request(t, handler, http.MethodPost, "/api/push/subscriptions", body, alice); response.Code != http.StatusNoContent {
			t.Fatalf("device %d returned %d", device, response.Code)
		}
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE subscriber_id = 'alice'").Scan(&count)
	if count != 5 {
		t.Fatalf("stored %d subscriptions, want 5", count)
	}

	deleteBody := []byte(`{"endpoint":"https://push.example/device-5"}`)
	if response := request(t, handler, http.MethodDelete, "/api/push/subscriptions", deleteBody, bob); response.Code != http.StatusNoContent {
		t.Fatalf("other player delete returned %d", response.Code)
	}
	db.QueryRow("SELECT COUNT(*) FROM push_subscriptions").Scan(&count)
	if count != 5 {
		t.Fatalf("other player deleted subscription")
	}
	if response := request(t, handler, http.MethodDelete, "/api/push/subscriptions", deleteBody, alice); response.Code != http.StatusNoContent {
		t.Fatalf("delete subscription returned %d", response.Code)
	}
	db.QueryRow("SELECT COUNT(*) FROM push_subscriptions").Scan(&count)
	if count != 4 {
		t.Fatalf("got %d subscriptions after delete, want 4", count)
	}
}

func TestCompletedPlayerMarksBrowserPushDelivered(t *testing.T) {
	privateKey, publicKey, _ := testPushKeys(t)
	cfg := testConfig()
	cfg.WebPush = webPushConfig{PrivateKey: privateKey, PublicKey: publicKey, Subject: "mailto:test@example.com"}
	db, _ := openDatabase(t.TempDir())
	defer db.Close()
	insertNotificationScore(t, db, "push-one", "alice", "geopolitix", 50)
	insertNotificationScore(t, db, "push-two", "alice", "krillion", 50)
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil {
		t.Fatal(err)
	}
	var sent bool
	if err := db.QueryRow("SELECT push_sent_at IS NOT NULL FROM completion_notifications WHERE player_id = 'alice'").Scan(&sent); err != nil || !sent {
		t.Fatalf("browser notification not recorded: sent=%v err=%v", sent, err)
	}
}

func TestFailedBrowserPushIsNotRepeated(t *testing.T) {
	privateKey, publicKey, keys := testPushKeys(t)
	cfg := testConfig()
	cfg.WebPush = webPushConfig{PrivateKey: privateKey, PublicKey: publicKey, Subject: "mailto:test@example.com"}
	db, _ := openDatabase(t.TempDir())
	defer db.Close()
	insertNotificationScore(t, db, "failed-one", "alice", "geopolitix", 50)
	insertNotificationScore(t, db, "failed-two", "alice", "krillion", 50)
	if _, err := db.Exec("INSERT INTO push_subscriptions (endpoint, p256dh, auth, subscriber_id) VALUES ('https://127.0.0.1/push', ?, ?, 'alice')", keys.P256dh, keys.Auth); err != nil {
		t.Fatal(err)
	}
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err == nil {
		t.Fatal("expected browser push failure")
	}
	if err := notifyCompletion(context.Background(), cfg, db, "alice", "2026-09-10"); err != nil {
		t.Fatalf("browser push was retried: %v", err)
	}
}

func TestDailyReminderTargetsIncompletePlayerOnce(t *testing.T) {
	privateKey, publicKey, keys := testPushKeys(t)
	var paths []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	cfg := testConfig()
	cfg.WebPush = webPushConfig{PrivateKey: privateKey, PublicKey: publicKey, Subject: "mailto:test@example.com"}
	db, _ := openDatabase(t.TempDir())
	defer db.Close()
	insertNotificationScore(t, db, "bob-reminder-one", "bob", "geopolitix", 50)
	insertNotificationScore(t, db, "bob-reminder-two", "bob", "krillion", 50)
	for _, subscriber := range []string{"alice", "bob"} {
		if _, err := db.Exec("INSERT INTO push_subscriptions (endpoint, p256dh, auth, subscriber_id) VALUES (?, ?, ?, ?)", server.URL+"/"+subscriber, keys.P256dh, keys.Auth, subscriber); err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		if err := sendDailyReminders(context.Background(), cfg, db, "2026-09-10", server.Client()); err != nil {
			t.Fatal(err)
		}
	}
	if len(paths) != 1 || paths[0] != "/alice" {
		t.Fatalf("reminder paths=%v, want [/alice]", paths)
	}
}

func TestBrowserPushDeliveryAndStaleCleanup(t *testing.T) {
	privateKey, publicKey, keys := testPushKeys(t)
	var calls int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "Alice") {
			t.Error("push payload was not encrypted")
		}
		if r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("Authorization") == "" {
			t.Errorf("missing web push headers: %v", r.Header)
		}
		if calls == 2 {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	cfg := testConfig()
	cfg.WebPush = webPushConfig{PrivateKey: privateKey, PublicKey: publicKey, Subject: "mailto:test@example.com"}
	db, _ := openDatabase(t.TempDir())
	defer db.Close()
	for _, endpoint := range []string{server.URL + "/one", server.URL + "/stale"} {
		if _, err := db.Exec("INSERT INTO push_subscriptions (endpoint, p256dh, auth, subscriber_id) VALUES (?, ?, ?, 'alice')", endpoint, keys.P256dh, keys.Auth); err != nil {
			t.Fatal(err)
		}
	}
	if err := sendBrowserPush(context.Background(), cfg, db, "Training complete", "Alice finished today's training with 100.0 points.", "", server.Client()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("push calls=%d", calls)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM push_subscriptions").Scan(&count)
	if count != 1 {
		t.Fatalf("stale subscription was not removed")
	}
}
