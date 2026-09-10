package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type config struct {
	Address       string         `json:"address"`
	DataDir       string         `json:"dataDir"`
	WebDir        string         `json:"webDir"`
	SessionSecret string         `json:"sessionSecret"`
	SecureCookies bool           `json:"secureCookies"`
	OwnerToken    string         `json:"ownerToken"`
	Players       []playerConfig `json:"players"`
	Games         []gameConfig   `json:"games"`
	Vision        serviceConfig  `json:"vision"`
	Ntfy          serviceConfig  `json:"ntfy"`
	WebPush       webPushConfig  `json:"webPush"`
}

type webPushConfig struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	Subject    string `json:"subject"`
}

func (cfg webPushConfig) enabled() bool {
	return cfg.PublicKey != "" && cfg.PrivateKey != "" && cfg.Subject != ""
}

type playerConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type gameConfig struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	MaxScore float64 `json:"maxScore"`
}

type serviceConfig struct {
	URL    string `json:"url"`
	Token  string `json:"token"`
	Topic  string `json:"topic,omitempty"`
	Model  string `json:"model,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
}

func loadConfig(path string) (config, error) {
	if path == "" {
		path = "config.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Address == "" {
		cfg.Address = ":8080"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}
	if cfg.WebDir == "" {
		cfg.WebDir = "web/dist"
	}
	if err := cfg.validate(); err != nil {
		return config{}, fmt.Errorf("validate config: %w", err)
	}
	if os.Getenv("REQUIRE_SECURE_COOKIES") == "true" && !cfg.SecureCookies {
		return config{}, fmt.Errorf("validate config: secureCookies must be true for deployment")
	}
	return cfg, nil
}

func (cfg config) validate() error {
	pushValues := 0
	for _, value := range []string{cfg.WebPush.PublicKey, cfg.WebPush.PrivateKey, cfg.WebPush.Subject} {
		if value != "" {
			pushValues++
		}
	}
	if pushValues != 0 && pushValues != 3 {
		return fmt.Errorf("webPush requires publicKey, privateKey, and subject")
	}
	if cfg.WebPush.Subject != "" {
		subject, err := url.ParseRequestURI(cfg.WebPush.Subject)
		if err != nil || (subject.Scheme != "mailto" && subject.Scheme != "https") {
			return fmt.Errorf("webPush subject must be a mailto or HTTPS URI")
		}
		publicKey, publicErr := base64.RawURLEncoding.DecodeString(cfg.WebPush.PublicKey)
		privateKey, privateErr := base64.RawURLEncoding.DecodeString(cfg.WebPush.PrivateKey)
		if publicErr != nil || privateErr != nil || len(publicKey) != 65 || len(privateKey) != 32 {
			return fmt.Errorf("webPush requires a valid VAPID key pair")
		}
	}
	if len(cfg.SessionSecret) < 32 || strings.HasPrefix(cfg.SessionSecret, "replace-") {
		return fmt.Errorf("sessionSecret must be at least 32 random characters")
	}
	if len(cfg.OwnerToken) < 16 || strings.HasPrefix(cfg.OwnerToken, "replace-") {
		return fmt.Errorf("ownerToken must be random and at least 16 characters")
	}
	if len(cfg.Players) == 0 {
		return fmt.Errorf("at least one player is required")
	}
	if len(cfg.Games) == 0 {
		return fmt.Errorf("at least one game is required")
	}

	ids := map[string]bool{}
	tokens := map[string]bool{cfg.OwnerToken: true}
	for _, player := range cfg.Players {
		if player.ID == "" || player.ID == "owner" || player.Name == "" || len(player.Token) < 16 || strings.HasPrefix(player.Token, "replace-") {
			return fmt.Errorf("each player requires an id, name, and random token of at least 16 characters")
		}
		if ids[player.ID] {
			return fmt.Errorf("duplicate player id %q", player.ID)
		}
		if tokens[player.Token] {
			return fmt.Errorf("tokens must be unique")
		}
		ids[player.ID] = true
		tokens[player.Token] = true
	}

	ids = map[string]bool{}
	for _, game := range cfg.Games {
		parsed, err := url.ParseRequestURI(game.URL)
		if game.ID == "" || game.Name == "" || err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("game %q requires an id, name, and HTTPS URL", game.ID)
		}
		if game.MaxScore <= 0 {
			return fmt.Errorf("game %q maxScore must be positive", game.ID)
		}
		if ids[game.ID] {
			return fmt.Errorf("duplicate game id %q", game.ID)
		}
		ids[game.ID] = true
	}
	return nil
}
