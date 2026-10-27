package transaction

import (
	"context"
	"log/slog"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Record struct {
	groupRepo       GroupRepository
	transactionRepo TransactionRepository
	userRepo        UserRepository

	geminiClient GeminiClient
}

func NewRecord(
	groupRepo GroupRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) *Record {
	return &Record{
		groupRepo:       groupRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		geminiClient:    geminiClient,
	}
}

type RecordResult struct {
	Transaction  *domain.Transaction
	ModelQuip    string
	IsFirstToday bool
}

func (uc *Record) Execute(ctx context.Context, rawText string, user *domain.User, group *domain.Group) (*RecordResult, error) {
	// validate user
	isValidUser, userID := ValidateUser(user)
	if !isValidUser {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	// validate group
	isGroup, groupID := ValidateGroup(group)

	// parse message
	message, err := uc.geminiClient.ParseRecordMessage(ctx, rawText)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "parse message failed", err)
	}

	if !message.Valid || message.TransactionMessage == nil {
		return nil, util.Wrap(constant.ErrInvalidRequest, "message is not a financial transaction")
	}

	// construct transaction
	newTransaction, err := constructTransaction(message.TransactionMessage, userID, groupID, isGroup)
	if err != nil {
		return nil, util.Wrap(err, "construct transaction failed")
	}

	// check today first transaction
	isFirstToday := uc.isFirstToday(ctx, userID)

	// create new transaction
	if err := uc.transactionRepo.Create(ctx, newTransaction); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "create transaction failed", err)
	}

	// construct record result
	return constructRecordResult(newTransaction, message.Quip, isFirstToday), nil
}

func constructTransaction(message *domain.ParsedTransactionMessage, userID, groupID int64, isGroup bool) (*domain.Transaction, error) {
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

	// validate transaction date
	transactionDate := time.Now()
	if date := ConvertTransactionDate(message.Date, message.DateLayout); date != nil {
		transactionDate = *date
	}

	// validate group
	var transactionGroupID *int64
	if isGroup {
		transactionGroupID = &groupID
	}

	return &domain.Transaction{
		UserID:          userID,
		Description:     message.Description,
		Type:            transactionType,
		Amount:          message.Amount,
		Category:        category,
		IsShared:        isGroup,
		GroupID:         transactionGroupID,
		TransactionDate: transactionDate,
	}, nil
}

func constructRecordResult(transaction *domain.Transaction, rawQuip string, isFirstToday bool) *RecordResult {
	return &RecordResult{
		Transaction:  transaction,
		ModelQuip:    rawQuip,
		IsFirstToday: isFirstToday,
	}
}

func (uc *Record) isFirstToday(ctx context.Context, userID int64) bool {
	lastTransactionTime, err := uc.transactionRepo.GetLastTransactionTime(ctx, userID)
	if err != nil {
		util.LogWarn("last transaction lookup failed, skipping greeting", err, "user_id", userID)
		return false
	}

	// no previous transaction at all, so this one is the first today
	if lastTransactionTime == nil {
		return true
	}

	return !util.IsSameDay(*lastTransactionTime, time.Now())
}

func (uc *Record) UpdateMessageID(ctx context.Context, transactionID int64, messageID string) error {
	if err := uc.transactionRepo.SetMessage(ctx, transactionID, messageID); err != nil {
		return util.Wrap(err, "update message id failed")
	}

	return nil
}
