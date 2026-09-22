// internal/adapter/gemini/parser.go
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/genai"

	"whatsup-bot/internal/port"
)

type Parser struct {
	client *genai.Client
	model  string
}

func NewParser(client *genai.Client) *Parser {
	return &Parser{client: client, model: "gemini-2.5-flash"}
}

const systemInstruction = `...` // unchanged from before

type geminiResponse struct {
	Valid    bool   `json:"valid"`
	Type     string `json:"type"`
	Amount   int64  `json:"amount"`
	Category string `json:"category"`
}

func (p *Parser) Parse(ctx context.Context, rawText string) (*port.ParsedMessage, error) {
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
	}

	result, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(rawText), config)
	if err != nil {
		slog.Error("gemini request failed", "error", err)
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}

	if result.UsageMetadata != nil {
		slog.Info("gemini token usage",
			"prompt_tokens", result.UsageMetadata.PromptTokenCount,
			"response_tokens", result.UsageMetadata.CandidatesTokenCount,
			"total_tokens", result.UsageMetadata.TotalTokenCount,
		)
	}

	raw := strings.TrimSpace(result.Text())
	var gr geminiResponse
	if err := json.Unmarshal([]byte(raw), &gr); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed message", "valid", gr.Valid, "type", gr.Type, "amount", gr.Amount, "category", gr.Category)

	return &port.ParsedMessage{
		Valid:    gr.Valid,
		Type:     gr.Type,
		Amount:   gr.Amount,
		Category: gr.Category,
	}, nil
}
