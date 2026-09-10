package main

import (
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

	handler, err := newHandler(cfg, db, nil)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      75 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("listening on %s", cfg.Address)
	return server.ListenAndServe()
}

func spaHandler(directory string) http.Handler {
	files := http.Dir(directory)
	server := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := files.Open(r.URL.Path)
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
