package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func notifyCompletion(ctx context.Context, cfg config, db *sql.DB, playerID, day string) error {
	total, complete, err := completionScore(db, cfg, playerID, day)
	if err != nil || !complete {
		return err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO completion_notifications (player_id, game_day) VALUES (?, ?)`, playerID, day); err != nil {
		return err
	}
	name := playerID
	for _, player := range cfg.Players {
		if player.ID == playerID {
			name = player.Name
			break
		}
	}
	message := fmt.Sprintf("%s finished today's training with %.1f points.", name, total)
	var deliveryErrors []error

	if cfg.Ntfy.URL != "" && cfg.Ntfy.Topic != "" {
		claimed, err := claimNotification(db, "sent_at", playerID, day)
		if err != nil {
			deliveryErrors = append(deliveryErrors, err)
		} else if claimed {
			if err := sendNtfy(ctx, cfg.Ntfy, message); err != nil {
				releaseNotification(db, "sent_at", playerID, day)
				deliveryErrors = append(deliveryErrors, err)
			} else if err := completeNotification(db, "sent_at", playerID, day); err != nil {
				deliveryErrors = append(deliveryErrors, err)
			}
		}
	}

	if cfg.WebPush.enabled() {
		claimed, err := claimNotification(db, "push_sent_at", playerID, day)
		if err != nil {
			deliveryErrors = append(deliveryErrors, err)
		} else if claimed {
			pushCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			deliveryErr := sendBrowserPush(pushCtx, cfg, db, "Training complete", message, "", nil)
			cancel()
			if err := completeNotification(db, "push_sent_at", playerID, day); err != nil {
				deliveryErrors = append(deliveryErrors, err)
			}
			if deliveryErr != nil {
				deliveryErrors = append(deliveryErrors, deliveryErr)
			}
		}
	}
	return errors.Join(deliveryErrors...)
}

func completionScore(db *sql.DB, cfg config, playerID, day string) (float64, bool, error) {
	required := make(map[string]bool, len(cfg.Games))
	for _, game := range cfg.Games {
		required[game.ID] = true
	}
	rows, err := db.Query(`SELECT game_id, normalized_score FROM submissions
		WHERE player_id = ? AND game_day = ? AND status = 'confirmed'`, playerID, day)
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()
	var total float64
	completed := map[string]bool{}
	for rows.Next() {
		var gameID string
		var normalized float64
		if err := rows.Scan(&gameID, &normalized); err != nil {
			return 0, false, err
		}
		if required[gameID] {
			completed[gameID] = true
			total += normalized
		}
	}
	return total, len(completed) == len(required), rows.Err()
}

func claimNotification(db *sql.DB, column, playerID, day string) (bool, error) {
	result, err := db.Exec("UPDATE completion_notifications SET "+column+" = 'pending' WHERE player_id = ? AND game_day = ? AND "+column+" IS NULL", playerID, day)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func releaseNotification(db *sql.DB, column, playerID, day string) {
	db.Exec("UPDATE completion_notifications SET "+column+" = NULL WHERE player_id = ? AND game_day = ? AND "+column+" = 'pending'", playerID, day)
}

func completeNotification(db *sql.DB, column, playerID, day string) error {
	_, err := db.Exec("UPDATE completion_notifications SET "+column+" = CURRENT_TIMESTAMP WHERE player_id = ? AND game_day = ?", playerID, day)
	return err
}

func sendNtfy(ctx context.Context, cfg serviceConfig, message string) error {
	endpoint, err := url.JoinPath(cfg.URL, cfg.Topic)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(message))
	if err != nil {
		return err
	}
	request.Header.Set("Title", "Training complete")
	request.Header.Set("Tags", "tada")
	if cfg.Token != "" {
		request.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	response, err := (&http.Client{Timeout: 5 * time.Second}).Do(request)
	if err != nil {
		return errors.New("ntfy request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned %s", response.Status)
	}
	return nil
}

func sendDailyReminders(ctx context.Context, cfg config, db *sql.DB, day string, client webpush.HTTPClient) error {
	var deliveryErrors []error
	for _, player := range cfg.Players {
		_, complete, err := completionScore(db, cfg, player.ID, day)
		if err != nil {
			deliveryErrors = append(deliveryErrors, err)
			continue
		}
		var subscribed bool
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM push_subscriptions WHERE subscriber_id = ?)", player.ID).Scan(&subscribed); err != nil {
			deliveryErrors = append(deliveryErrors, err)
			continue
		}
		if complete || !subscribed {
			continue
		}
		if _, err := db.Exec("INSERT OR IGNORE INTO reminder_notifications (player_id, game_day) VALUES (?, ?)", player.ID, day); err != nil {
			deliveryErrors = append(deliveryErrors, err)
			continue
		}
		result, err := db.Exec("UPDATE reminder_notifications SET sent_at = 'pending' WHERE player_id = ? AND game_day = ? AND sent_at IS NULL", player.ID, day)
		if err != nil {
			deliveryErrors = append(deliveryErrors, err)
			continue
		}
		claimed, _ := result.RowsAffected()
		if claimed != 1 {
			continue
		}
		message := fmt.Sprintf("%s, your daily games are waiting.", player.Name)
		deliveryErr := sendBrowserPush(ctx, cfg, db, "Friendly reminder", message, player.ID, client)
		if _, err := db.Exec("UPDATE reminder_notifications SET sent_at = CURRENT_TIMESTAMP WHERE player_id = ? AND game_day = ?", player.ID, day); err != nil {
			deliveryErrors = append(deliveryErrors, err)
		}
		if deliveryErr != nil {
			deliveryErrors = append(deliveryErrors, deliveryErr)
		}
	}
	return errors.Join(deliveryErrors...)
}

func sendBrowserPush(ctx context.Context, cfg config, db *sql.DB, title, message, subscriberID string, client webpush.HTTPClient) error {
	if client == nil {
		client = publicHTTPClient()
	}
	query := "SELECT endpoint, p256dh, auth, subscriber_id FROM push_subscriptions"
	var args []any
	if subscriberID != "" {
		query += " WHERE subscriber_id = ?"
		args = append(args, subscriberID)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}
	var subscriptions []struct {
		endpoint, p256dh, auth, subscriberID string
	}
	for rows.Next() {
		var subscription struct {
			endpoint, p256dh, auth, subscriberID string
		}
		if err := rows.Scan(&subscription.endpoint, &subscription.p256dh, &subscription.auth, &subscription.subscriberID); err != nil {
			rows.Close()
			return err
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	valid := map[string]bool{"owner": true}
	for _, player := range cfg.Players {
		valid[player.ID] = true
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": message, "url": "/"})
	var deliveryErrors []error
	for _, subscription := range subscriptions {
		if !valid[subscription.subscriberID] {
			db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", subscription.endpoint)
			continue
		}
		response, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: subscription.endpoint,
			Keys:     webpush.Keys{P256dh: subscription.p256dh, Auth: subscription.auth},
		}, &webpush.Options{
			HTTPClient: client, Subscriber: cfg.WebPush.Subject, VAPIDPublicKey: cfg.WebPush.PublicKey,
			VAPIDPrivateKey: cfg.WebPush.PrivateKey, TTL: 86400,
		})
		if err != nil {
			deliveryErrors = append(deliveryErrors, errors.New("web push request failed"))
			continue
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
			db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", subscription.endpoint)
		} else if response.StatusCode < 200 || response.StatusCode >= 300 {
			deliveryErrors = append(deliveryErrors, fmt.Errorf("web push returned %s", response.Status))
		}
	}
	return errors.Join(deliveryErrors...)
}
