package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultGeminiModel = "gemini-3.5-flash-lite"
	defaultMCPAddr     = "127.0.0.1:8080"
	defaultMCPURL      = "http://" + defaultMCPAddr + "/mcp"

	// 12s keeps record parsing at 5 requests per minute, the Gemini free-tier limit.
	defaultRecordBatchInterval = 12 * time.Second
	defaultRecordBatchMaxSize  = 20
)

// Config holds everything read from the environment at startup. In
// production the env comes from ~/whatsup-bot.env via systemd's
// EnvironmentFile= (see the deploy workflow).
type Config struct {
	GeminiAPIKey string
	GeminiModel  string
	AppDBPath    string
	WAStorePath  string
	MCPURL       string // public URL of the MCP server, sent to users who request an MCP token

	RecordBatchInterval time.Duration // how often queued record messages are parsed together
	RecordBatchMaxSize  int           // max record lines per parse request; the rest wait for the next tick
}

// Load reads the environment and applies defaults.
func Load() (Config, error) {
	cfg := Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  os.Getenv("GEMINI_MODEL"),
		AppDBPath:    "whatsup.db",
		WAStorePath:  "whatsmeow.db",
		MCPURL:       os.Getenv("MCP_URL"),
	}
	if cfg.MCPURL == "" {
		cfg.MCPURL = defaultMCPURL
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = defaultGeminiModel
	}
	if cfg.GeminiAPIKey == "" {
		return Config{}, errors.New("GEMINI_API_KEY is not set")
	}

	var err error
	if cfg.RecordBatchInterval, err = durationEnv("RECORD_BATCH_INTERVAL", defaultRecordBatchInterval); err != nil {
		return Config{}, err
	}
	if cfg.RecordBatchMaxSize, err = positiveIntEnv("RECORD_BATCH_MAX_SIZE", defaultRecordBatchMaxSize); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// durationEnv reads a Go duration (e.g. "12s") from key, or returns fallback when unset.
func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration like 12s, got %q", key, raw)
	}
	return value, nil
}

// positiveIntEnv reads a positive integer from key, or returns fallback when unset.
func positiveIntEnv(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, raw)
	}
	return value, nil
}

// MCPConfig holds everything the MCP server reads from the environment.
type MCPConfig struct {
	Addr      string
	AppDBPath string
}

// LoadMCP reads the MCP server environment and applies defaults.
func LoadMCP() (MCPConfig, error) {
	cfg := MCPConfig{
		Addr:      os.Getenv("MCP_ADDR"),
		AppDBPath: "whatsup.db",
	}
	if cfg.Addr == "" {
		cfg.Addr = defaultMCPAddr
	}
	return cfg, nil
}
