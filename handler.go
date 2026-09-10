package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

type dashboardResponse struct {
	Date    string            `json:"date"`
	Viewer  principal         `json:"viewer"`
	Games   []dashboardGame   `json:"games"`
	Players []dashboardPlayer `json:"players"`
}

type dashboardGame struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	MaxScore float64 `json:"maxScore"`
}

type dashboardScore struct {
	GameID          string  `json:"gameId"`
	RawScore        int     `json:"rawScore"`
	NormalizedScore float64 `json:"normalizedScore"`
	ScreenshotURL   string  `json:"screenshotUrl"`
}

type dashboardPlayer struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Rank          int              `json:"rank"`
	Completed     int              `json:"completed"`
	CombinedScore float64          `json:"combinedScore"`
	Scores        []dashboardScore `json:"scores"`
}

func newHandler(cfg config, db *sql.DB, now func() time.Time, providedBroker ...*eventBroker) (http.Handler, error) {
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	vision := newVisionClient(cfg.Vision)
	broker := newEventBroker()
	if len(providedBroker) > 0 {
		broker = providedBroker[0]
	}
	if err := backfillNormalizedScores(db, cfg); err != nil {
		return nil, err
	}
	day := func() string { return now().In(oslo).Format(time.DateOnly) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Token string `json:"token"`
		}
		if err := readJSON(w, r, &request); err != nil || request.Token == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		user, credential, ok := findPrincipal(cfg, request.Token)
		if !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		setSessionCookie(w, cfg, user, credential)
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, user)
	})
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w, cfg.SecureCookies)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/session", authenticated(cfg, func(w http.ResponseWriter, _ *http.Request, user principal) {
		writeJSON(w, http.StatusOK, user)
	}))
	mux.HandleFunc("GET /api/dashboard", authenticated(cfg, func(w http.ResponseWriter, _ *http.Request, user principal) {
		response := dashboardResponse{Date: day(), Viewer: user}
		for _, game := range cfg.Games {
			response.Games = append(response.Games, dashboardGame{ID: game.ID, Name: game.Name, URL: game.URL, MaxScore: game.MaxScore})
		}
		playerIndexes := map[string]int{}
		for _, player := range cfg.Players {
			playerIndexes[player.ID] = len(response.Players)
			response.Players = append(response.Players, dashboardPlayer{ID: player.ID, Name: player.Name, Scores: []dashboardScore{}})
		}
		rows, err := db.Query(`SELECT id, player_id, game_id, raw_score, normalized_score FROM submissions
			WHERE game_day = ? AND status = 'confirmed'`, day())
		if err != nil {
			http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, playerID, gameID string
			var rawScore int
			var normalizedScore sql.NullFloat64
			if err := rows.Scan(&id, &playerID, &gameID, &rawScore, &normalizedScore); err != nil {
				http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
				return
			}
			if index, ok := playerIndexes[playerID]; ok && normalizedScore.Valid {
				response.Players[index].Scores = append(response.Players[index].Scores, dashboardScore{GameID: gameID, RawScore: rawScore, NormalizedScore: normalizedScore.Float64, ScreenshotURL: "/api/screenshots/" + id})
				response.Players[index].Completed++
				response.Players[index].CombinedScore += normalizedScore.Float64
			}
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
			return
		}
		rankPlayers(response.Players)
		writeJSON(w, http.StatusOK, response)
	}))
	mux.HandleFunc("POST /api/games/{gameID}/draft", authenticated(cfg, func(w http.ResponseWriter, r *http.Request, user principal) {
		createDraft(w, r, cfg, db, vision, user, day())
	}))
	mux.HandleFunc("POST /api/drafts/{id}/confirm", authenticated(cfg, func(w http.ResponseWriter, r *http.Request, user principal) {
		if confirmDraft(w, r, cfg, db, user, day()) {
			broker.publish()
		}
	}))
	mux.HandleFunc("GET /api/events", authenticated(cfg, func(w http.ResponseWriter, r *http.Request, _ principal) {
		serveEvents(w, r, broker)
	}))
	mux.HandleFunc("GET /api/screenshots/{id}", authenticated(cfg, func(w http.ResponseWriter, r *http.Request, user principal) {
		serveScreenshot(w, r, cfg, db, user)
	}))
	mux.Handle("/api/", http.NotFoundHandler())
	mux.Handle("/", spaHandler(cfg.WebDir))
	return mux, nil
}

func authenticated(cfg config, next func(http.ResponseWriter, *http.Request, principal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authenticate(cfg, r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next(w, r, user)
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
