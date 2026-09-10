package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	broker := newEventBroker()
	handler, err := newHandler(cfg, db, nil, broker)
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
		MaxHeaderBytes:    16 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		broker.close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		close(shutdownDone)
	}()
	log.Printf("listening on %s", cfg.Address)
	err = server.ListenAndServe()
	stop()
	<-shutdownDone
	return err
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
