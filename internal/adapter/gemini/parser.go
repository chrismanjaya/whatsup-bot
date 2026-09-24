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
	categories := categoryEnum()
	systemInstruction := fmt.Sprintf(systemInstructionTemplate, today, strings.Join(categories, ", "), today)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"valid":       {Type: genai.TypeBoolean},
				"type":        {Type: genai.TypeString, Enum: []string{"CR", "DB"}},
				"amount":      {Type: genai.TypeInteger},
				"category":    {Type: genai.TypeString, Enum: categories},
				"description": {Type: genai.TypeString},
				"date":        {Type: genai.TypeString},
			},
			Required: []string{"valid", "type", "amount", "category", "description", "date"},
		},
	}

	raw, err := p.generate(ctx, config, rawText, "gemini request failed")
	if err != nil {
		return nil, err
	}

	var gr geminiResponse
	if err := json.Unmarshal([]byte(raw), &gr); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
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

func (p *Parser) ParseQuery(ctx context.Context, rawText string) (*port.QueryIntent, error) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	systemInstruction := fmt.Sprintf(queryInstructionTemplate, today, today, today)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"is_query":   {Type: genai.TypeBoolean},
				"start_date": {Type: genai.TypeString},
				"end_date":   {Type: genai.TypeString},
			},
			Required: []string{"is_query", "start_date", "end_date"},
		},
	}

	raw, err := p.generate(ctx, config, rawText, "gemini query request failed")
	if err != nil {
		return nil, err
	}

	var qr geminiQueryResponse
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		slog.Error("failed to parse gemini query response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini query response %q: %w", raw, err)
	}

	slog.Info("gemini parsed query", "is_query", qr.IsQuery, "start_date", qr.StartDate, "end_date", qr.EndDate)

	return &port.QueryIntent{IsQuery: qr.IsQuery, StartDate: qr.StartDate, EndDate: qr.EndDate}, nil
}

func (p *Parser) ParseSummary(ctx context.Context, rawText string) (*port.SummaryIntent, error) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	systemInstruction := fmt.Sprintf(summaryInstructionTemplate, today, today, today)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"is_summary": {Type: genai.TypeBoolean},
				"start_date": {Type: genai.TypeString},
				"end_date":   {Type: genai.TypeString},
			},
			Required: []string{"is_summary", "start_date", "end_date"},
		},
	}

	raw, err := p.generate(ctx, config, rawText, "gemini summary request failed")
	if err != nil {
		return nil, err
	}

	var sr geminiSummaryResponse
	if err := json.Unmarshal([]byte(raw), &sr); err != nil {
		slog.Error("failed to parse gemini summary response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini summary response %q: %w", raw, err)
	}

	slog.Info("gemini parsed summary", "is_summary", sr.IsSummary, "start_date", sr.StartDate, "end_date", sr.EndDate)

	return &port.SummaryIntent{IsSummary: sr.IsSummary, StartDate: sr.StartDate, EndDate: sr.EndDate}, nil
}

func (p *Parser) ParseAmend(ctx context.Context, rawText string, current *port.CurrentTransaction) (*port.AmendResult, error) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")
	categories := categoryEnum()

	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, utils.Wrap(err, "marshal current transaction failed")
	}
	systemInstruction := fmt.Sprintf(amendInstructionTemplate, today, string(currentJSON), strings.Join(categories, ", "))

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"action":      {Type: genai.TypeString, Enum: []string{"update", "delete", "none"}},
				"type":        {Type: genai.TypeString, Enum: []string{"CR", "DB"}},
				"amount":      {Type: genai.TypeInteger},
				"category":    {Type: genai.TypeString, Enum: categories},
				"description": {Type: genai.TypeString},
				"date":        {Type: genai.TypeString},
			},
			Required: []string{"action", "type", "amount", "category", "description", "date"},
		},
	}

	raw, err := p.generate(ctx, config, rawText, "gemini amend request failed")
	if err != nil {
		return nil, err
	}

	var ar geminiAmendResponse
	if err := json.Unmarshal([]byte(raw), &ar); err != nil {
		slog.Error("failed to parse gemini amend response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini amend response %q: %w", raw, err)
	}

	slog.Info("gemini parsed amend",
		"action", ar.Action,
		"type", ar.Type,
		"amount", ar.Amount,
		"category", ar.Category,
		"description", ar.Description,
		"date", ar.Date,
	)

	return &port.AmendResult{
		Action:      ar.Action,
		Type:        ar.Type,
		Amount:      ar.Amount,
		Category:    ar.Category,
		Description: ar.Description,
		Date:        ar.Date,
	}, nil
}

// generate runs a JSON-constrained Gemini request with retry on transient
// errors and returns the raw trimmed response text.
func (p *Parser) generate(ctx context.Context, config *genai.GenerateContentConfig, rawText, errMsg string) (string, error) {
	var result *genai.GenerateContentResponse
	var err error
	backoff := 500 * time.Millisecond
	for attempt := 1; attempt <= 3; attempt++ {
		result, err = p.client.Models.GenerateContent(ctx, p.model, genai.Text(rawText), config)
		if err == nil {
			break
		}
		if !isRetryable(err) || attempt == 3 {
			utils.LogError(errMsg, err, "attempt", attempt)
			code := constant.ErrInternal
			if isRetryable(err) {
				code = constant.ErrServiceUnavailable
			}
			return "", utils.WrapStd(code, errMsg, err)
		}
		utils.LogWarn(errMsg+", retrying", err, "attempt", attempt)
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

	return strings.TrimSpace(result.Text()), nil
}

func isRetryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") ||
		strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED")
}
