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
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type dashboardPlayer struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Completed     int    `json:"completed"`
	CombinedScore int    `json:"combinedScore"`
}

func newHandler(cfg config, db *sql.DB, now func() time.Time) (http.Handler, error) {
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}

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
		response := dashboardResponse{Date: now().In(oslo).Format(time.DateOnly), Viewer: user}
		for _, game := range cfg.Games {
			response.Games = append(response.Games, dashboardGame{ID: game.ID, Name: game.Name, URL: game.URL})
		}
		for _, player := range cfg.Players {
			response.Players = append(response.Players, dashboardPlayer{ID: player.ID, Name: player.Name})
		}
		writeJSON(w, http.StatusOK, response)
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
