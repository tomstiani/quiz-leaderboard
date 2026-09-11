package main

import (
	"errors"
	"log"
	"net/http"

	"dailygame-leaderboard/internal/server"
)

func main() {
	if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
