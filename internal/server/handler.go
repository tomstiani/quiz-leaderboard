package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type dashboardResponse struct {
	Date    string            `json:"date"`
	Week    []string          `json:"week"`
	Month   []string          `json:"month"`
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
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Rank          int                `json:"rank"`
	Completed     int                `json:"completed"`
	CombinedScore float64            `json:"combinedScore"`
	Scores        []dashboardScore   `json:"scores"`
	DailyTotals   map[string]float64 `json:"dailyTotals"`
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
	if err := recalculateNormalizedScores(db, cfg); err != nil {
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
		if err := setSessionCookie(w, cfg, db, user, credential); err != nil {
			http.Error(w, "could not create session", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, user)
	})
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		revokeSession(cfg, db, r)
		clearSessionCookie(w, cfg.SecureCookies)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/session", authenticated(cfg, db, func(w http.ResponseWriter, _ *http.Request, user principal) {
		writeJSON(w, http.StatusOK, user)
	}))
	mux.HandleFunc("GET /api/dashboard", authenticated(cfg, db, func(w http.ResponseWriter, _ *http.Request, user principal) {
		today := now().In(oslo)
		weekStart := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
		monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, oslo)
		response := dashboardResponse{Date: today.Format(time.DateOnly), Viewer: user}
		for offset := 0; offset < 7; offset++ {
			response.Week = append(response.Week, weekStart.AddDate(0, 0, offset).Format(time.DateOnly))
		}
		for date := monthStart; date.Month() == today.Month(); date = date.AddDate(0, 0, 1) {
			response.Month = append(response.Month, date.Format(time.DateOnly))
		}
		for _, game := range cfg.Games {
			response.Games = append(response.Games, dashboardGame{ID: game.ID, Name: game.Name, URL: game.URL, MaxScore: game.MaxScore})
		}
		playerIndexes := map[string]int{}
		for _, player := range cfg.Players {
			playerIndexes[player.ID] = len(response.Players)
			response.Players = append(response.Players, dashboardPlayer{ID: player.ID, Name: player.Name, Scores: []dashboardScore{}, DailyTotals: map[string]float64{}})
		}
		start, end := response.Month[0], response.Month[len(response.Month)-1]
		if response.Week[0] < start {
			start = response.Week[0]
		}
		if response.Week[6] > end {
			end = response.Week[6]
		}
		rows, err := db.Query(`SELECT id, player_id, game_id, game_day, raw_score, normalized_score FROM submissions
			WHERE game_day BETWEEN ? AND ? AND status = 'confirmed'`, start, end)
		if err != nil {
			http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, playerID, gameID, gameDay string
			var rawScore int
			var normalizedScore sql.NullFloat64
			if err := rows.Scan(&id, &playerID, &gameID, &gameDay, &rawScore, &normalizedScore); err != nil {
				http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
				return
			}
			if index, ok := playerIndexes[playerID]; ok && normalizedScore.Valid {
				response.Players[index].DailyTotals[gameDay] += normalizedScore.Float64
				if gameDay != response.Date {
					continue
				}
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
	mux.HandleFunc("POST /api/games/{gameID}/draft", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		createDraft(w, r, cfg, db, vision, user, day())
	}))
	mux.HandleFunc("POST /api/drafts/{id}/confirm", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		if confirmDraft(w, r, cfg, db, user, day()) {
			broker.publish()
			if err := notifyCompletion(r.Context(), cfg, db, user.PlayerID, day()); err != nil {
				log.Printf("send completion notification: %v", err)
			}
		}
	}))
	mux.HandleFunc("GET /api/events", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, _ principal) {
		serveEvents(w, r, broker)
	}))
	mux.HandleFunc("GET /api/push/config", authenticated(cfg, db, func(w http.ResponseWriter, _ *http.Request, _ principal) {
		pushConfigResponse(w, cfg)
	}))
	mux.HandleFunc("POST /api/push/subscriptions", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		savePushSubscription(w, r, db, user)
	}))
	mux.HandleFunc("DELETE /api/push/subscriptions", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		deletePushSubscription(w, r, db, user)
	}))
	mux.HandleFunc("GET /api/screenshots/{id}", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		serveScreenshot(w, r, cfg, db, user)
	}))
	mux.HandleFunc("GET /api/owner/submissions", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		listOwnerSubmissions(w, r, cfg, db, user, day())
	}))
	mux.HandleFunc("POST /api/owner/submissions/{id}/score", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		if correctOwnerScore(w, r, cfg, db, user, day()) {
			broker.publish()
		}
	}))
	mux.HandleFunc("DELETE /api/owner/submissions/{id}", authenticated(cfg, db, func(w http.ResponseWriter, r *http.Request, user principal) {
		if reopenOwnerSubmission(w, r, cfg, db, user, day()) {
			broker.publish()
		}
	}))
	mux.Handle("/api/", http.NotFoundHandler())
	mux.Handle("/", spaHandler(cfg.WebDir))
	return secureHTTP(mux), nil
}

func authenticated(cfg config, db *sql.DB, next func(http.ResponseWriter, *http.Request, principal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authenticate(cfg, db, r)
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
