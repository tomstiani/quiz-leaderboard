package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := loadConfig(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return err
	}

	db, err := openDatabase(cfg.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/api/", http.NotFoundHandler())
	mux.Handle("/", spaHandler(cfg.WebDir))

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("listening on %s", cfg.Address)
	return server.ListenAndServe()
}

func spaHandler(directory string) http.Handler {
	files := http.Dir(directory)
	server := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		file, err := files.Open(path)
		if err == nil {
			info, statErr := file.Stat()
			file.Close()
			if statErr == nil && !info.IsDir() {
				server.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/"
		server.ServeHTTP(w, r)
	})
}
