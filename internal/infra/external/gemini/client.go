package gemini

import (
	"context"
	"whatsupbot/internal/util"

	"google.golang.org/genai"
)

type (
	Client struct {
		genaiClient *genai.Client
		config      Config
	}
	Config struct {
		ClientConfig *genai.ClientConfig
		Model        string
	}
)

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.ClientConfig.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, util.Wrap(err, "create gemini client failed")
	}

	return &Client{
		genaiClient: geminiClient,
		config:      cfg,
	}, nil
}
