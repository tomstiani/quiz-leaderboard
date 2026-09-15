package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testConfig() config {
	return config{
		SessionSecret: "12345678901234567890123456789012",
		OwnerToken:    "owner-token-1234567890",
		Players: []playerConfig{
			{ID: "alice", Name: "Alice", Token: "alice-token-1234567890"},
			{ID: "bob", Name: "Bob", Token: "bob-token-123456789012"},
		},
		Games: []gameConfig{
			{ID: "geopolitix", Name: "Geopolitix", URL: "https://geopolitix.live/", MaxScore: 900},
			{ID: "krillion", Name: "Krillion", URL: "https://krillion.io/", MaxScore: 700},
		},
	}
}

func testHandler(t *testing.T, cfg config, now time.Time) http.Handler {
	t.Helper()
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	handler, err := newHandler(cfg, db, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func request(t *testing.T, handler http.Handler, method, path string, body []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func login(t *testing.T, handler http.Handler, token string) (*http.Cookie, principal) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"token": token})
	response := request(t, handler, http.MethodPost, "/api/login", body, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login returned %d: %s", response.Code, response.Body.String())
	}
	var user principal
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	return cookies[0], user
}

func TestPlayerAuthenticationAndDashboard(t *testing.T) {
	now := time.Date(2026, 3, 28, 23, 30, 0, 0, time.UTC)
	handler := testHandler(t, testConfig(), now)

	if response := request(t, handler, http.MethodGet, "/api/dashboard", nil, nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated dashboard returned %d", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/api/events", nil, nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated event stream returned %d", response.Code)
	}
	cookie, user := login(t, handler, "alice-token-1234567890")
	if user.Role != "player" || user.PlayerID != "alice" || user.Name != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge <= 0 {
		t.Fatalf("insecure session cookie: %+v", cookie)
	}

	response := request(t, handler, http.MethodGet, "/api/dashboard", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard returned %d: %s", response.Code, response.Body.String())
	}
	var dashboard dashboardResponse
	if err := json.NewDecoder(response.Body).Decode(&dashboard); err != nil {
		t.Fatal(err)
	}
	if dashboard.Date != "2026-03-29" || len(dashboard.Week) != 7 || dashboard.Week[0] != "2026-03-23" || dashboard.Week[6] != "2026-03-29" || len(dashboard.Month) != 31 || dashboard.Month[0] != "2026-03-01" || dashboard.Month[30] != "2026-03-31" || len(dashboard.Games) != 2 || len(dashboard.Players) != 2 {
		t.Fatalf("unexpected dashboard: %+v", dashboard)
	}
	if dashboard.Games[0].URL != "https://geopolitix.live/" || dashboard.Players[0].Name != "Alice" || dashboard.Players[0].Rank != 1 || dashboard.Players[1].Rank != 1 {
		t.Fatalf("configuration order or shared rank was not preserved: %+v", dashboard)
	}

	response = request(t, handler, http.MethodPost, "/api/logout", nil, cookie)
	if response.Code != http.StatusNoContent || response.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout did not clear cookie: %+v", response.Result().Cookies())
	}
	if response := request(t, handler, http.MethodGet, "/api/session", nil, cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out cookie returned %d", response.Code)
	}
}

func TestPastDashboard(t *testing.T) {
	now := time.Date(2026, 3, 28, 23, 30, 0, 0, time.UTC)
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type, raw_score, normalized_score, confirmed_at)
		VALUES ('past', 'alice', 'geopolitix', '2026-03-28', 'confirmed', 'past.png', 'image/png', 450, 50, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	handler, err := newHandler(testConfig(), db, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	cookie, _ := login(t, handler, "alice-token-1234567890")

	response := request(t, handler, http.MethodGet, "/api/dashboard?date=2026-03-28", nil, cookie)
	var dashboard dashboardResponse
	if response.Code != http.StatusOK {
		t.Fatalf("past dashboard returned %d: %s", response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(&dashboard); err != nil {
		t.Fatal(err)
	}
	if dashboard.Date != "2026-03-28" || dashboard.Today != "2026-03-29" || dashboard.Players[0].CombinedScore != 50 || dashboard.Players[0].Scores[0].ScreenshotURL != "/api/screenshots/past" {
		t.Fatalf("unexpected past dashboard: %+v", dashboard)
	}
	for _, path := range []string{"/api/dashboard?date=invalid", "/api/dashboard?date=2026-03-30"} {
		if response := request(t, handler, http.MethodGet, path, nil, cookie); response.Code != http.StatusBadRequest {
			t.Fatalf("%s returned %d, want 400", path, response.Code)
		}
	}
}

func TestOwnerAndCurrentSession(t *testing.T) {
	cfg := testConfig()
	cfg.SecureCookies = true
	handler := testHandler(t, cfg, time.Now())
	cookie, user := login(t, handler, cfg.OwnerToken)
	if user.Role != "owner" || user.PlayerID != "" || !cookie.Secure {
		t.Fatalf("unexpected owner session: %+v, cookie: %+v", user, cookie)
	}
	response := request(t, handler, http.MethodGet, "/api/session", nil, cookie)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("session returned %d with cache control %q", response.Code, response.Header().Get("Cache-Control"))
	}
}

func TestLoginValidation(t *testing.T) {
	handler := testHandler(t, testConfig(), time.Now())
	cases := []struct {
		name string
		body []byte
		want int
	}{
		{"invalid token", []byte(`{"token":"not-a-token"}`), http.StatusUnauthorized},
		{"malformed JSON", []byte(`{"token":`), http.StatusBadRequest},
		{"unknown field", []byte(`{"token":"x","extra":true}`), http.StatusBadRequest},
		{"multiple values", []byte(`{"token":"x"}{}`), http.StatusBadRequest},
		{"oversized body", bytes.Repeat([]byte("x"), 3000), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := request(t, handler, http.MethodPost, "/api/login", tc.body, nil)
			if response.Code != tc.want {
				t.Fatalf("got %d, want %d", response.Code, tc.want)
			}
		})
	}
}

func TestRotatedAndTamperedSessionsAreRejected(t *testing.T) {
	cfg := testConfig()
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler, _ := newHandler(cfg, db, time.Now)
	cookie, _ := login(t, handler, cfg.Players[0].Token)

	tampered := *cookie
	tampered.Value += "x"
	if response := request(t, handler, http.MethodGet, "/api/session", nil, &tampered); response.Code != http.StatusUnauthorized {
		t.Fatalf("tampered cookie returned %d", response.Code)
	}

	cfg.Players[0].Token = "rotated-token-123456789"
	rotatedHandler, _ := newHandler(cfg, db, time.Now)
	if response := request(t, rotatedHandler, http.MethodGet, "/api/session", nil, cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("rotated token cookie returned %d", response.Code)
	}

	cfg = testConfig()
	cfg.Players = cfg.Players[1:]
	removedHandler, _ := newHandler(cfg, db, time.Now)
	if response := request(t, removedHandler, http.MethodGet, "/api/session", nil, cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("removed player cookie returned %d", response.Code)
	}
}

func TestHealth(t *testing.T) {
	handler := testHandler(t, testConfig(), time.Now())
	response := request(t, handler, http.MethodGet, "/api/health", nil, nil)
	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected health response: %d %q", response.Code, response.Body.String())
	}
}
