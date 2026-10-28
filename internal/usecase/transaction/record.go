package transaction

import (
	"context"
	"log/slog"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Record struct {
	transactionRepo TransactionRepository
}

func NewRecord(
	transactionRepo TransactionRepository,
) *Record {
	return &Record{
		transactionRepo: transactionRepo,
	}
}

func constructTransaction(message *domain.ParsedTransactionMessage, input RecordBatchInput) (*domain.Transaction, error) {
	// mapping transaction type
	transactionType, err := domain.ParseTransactionType(message.Type)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInvalidTransactionType, "invalid type from parser", err)
	}

	// mapping transaction category
	category, err := domain.ParseCategory(message.Category)
	if err != nil {
		slog.Warn("invalid category from gemini, defaulting to other", "category", message.Category, "error", err)
		category = domain.CategoryOther
	}

	// validate amount
	if message.Amount < 0 || message.Amount > maxTransactionAmount {
		return nil, util.Wrap(constant.ErrInvalidTransactionAmount, "amount out of range")
	}

	// validate transaction date against the day the line was received
	transactionDate := input.ReceivedAt
	if date := ConvertTransactionDate(message.Date, message.DateLayout); date != nil {
		transactionDate = *date
	}

	if isAfterDay(transactionDate, input.ReceivedAt) {
		return nil, util.Wrap(constant.ErrFutureTransactionDate, "transaction date after received date")
	}

	return &domain.Transaction{
		UserID:          input.UserID,
		Description:     message.Description,
		Type:            transactionType,
		Amount:          message.Amount,
		Category:        category,
		IsShared:        input.GroupID != nil,
		GroupID:         input.GroupID,
		TransactionDate: transactionDate,
	}, nil
}

func (uc *Record) UpdateMessageID(ctx context.Context, transactionID int64, messageID string) error {
	if err := uc.transactionRepo.SetMessage(ctx, transactionID, messageID); err != nil {
		return util.Wrap(err, "update message id failed")
	}

	return nil
}
