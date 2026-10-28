package transaction

import (
	"context"
	"math/rand/v2"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

const (
	// maxTransactionAmount rejects amounts no personal transaction plausibly reaches (IDR 10 billion), usually a parsing slip.
	maxTransactionAmount int64 = 10_000_000_000
)

type RecordBatch struct {
	transactionRepo TransactionRepository

	geminiClient GeminiClient

	pickIndex func(n int) int
}

func NewRecordBatch(
	transactionRepo TransactionRepository,
	geminiClient GeminiClient,
) *RecordBatch {
	return &RecordBatch{
		transactionRepo: transactionRepo,
		geminiClient:    geminiClient,
		pickIndex:       rand.IntN,
	}
}

type (
	// RecordBatchInput is one record line, already attributed to a validated user (and group, for shared chats).
	RecordBatchInput struct {
		UserID     int64
		GroupID    *int64
		Text       string
		ReceivedAt time.Time
	}

	// RecordBatchResult is the outcome for the input at the same position: either Transaction (saved) or Err (a classified per-item failure).
	RecordBatchResult struct {
		Transaction *domain.Transaction
		Quip        string
		Err         error
	}
)

// Execute parses every input in one parser call and saves the valid ones. It returns one result per input, in input order; one bad item never fails the others. A non-nil error means the whole batch failed before anything was saved (e.g. constant.ErrTooManyRequests), so the caller may retry all inputs later.
func (uc *RecordBatch) Execute(ctx context.Context, inputs []RecordBatchInput) ([]RecordBatchResult, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// build parse items, one quip per user
	quipPositions := uc.selectQuipPositions(inputs)
	items := make([]domain.ParseBatchItem, 0, len(inputs))
	for i, input := range inputs {
		_, needQuip := quipPositions[i]
		items = append(items, domain.ParseBatchItem{
			Index:      i + 1,
			Text:       input.Text,
			ReceivedAt: input.ReceivedAt,
			NeedQuip:   needQuip,
		})
	}

	// parse batch
	messages, err := uc.geminiClient.ParseRecordBatch(ctx, items)
	if err != nil {
		return nil, util.Wrap(err, "parse record batch failed")
	}

	// map parsed messages back to inputs
	parsed := mapParsedByIndex(messages, len(inputs))

	// validate and save each item independently
	results := make([]RecordBatchResult, len(inputs))
	for i, input := range inputs {
		_, needQuip := quipPositions[i]
		results[i] = uc.recordItem(ctx, input, parsed[i], needQuip)
	}

	return results, nil
}

// selectQuipPositions picks one random input position per user; only those items ask the parser for a quip.
func (uc *RecordBatch) selectQuipPositions(inputs []RecordBatchInput) map[int]struct{} {
	positionsByUser := make(map[int64][]int)
	for i, input := range inputs {
		positionsByUser[input.UserID] = append(positionsByUser[input.UserID], i)
	}

	selected := make(map[int]struct{}, len(positionsByUser))
	for _, positions := range positionsByUser {
		selected[positions[uc.pickIndex(len(positions))]] = struct{}{}
	}

	return selected
}

// parsedItem is the parser output for one input position: message is nil when its index was missing or returned more than once.
type parsedItem struct {
	message *domain.ParsedMessage
	err     error
}

// mapParsedByIndex places each parsed message at position Index-1. Missing and duplicated indexes become per-item failures; out-of-range indexes are dropped.
func mapParsedByIndex(messages []*domain.ParsedMessage, size int) []parsedItem {
	counts := make([]int, size)
	items := make([]parsedItem, size)

	for _, message := range messages {
		if message == nil || message.Index < 1 || message.Index > size {
			util.LogWarn("parser returned unknown index", nil, "index", indexOf(message))
			continue
		}

		position := message.Index - 1
		counts[position]++
		items[position].message = message
	}

	for position, count := range counts {
		switch {
		case count == 0:
			items[position] = parsedItem{err: util.Wrap(constant.ErrTransactionUnparsed, "index missing from parse result")}
		case count > 1:
			items[position] = parsedItem{err: util.Wrap(constant.ErrTransactionUnparsed, "index duplicated in parse result")}
		}
	}

	return items
}

func indexOf(message *domain.ParsedMessage) int {
	if message == nil {
		return 0
	}
	return message.Index
}

func (uc *RecordBatch) recordItem(ctx context.Context, input RecordBatchInput, item parsedItem, needQuip bool) RecordBatchResult {
	if item.err != nil {
		return RecordBatchResult{Err: item.err}
	}

	message := item.message
	if !message.Valid || message.TransactionMessage == nil {
		return RecordBatchResult{Err: util.Wrap(constant.ErrInvalidRequest, "message is not a financial transaction")}
	}

	// construct transaction
	newTransaction, err := constructTransaction(message.TransactionMessage, input)
	if err != nil {
		return RecordBatchResult{Err: util.Wrap(err, "construct transaction failed")}
	}

	// create new transaction
	if err := uc.transactionRepo.Create(ctx, newTransaction); err != nil {
		return RecordBatchResult{Err: util.WrapStd(constant.ErrInternal, "create transaction failed", err)}
	}

	// a quip the model wrote for an item that didn't ask for one is dropped, keeping it to one per user
	quip := ""
	if needQuip {
		quip = message.Quip
	}

	return RecordBatchResult{
		Transaction: newTransaction,
		Quip:        quip,
	}
}
