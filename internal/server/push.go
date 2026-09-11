package server

import (
	"crypto/ecdh"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/url"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type pushSubscriptionRequest struct {
	Endpoint string       `json:"endpoint"`
	Keys     webpush.Keys `json:"keys"`
}

func pushConfigResponse(w http.ResponseWriter, cfg config) {
	writeJSON(w, http.StatusOK, map[string]any{"enabled": cfg.WebPush.enabled(), "publicKey": cfg.WebPush.PublicKey})
}

func savePushSubscription(w http.ResponseWriter, r *http.Request, db *sql.DB, user principal) {
	var subscription pushSubscriptionRequest
	if readJSON(w, r, &subscription) != nil {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	endpoint, err := url.ParseRequestURI(subscription.Endpoint)
	p256dh, keyErr := base64.RawURLEncoding.DecodeString(subscription.Keys.P256dh)
	auth, authErr := base64.RawURLEncoding.DecodeString(subscription.Keys.Auth)
	_, publicKeyErr := ecdh.P256().NewPublicKey(p256dh)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || len(subscription.Endpoint) > 2048 || keyErr != nil || publicKeyErr != nil || authErr != nil || len(auth) != 16 {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	subscriberID := user.PlayerID
	if user.Role == "owner" {
		subscriberID = "owner"
	}
	tx, err := db.Begin()
	if err == nil {
		defer tx.Rollback()
		_, err = tx.Exec(`INSERT INTO push_subscriptions (endpoint, p256dh, auth, subscriber_id)
			VALUES (?, ?, ?, ?) ON CONFLICT(endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth, subscriber_id = excluded.subscriber_id, created_at = CURRENT_TIMESTAMP`,
			subscription.Endpoint, subscription.Keys.P256dh, subscription.Keys.Auth, subscriberID)
	}
	if err == nil {
		_, err = tx.Exec(`DELETE FROM push_subscriptions WHERE subscriber_id = ? AND endpoint NOT IN
			(SELECT endpoint FROM push_subscriptions WHERE subscriber_id = ? ORDER BY created_at DESC, rowid DESC LIMIT 5)`, subscriberID, subscriberID)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		http.Error(w, "could not save subscription", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deletePushSubscription(w http.ResponseWriter, r *http.Request, db *sql.DB, user principal) {
	var subscription struct {
		Endpoint string `json:"endpoint"`
	}
	if readJSON(w, r, &subscription) != nil || subscription.Endpoint == "" {
		http.Error(w, "invalid subscription", http.StatusBadRequest)
		return
	}
	subscriberID := user.PlayerID
	if user.Role == "owner" {
		subscriberID = "owner"
	}
	if _, err := db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ? AND subscriber_id = ?", subscription.Endpoint, subscriberID); err != nil {
		http.Error(w, "could not remove subscription", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
