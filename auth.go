package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const sessionCookie = "dailygame_session"

type principal struct {
	Role     string `json:"role"`
	PlayerID string `json:"playerId,omitempty"`
	Name     string `json:"name"`
}

type sessionPayload struct {
	Role       string `json:"role"`
	PlayerID   string `json:"playerId,omitempty"`
	Credential string `json:"credential"`
}

func findPrincipal(cfg config, token string) (principal, string, bool) {
	ownerMatch := tokenEqual(token, cfg.OwnerToken)
	matched := principal{}
	credential := ""
	playerMatch := false
	for _, player := range cfg.Players {
		if tokenEqual(token, player.Token) {
			matched = principal{Role: "player", PlayerID: player.ID, Name: player.Name}
			credential = player.Token
			playerMatch = true
		}
	}
	if ownerMatch {
		return principal{Role: "owner", Name: "Owner"}, cfg.OwnerToken, true
	}
	return matched, credential, playerMatch
}

func tokenEqual(got, want string) bool {
	gotHash := sha256.Sum256([]byte(got))
	wantHash := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(gotHash[:], wantHash[:]) == 1
}

func setSessionCookie(w http.ResponseWriter, cfg config, user principal, credential string) {
	payload := sessionPayload{Role: user.Role, PlayerID: user.PlayerID, Credential: credentialDigest(credential)}
	data, _ := json.Marshal(payload)
	body := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	mac.Write([]byte(body))
	value := body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   10 * 365 * 24 * 60 * 60,
		Expires:  time.Now().AddDate(10, 0, 0),
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func authenticate(cfg config, r *http.Request) (principal, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return principal{}, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return principal{}, false
	}
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	mac.Write([]byte(parts[0]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return principal{}, false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return principal{}, false
	}
	var payload sessionPayload
	if json.Unmarshal(data, &payload) != nil {
		return principal{}, false
	}
	if payload.Role == "owner" && tokenEqual(payload.Credential, credentialDigest(cfg.OwnerToken)) {
		return principal{Role: "owner", Name: "Owner"}, true
	}
	if payload.Role == "player" {
		for _, player := range cfg.Players {
			if player.ID == payload.PlayerID && tokenEqual(payload.Credential, credentialDigest(player.Token)) {
				return principal{Role: "player", PlayerID: player.ID, Name: player.Name}, true
			}
		}
	}
	return principal{}, false
}

func credentialDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
