package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/genai"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type Parser struct {
	client *genai.Client
	model  string
}

func NewParser(client *genai.Client, model string) *Parser {
	return &Parser{client: client, model: model}
}

func (p *Parser) Parse(ctx context.Context, rawText string) (*port.ParsedMessage, error) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	// thinkingBudget := int32(0)
	systemInstruction := fmt.Sprintf(systemInstructionTemplate, today, today)

	slog.Debug("gemini system instruction built", "today", today, "instruction", systemInstruction)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		// ThinkingConfig: &genai.ThinkingConfig{
		// 	ThinkingBudget: &thinkingBudget,
		// },
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"valid":       {Type: genai.TypeBoolean},
				"type":        {Type: genai.TypeString, Enum: []string{"CR", "DB"}},
				"amount":      {Type: genai.TypeInteger},
				"category":    {Type: genai.TypeString},
				"description": {Type: genai.TypeString},
				"date":        {Type: genai.TypeString},
			},
			Required: []string{"valid", "type", "amount", "category", "description", "date"},
		},
	}

	var result *genai.GenerateContentResponse
	backoff := 500 * time.Millisecond
	for attempt := 1; attempt <= 3; attempt++ {
		result, err = p.client.Models.GenerateContent(ctx, p.model, genai.Text(rawText), config)
		if err == nil {
			break
		}
		if !isRetryable(err) || attempt == 3 {
			utils.LogError("gemini request failed", err, "attempt", attempt)
			code := constant.ErrInternal
			if isRetryable(err) {
				code = constant.ErrServiceUnavailable
			}
			return nil, utils.WrapStd(code, "gemini request failed", err)
		}
		utils.LogWarn("gemini request failed, retrying", err, "attempt", attempt)
		time.Sleep(backoff)
		backoff *= 2
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
		utils.LogError("failed to parse gemini response", err, "raw", raw)
		return nil, utils.WrapStd(constant.ErrParseFailed, fmt.Sprintf("failed to parse gemini response %q", raw), err)
	}

	slog.Info("gemini parsed message",
		"valid", gr.Valid,
		"type", gr.Type,
		"amount", gr.Amount,
		"category", gr.Category,
		"description", gr.Description,
		"date", gr.Date,
	)

	return &port.ParsedMessage{
		Valid:       gr.Valid,
		Type:        gr.Type,
		Amount:      gr.Amount,
		Category:    gr.Category,
		Description: gr.Description,
		Date:        gr.Date,
	}, nil
}

func isRetryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") ||
		strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED")
}
