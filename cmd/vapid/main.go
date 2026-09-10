package main

import (
	"encoding/json"
	"log"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]string{"publicKey": publicKey, "privateKey": privateKey}); err != nil {
		log.Fatal(err)
	}
}
