package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

// ParseRecordBatch parses many record lines in one request. The model only sees each item's neutral Index; callers keep the mapping back to users and messages. Results carry the Index the model returned and may be missing, duplicated or out of range — the caller validates them.
func (c *Client) ParseRecordBatch(ctx context.Context, items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
	// generate system instruction
	categories := categoryEnum()
	systemInstruction := fmt.Sprintf(
		recordBatchInstructionTemplate,
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
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"index":       {Type: genai.TypeInteger},
					"valid":       {Type: genai.TypeBoolean},
					"type":        {Type: genai.TypeString, Enum: transactionTypeEnum()},
					"amount":      {Type: genai.TypeInteger},
					"category":    {Type: genai.TypeString, Enum: categories},
					"description": {Type: genai.TypeString},
					"date":        {Type: genai.TypeString},
					"quip":        {Type: genai.TypeString},
				},
				Required: []string{"index", "valid", "type", "amount", "category", "description", "date"},
			},
		},
	}

	// build user content
	content, err := buildRecordBatchContent(items)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "build record batch content failed", err)
	}

	// parse message
	raw, err := c.generate(ctx, config, content, "gemini parse record batch failed")
	if err != nil {
		return nil, err
	}

	// mapping response
	var responses []geminiRecordBatchResponse
	if err := json.Unmarshal([]byte(raw), &responses); err != nil {
		slog.Error("failed to parse gemini response", "raw", raw, "error", err)
		return nil, fmt.Errorf("failed to parse gemini response %q: %w", raw, err)
	}

	slog.Info("gemini parsed record batch", "requested", len(items), "returned", len(responses))

	messages := make([]*domain.ParsedMessage, 0, len(responses))
	for _, response := range responses {
		messages = append(messages, &domain.ParsedMessage{
			Index: response.Index,
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
		})
	}

	return messages, nil
}

// buildRecordBatchContent wraps the items as a JSON array inside <items> tags. json.Marshal escapes '<' and '>', so a text can't close the tag and pose as instructions.
func buildRecordBatchContent(items []domain.ParseBatchItem) (string, error) {
	requests := make([]geminiRecordBatchRequest, 0, len(items))
	for _, item := range items {
		quip := "no"
		if item.NeedQuip {
			quip = "yes"
		}

		requests = append(requests, geminiRecordBatchRequest{
			Index:        item.Index,
			ReceivedDate: item.ReceivedAt.In(util.Jakarta).Format("2006-01-02"),
			Quip:         quip,
			Text:         item.Text,
		})
	}

	itemsJSON, err := json.Marshal(requests)
	if err != nil {
		return "", err
	}

	return "<items>\n" + string(itemsJSON) + "\n</items>", nil
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

		// quota exceeded: retrying within seconds can't help, fail fast
		if isQuotaExceeded(err) {
			util.LogError(errMsg, err, "attempt", attempt)
			return "", util.WrapStd(constant.ErrTooManyRequests, errMsg, err)
		}

		if !isUnavailable(err) || attempt == maxAttempt {
			util.LogError(errMsg, err, "attempt", attempt)
			code := constant.ErrInternal
			if isUnavailable(err) {
				code = constant.ErrServiceUnavailable
			}
			return "", util.WrapStd(code, errMsg, err)
		}

		util.LogWarn(errMsg+", retrying", err, "attempt", attempt)
		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return "", util.WrapStd(constant.ErrServiceUnavailable, errMsg, ctx.Err())
		}
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

func isQuotaExceeded(err error) bool {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusTooManyRequests || apiErr.Status == "RESOURCE_EXHAUSTED"
	}

	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED")
}

func isUnavailable(err error) bool {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusServiceUnavailable || apiErr.Status == "UNAVAILABLE"
	}

	msg := err.Error()
	return strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE")
}
