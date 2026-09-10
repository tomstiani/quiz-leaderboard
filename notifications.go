package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func notifyCompletion(ctx context.Context, cfg config, db *sql.DB, playerID, day string) error {
	if cfg.Ntfy.URL == "" || cfg.Ntfy.Topic == "" {
		return nil
	}
	required := make(map[string]bool, len(cfg.Games))
	for _, game := range cfg.Games {
		required[game.ID] = true
	}
	rows, err := db.Query(`SELECT game_id, normalized_score FROM submissions
		WHERE player_id = ? AND game_day = ? AND status = 'confirmed'`, playerID, day)
	if err != nil {
		return err
	}
	var total float64
	completed := map[string]bool{}
	for rows.Next() {
		var gameID string
		var normalized float64
		if err := rows.Scan(&gameID, &normalized); err != nil {
			rows.Close()
			return err
		}
		if required[gameID] {
			completed[gameID] = true
			total += normalized
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(completed) != len(required) {
		return nil
	}

	result, err := db.Exec(`INSERT OR IGNORE INTO completion_notifications (player_id, game_day)
		VALUES (?, ?)`, playerID, day)
	if err != nil {
		return err
	}
	claimed, _ := result.RowsAffected()
	if claimed == 0 {
		return nil
	}

	name := playerID
	for _, player := range cfg.Players {
		if player.ID == playerID {
			name = player.Name
			break
		}
	}
	endpoint, err := url.JoinPath(cfg.Ntfy.URL, cfg.Ntfy.Topic)
	if err == nil {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(fmt.Sprintf("%s finished today's training with %.1f points.", name, total)))
		err = requestErr
		if err == nil {
			request.Header.Set("Title", "Training complete")
			request.Header.Set("Tags", "tada")
			if cfg.Ntfy.Token != "" {
				request.Header.Set("Authorization", "Bearer "+cfg.Ntfy.Token)
			}
			client := &http.Client{Timeout: 5 * time.Second}
			var response *http.Response
			response, err = client.Do(request)
			if err == nil {
				response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode >= 300 {
					err = fmt.Errorf("ntfy returned %s", response.Status)
				}
			}
		}
	}
	if err != nil {
		db.Exec("DELETE FROM completion_notifications WHERE player_id = ? AND game_day = ? AND sent_at IS NULL", playerID, day)
		return err
	}
	_, err = db.Exec("UPDATE completion_notifications SET sent_at = CURRENT_TIMESTAMP WHERE player_id = ? AND game_day = ?", playerID, day)
	return err
}
