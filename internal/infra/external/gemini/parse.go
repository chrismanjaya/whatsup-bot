package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"google.golang.org/genai"
)

const (
	defaultResponseMIMEType = "application/json"

	summaryTypeAll = "ALL"

	maxAttempt        = 3
	initialRetryDelay = 500 * time.Millisecond
)

var (
	thinkingLevel = genai.ThinkingLevelMinimal
)

func (c *Client) ParseRecordMessage(ctx context.Context, rawMessage string) (*domain.ParsedMessage, error) {
	// generate system instruction
	today := util.GetTodayLocalTimeString("2006-01-02")
	categories := categoryEnum()
	systemInstruction := fmt.Sprintf(
		recordInstructionTemplate,
		today,
		strings.Join(categories, ", "),
		today,
	)

	// generate parse config
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  defaultResponseMIMEType,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: thinkingLevel,
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"valid":       {Type: genai.TypeBoolean},
				"type":        {Type: genai.TypeString, Enum: transactionTypeEnum()},
				"amount":      {Type: genai.TypeInteger},
				"category":    {Type: genai.TypeString, Enum: categories},
				"description": {Type: genai.TypeString},
				"date":        {Type: genai.TypeString},
				"quip":        {Type: genai.TypeString},
			},
			Required: []string{"valid", "type", "amount", "category", "description", "date"},
		},
	}

	// parse message
	raw, err := c.generate(ctx, config, rawMessage, "gemini parse record message failed")
	if err != nil {
		return nil, err
	}

	// mapping response
	var response geminiRecordResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed message",
		"valid", response.Valid,
		"type", response.Type,
		"amount", response.Amount,
		"category", response.Category,
		"description", response.Description,
		"date", response.Date,
		"quip", response.Quip,
	)

	return &domain.ParsedMessage{
		Valid: response.Valid,
		Quip:  response.Quip,
		TransactionMessage: &domain.ParsedTransactionMessage{
			Type:        response.Type,
			Amount:      response.Amount,
			Category:    response.Category,
			Description: response.Description,
			Date:        response.Date,
			DateLayout:  "2006-01-02",
		},
	}, nil
}

func (c *Client) ParseAmendMessage(ctx context.Context, rawMessage string, transaction *domain.Transaction) (*domain.ParsedMessage, error) {
	// parse transaction
	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return nil, util.Wrap(err, "marshal transaction failed")
	}

	// generate system instruction
	today := util.GetTodayLocalTimeString("2006-01-02")
	categories := categoryEnum()

	systemInstruction := fmt.Sprintf(
		amendInstructionTemplate,
		today,
		string(transactionJSON),
		strings.Join(categories, ", "),
	)

	// generate parse config
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  defaultResponseMIMEType,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: thinkingLevel,
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"action":      {Type: genai.TypeString, Enum: actionEnum()},
				"type":        {Type: genai.TypeString, Enum: transactionTypeEnum()},
				"amount":      {Type: genai.TypeInteger},
				"category":    {Type: genai.TypeString, Enum: categories},
				"description": {Type: genai.TypeString},
				"date":        {Type: genai.TypeString},
				"quip":        {Type: genai.TypeString},
			},
			Required: []string{"action", "type", "amount", "category", "description", "date"},
		},
	}

	// parse message
	raw, err := c.generate(ctx, config, rawMessage, "gemini parse record message failed")
	if err != nil {
		return nil, err
	}

	// mapping response
	var response geminiAmendResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed message",
		"action", response.Action,
		"type", response.Type,
		"amount", response.Amount,
		"category", response.Category,
		"description", response.Description,
		"date", response.Date,
	)

	isValid := true
	if domain.TransactionActionEnum(response.Action) == domain.TRANSACTION_ACTION_NONE {
		isValid = false
	}

	return &domain.ParsedMessage{
		Valid: isValid,
		TransactionMessage: &domain.ParsedTransactionMessage{
			Action:      domain.TransactionActionEnum(response.Action),
			Type:        response.Type,
			Amount:      response.Amount,
			Category:    response.Category,
			Description: response.Description,
			Date:        response.Date,
			DateLayout:  "2006-01-02",
		},
	}, nil
}

func (c *Client) ParseReportSummaryMessage(ctx context.Context, rawMessage string) (*domain.ParsedMessage, error) {
	// generate system instruction
	today := util.GetTodayLocalTimeString("2006-01-02")
	systemInstruction := fmt.Sprintf(
		reportSummaryInstructionTemplate,
		today,
		today,
		today,
	)

	// generate parse config
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  defaultResponseMIMEType,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: thinkingLevel,
		},
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"is_summary": {Type: genai.TypeBoolean},
				"type":       {Type: genai.TypeString, Enum: summaryTypeEnum()},
				"start_date": {Type: genai.TypeString},
				"end_date":   {Type: genai.TypeString},
			},
			Required: []string{"is_summary", "type", "start_date", "end_date"},
		},
	}

	// parse message
	raw, err := c.generate(ctx, config, rawMessage, "gemini parse record message failed")
	if err != nil {
		return nil, err
	}

	// mapping response
	var response geminiSummaryResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed message",
		"is_summary", response.IsSummary,
		"type", response.Type,
		"start_date", response.StartDate,
		"end_date", response.EndDate,
	)

	isValid := true
	if !response.IsSummary {
		isValid = false
	}

	// unknown or "ALL" type falls back to summarizing every transaction type
	summaryType, err := domain.ParseTransactionType(response.Type)
	if err != nil {
		summaryType = ""
	}

	return &domain.ParsedMessage{
		Valid: isValid,
		ReportMessage: &domain.ParsedReportMessage{
			StartDate:  response.StartDate,
			EndDate:    response.EndDate,
			DateLayout: "2006-01-02",
			Type:       summaryType,
		},
	}, nil
}

func (c *Client) ParseReportDetailMessage(ctx context.Context, rawMessage string) (*domain.ParsedMessage, error) {
	// generate system instruction
	today := util.GetTodayLocalTimeString("2006-01-02")
	systemInstruction := fmt.Sprintf(
		reportDetailInstructionTemplate,
		today,
		today,
		today,
	)

	// generate parse config
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemInstruction, genai.RoleUser),
		ResponseMIMEType:  defaultResponseMIMEType,
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: thinkingLevel,
		},
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

	// parse message
	raw, err := c.generate(ctx, config, rawMessage, "gemini parse record message failed")
	if err != nil {
		return nil, err
	}

	// mapping response
	var response geminiQueryResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed message",
		"is_query", response.IsQuery,
		"start_date", response.StartDate,
		"end_date", response.EndDate,
	)

	isValid := true
	if !response.IsQuery {
		isValid = false
	}

	return &domain.ParsedMessage{
		Valid: isValid,
		ReportMessage: &domain.ParsedReportMessage{
			StartDate:  response.StartDate,
			EndDate:    response.EndDate,
			DateLayout: "2006-01-02",
		},
	}, nil
}

func (c *Client) generate(ctx context.Context, config *genai.GenerateContentConfig, rawText, errMsg string) (string, error) {
	var (
		result     *genai.GenerateContentResponse
		err        error
		retryDelay = initialRetryDelay
	)

	for attempt := 1; attempt <= maxAttempt; attempt++ {
		result, err = c.genaiClient.Models.GenerateContent(ctx, c.config.Model, genai.Text(rawText), config)
		if err == nil {
			break
		}

		if !isRetryable(err) || attempt == maxAttempt {
			util.LogError(errMsg, err, "attempt", attempt)
			code := constant.ErrInternal
			if isRetryable(err) {
				code = constant.ErrServiceUnavailable
			}
			return "", util.WrapStd(code, errMsg, err)
		}

		util.LogWarn(errMsg+", retrying", err, "attempt", attempt)
		time.Sleep(retryDelay)
		retryDelay *= 2
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
	return strings.Contains(msg, "503") ||
		strings.Contains(msg, "UNAVAILABLE") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "RESOURCE_EXHAUSTED")
}
