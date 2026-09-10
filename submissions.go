package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const maxScreenshotSize = 10 << 20

type draftResponse struct {
	ID    string `json:"id"`
	Score *int   `json:"score"`
}

func createDraft(w http.ResponseWriter, r *http.Request, cfg config, db *sql.DB, vision visionClient, user principal, day string) {
	if user.Role != "player" {
		http.Error(w, "players only", http.StatusForbidden)
		return
	}
	game, ok := findGame(cfg, r.PathValue("gameID"))
	if !ok {
		http.Error(w, "unknown game", http.StatusNotFound)
		return
	}
	if vision.url == "" || vision.model == "" || vision.apiKey == "" {
		http.Error(w, "screenshot analysis is not configured", http.StatusServiceUnavailable)
		return
	}
	var status string
	err := db.QueryRow("SELECT status FROM submissions WHERE player_id = ? AND game_id = ? AND game_day = ?", user.PlayerID, game.ID, day).Scan(&status)
	if err == nil && status == "confirmed" {
		http.Error(w, "result already submitted", http.StatusConflict)
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "could not check submission", http.StatusInternalServerError)
		return
	}

	image, mediaType, extension, err := readScreenshot(w, r)
	if err != nil {
		var uploadErr uploadError
		if errors.As(err, &uploadErr) {
			http.Error(w, uploadErr.message, uploadErr.status)
		} else {
			http.Error(w, "invalid upload", http.StatusBadRequest)
		}
		return
	}
	result, err := vision.analyze(r.Context(), game, mediaType, image)
	if err != nil {
		http.Error(w, "screenshot analysis failed", http.StatusBadGateway)
		return
	}
	if !result.Valid {
		reason := result.Reason
		if reason == "" {
			reason = "screenshot is not a completed result page"
		}
		http.Error(w, reason, http.StatusUnprocessableEntity)
		return
	}

	cleanupDrafts(db, cfg.DataDir, time.Now().UTC().Add(-24*time.Hour))
	id, err := randomID()
	if err != nil {
		http.Error(w, "could not create submission", http.StatusInternalServerError)
		return
	}
	filename := id + extension
	path := filepath.Join(cfg.DataDir, "screenshots", filename)
	if err := os.WriteFile(path, image, 0o600); err != nil {
		http.Error(w, "could not store screenshot", http.StatusInternalServerError)
		return
	}
	oldFilename, err := replaceDraft(db, id, user.PlayerID, game.ID, day, filename, mediaType)
	if err != nil {
		os.Remove(path)
		if errors.Is(err, errConfirmedSubmission) {
			http.Error(w, "result already submitted", http.StatusConflict)
		} else {
			http.Error(w, "could not store submission", http.StatusInternalServerError)
		}
		return
	}
	if oldFilename != "" {
		os.Remove(filepath.Join(cfg.DataDir, "screenshots", oldFilename))
	}
	writeJSON(w, http.StatusCreated, draftResponse{ID: id, Score: result.Score})
}

type uploadError struct {
	status  int
	message string
}

func (err uploadError) Error() string { return err.message }

func readScreenshot(w http.ResponseWriter, r *http.Request) ([]byte, string, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScreenshotSize+(1<<20))
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", "", uploadError{http.StatusBadRequest, "expected multipart upload"}
	}
	var image []byte
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				return nil, "", "", uploadError{http.StatusRequestEntityTooLarge, "screenshot must be 10 MB or smaller"}
			}
			return nil, "", "", uploadError{http.StatusBadRequest, "invalid multipart upload"}
		}
		if part.FormName() != "screenshot" || part.FileName() == "" || image != nil {
			part.Close()
			return nil, "", "", uploadError{http.StatusBadRequest, "upload exactly one screenshot"}
		}
		image, err = io.ReadAll(io.LimitReader(part, maxScreenshotSize+1))
		part.Close()
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				return nil, "", "", uploadError{http.StatusRequestEntityTooLarge, "screenshot must be 10 MB or smaller"}
			}
			return nil, "", "", uploadError{http.StatusBadRequest, "could not read screenshot"}
		}
	}
	if len(image) == 0 {
		return nil, "", "", uploadError{http.StatusBadRequest, "screenshot is required"}
	}
	if len(image) > maxScreenshotSize {
		return nil, "", "", uploadError{http.StatusRequestEntityTooLarge, "screenshot must be 10 MB or smaller"}
	}
	mediaType := http.DetectContentType(image)
	extensions := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp"}
	extension, ok := extensions[mediaType]
	if !ok {
		return nil, "", "", uploadError{http.StatusUnsupportedMediaType, "use a PNG, JPEG, or WebP screenshot"}
	}
	return image, mediaType, extension, nil
}

var errConfirmedSubmission = errors.New("confirmed submission exists")

func replaceDraft(db *sql.DB, id, playerID, gameID, day, filename, mediaType string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var oldFilename, status string
	err = tx.QueryRow("SELECT filename, status FROM submissions WHERE player_id = ? AND game_id = ? AND game_day = ?", playerID, gameID, day).Scan(&oldFilename, &status)
	if err == nil {
		if status == "confirmed" {
			return "", errConfirmedSubmission
		}
		if _, err := tx.Exec("DELETE FROM submissions WHERE player_id = ? AND game_id = ? AND game_day = ?", playerID, gameID, day); err != nil {
			return "", err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	_, err = tx.Exec(`INSERT INTO submissions (id, player_id, game_id, game_day, status, filename, media_type)
		VALUES (?, ?, ?, ?, 'draft', ?, ?)`, id, playerID, gameID, day, filename, mediaType)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return oldFilename, nil
}

func confirmDraft(w http.ResponseWriter, r *http.Request, cfg config, db *sql.DB, user principal, day string) bool {
	if user.Role != "player" {
		http.Error(w, "players only", http.StatusForbidden)
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
	err := db.QueryRow(`SELECT game_id FROM submissions
		WHERE id = ? AND player_id = ? AND game_day = ? AND status = 'draft'`, r.PathValue("id"), user.PlayerID, day).Scan(&gameID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "draft not found", http.StatusNotFound)
		return false
	}
	if err != nil {
		http.Error(w, "could not read draft", http.StatusInternalServerError)
		return false
	}
	game, ok := findGame(cfg, gameID)
	if !ok || *request.Score < 0 || float64(*request.Score) > game.MaxScore {
		http.Error(w, "score is outside the valid range", http.StatusUnprocessableEntity)
		return false
	}
	normalized := normalizeScore(*request.Score, game.MaxScore)
	result, err := db.Exec(`UPDATE submissions SET status = 'confirmed', raw_score = ?, normalized_score = ?, confirmed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND player_id = ? AND game_day = ? AND status = 'draft'`, *request.Score, normalized, r.PathValue("id"), user.PlayerID, day)
	if err != nil {
		http.Error(w, "could not confirm submission", http.StatusInternalServerError)
		return false
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		http.Error(w, "draft not found", http.StatusNotFound)
		return false
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "score": *request.Score, "normalizedScore": normalized})
	return true
}

func serveScreenshot(w http.ResponseWriter, r *http.Request, cfg config, db *sql.DB, user principal) {
	var filename, mediaType, status, playerID string
	err := db.QueryRow("SELECT filename, media_type, status, player_id FROM submissions WHERE id = ?", r.PathValue("id")).Scan(&filename, &mediaType, &status, &playerID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not read screenshot", http.StatusInternalServerError)
		return
	}
	if status != "confirmed" && user.Role != "owner" && user.PlayerID != playerID {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, filepath.Join(cfg.DataDir, "screenshots", filepath.Base(filename)))
}

func cleanupDrafts(db *sql.DB, dataDir string, before time.Time) {
	rows, err := db.Query(`DELETE FROM submissions WHERE status = 'draft' AND created_at < ? RETURNING filename`, before.Format("2006-01-02 15:04:05"))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var filename string
		if rows.Scan(&filename) == nil {
			os.Remove(filepath.Join(dataDir, "screenshots", filepath.Base(filename)))
		}
	}
}

func findGame(cfg config, id string) (gameConfig, bool) {
	for _, game := range cfg.Games {
		if game.ID == id {
			return game, true
		}
	}
	return gameConfig{}, false
}

func randomID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(value), nil
}
