package config

import (
	"errors"
	"os"
)

const defaultGeminiModel = "gemini-3.6-flash"

// Config holds everything read from the environment at startup. In
// production the env comes from ~/whatsup-bot.env via systemd's
// EnvironmentFile= (see the deploy workflow).
type Config struct {
	GeminiAPIKey string
	GeminiModel  string
	AppDBPath    string
	WAStorePath  string
}

// Load reads the environment and applies defaults.
func Load() (Config, error) {
	cfg := Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  os.Getenv("GEMINI_MODEL"),
		AppDBPath:    "whatsup.db",
		WAStorePath:  "whatsmeow.db",
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = defaultGeminiModel
	}
	if cfg.GeminiAPIKey == "" {
		return Config{}, errors.New("GEMINI_API_KEY is not set")
	}
	return cfg, nil
}
