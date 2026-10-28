package config

import (
	"errors"
	"os"
)

const (
	defaultGeminiModel = "gemini-3.5-flash-lite"
	defaultMCPAddr     = "127.0.0.1:8080"
	defaultMCPURL      = "http://" + defaultMCPAddr + "/mcp"
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
	return cfg, nil
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
