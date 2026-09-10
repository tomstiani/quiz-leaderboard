package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

type ownerSubmission struct {
	ID              string  `json:"id"`
	PlayerID        string  `json:"playerId"`
	PlayerName      string  `json:"playerName"`
	GameID          string  `json:"gameId"`
	GameName        string  `json:"gameName"`
	RawScore        int     `json:"rawScore"`
	NormalizedScore float64 `json:"normalizedScore"`
	ScreenshotURL   string  `json:"screenshotUrl"`
}

func listOwnerSubmissions(w http.ResponseWriter, _ *http.Request, cfg config, db *sql.DB, user principal, day string) {
	if user.Role != "owner" {
		http.Error(w, "owner only", http.StatusForbidden)
		return
	}
	rows, err := db.Query(`SELECT id, player_id, game_id, raw_score, normalized_score FROM submissions
		WHERE game_day = ? AND status = 'confirmed' ORDER BY confirmed_at`, day)
	if err != nil {
		http.Error(w, "could not load submissions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	result := []ownerSubmission{}
	for rows.Next() {
		var submission ownerSubmission
		if err := rows.Scan(&submission.ID, &submission.PlayerID, &submission.GameID, &submission.RawScore, &submission.NormalizedScore); err != nil {
			http.Error(w, "could not load submissions", http.StatusInternalServerError)
			return
		}
		for _, player := range cfg.Players {
			if player.ID == submission.PlayerID {
				submission.PlayerName = player.Name
			}
		}
		if game, ok := findGame(cfg, submission.GameID); ok {
			submission.GameName = game.Name
		}
		submission.ScreenshotURL = "/api/screenshots/" + submission.ID
		result = append(result, submission)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "could not load submissions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func correctOwnerScore(w http.ResponseWriter, r *http.Request, cfg config, db *sql.DB, user principal, day string) bool {
	if user.Role != "owner" {
		http.Error(w, "owner only", http.StatusForbidden)
		return false
	}
	var request struct {
		Score *int `json:"score"`
	}
	if readJSON(w, r, &request) != nil || request.Score == nil {
		http.Error(w, "score is required", http.StatusBadRequest)
		return false
	}
	var gameID string
	err := db.QueryRow(`SELECT game_id FROM submissions WHERE id = ? AND game_day = ? AND status = 'confirmed'`, r.PathValue("id"), day).Scan(&gameID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return false
	}
	if err != nil {
		http.Error(w, "could not read submission", http.StatusInternalServerError)
		return false
	}
	game, ok := findGame(cfg, gameID)
	if !ok || *request.Score < 0 || float64(*request.Score) > game.MaxScore {
		http.Error(w, "score is outside the valid range", http.StatusUnprocessableEntity)
		return false
	}
	normalized := normalizeScore(*request.Score, game.MaxScore)
	if _, err := db.Exec("UPDATE submissions SET raw_score = ?, normalized_score = ? WHERE id = ?", *request.Score, normalized, r.PathValue("id")); err != nil {
		http.Error(w, "could not correct submission", http.StatusInternalServerError)
		return false
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "score": *request.Score, "normalizedScore": normalized})
	return true
}

func reopenOwnerSubmission(w http.ResponseWriter, r *http.Request, cfg config, db *sql.DB, user principal, day string) bool {
	if user.Role != "owner" {
		http.Error(w, "owner only", http.StatusForbidden)
		return false
	}
	var filename string
	err := db.QueryRow("DELETE FROM submissions WHERE id = ? AND game_day = ? RETURNING filename", r.PathValue("id"), day).Scan(&filename)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return false
	}
	if err != nil {
		http.Error(w, "could not reopen submission", http.StatusInternalServerError)
		return false
	}
	_ = os.Remove(filepath.Join(cfg.DataDir, "screenshots", filepath.Base(filename)))
	w.WriteHeader(http.StatusNoContent)
	return true
}
