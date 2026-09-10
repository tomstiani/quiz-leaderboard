package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
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
	ID string `json:"id"`
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

func setSessionCookie(w http.ResponseWriter, cfg config, db *sql.DB, user principal, credential string) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO sessions (id, role, player_id, credential) VALUES (?, ?, ?, ?)", id, user.Role, user.PlayerID, credentialDigest(credential)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE role = ? AND player_id = ? AND id NOT IN
		(SELECT id FROM sessions WHERE role = ? AND player_id = ? ORDER BY created_at DESC, rowid DESC LIMIT 10)`, user.Role, user.PlayerID, user.Role, user.PlayerID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	data, _ := json.Marshal(sessionPayload{ID: id})
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
	return nil
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

func authenticate(cfg config, db *sql.DB, r *http.Request) (principal, bool) {
	id, ok := sessionID(cfg, r)
	if !ok {
		return principal{}, false
	}
	var role, playerID, credential string
	if err := db.QueryRow("SELECT role, player_id, credential FROM sessions WHERE id = ?", id).Scan(&role, &playerID, &credential); err != nil {
		return principal{}, false
	}
	if role == "owner" && tokenEqual(credential, credentialDigest(cfg.OwnerToken)) {
		return principal{Role: "owner", Name: "Owner"}, true
	}
	if role == "player" {
		for _, player := range cfg.Players {
			if player.ID == playerID && tokenEqual(credential, credentialDigest(player.Token)) {
				return principal{Role: "player", PlayerID: player.ID, Name: player.Name}, true
			}
		}
	}
	return principal{}, false
}

func revokeSession(cfg config, db *sql.DB, r *http.Request) {
	if id, ok := sessionID(cfg, r); ok {
		db.Exec("DELETE FROM sessions WHERE id = ?", id)
	}
}

func sessionID(cfg config, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	mac.Write([]byte(parts[0]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return "", false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	var payload sessionPayload
	if json.Unmarshal(data, &payload) != nil || payload.ID == "" {
		return "", false
	}
	return payload.ID, true
}

func credentialDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
