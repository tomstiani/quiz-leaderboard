package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type config struct {
	Address       string         `json:"address"`
	DataDir       string         `json:"dataDir"`
	WebDir        string         `json:"webDir"`
	SessionSecret string         `json:"sessionSecret"`
	OwnerToken    string         `json:"ownerToken"`
	Players       []playerConfig `json:"players"`
	Games         []gameConfig   `json:"games"`
	Vision        serviceConfig  `json:"vision"`
	Ntfy          serviceConfig  `json:"ntfy"`
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
	return cfg, nil
}

func (cfg config) validate() error {
	if len(cfg.SessionSecret) < 32 {
		return fmt.Errorf("sessionSecret must be at least 32 characters")
	}
	if len(cfg.OwnerToken) < 16 {
		return fmt.Errorf("ownerToken must be at least 16 characters")
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
		if player.ID == "" || player.Name == "" || len(player.Token) < 16 {
			return fmt.Errorf("each player requires an id, name, and token of at least 16 characters")
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
		if game.MaxScore < 0 {
			return fmt.Errorf("game %q maxScore cannot be negative", game.ID)
		}
		if ids[game.ID] {
			return fmt.Errorf("duplicate game id %q", game.ID)
		}
		ids[game.ID] = true
	}
	return nil
}
