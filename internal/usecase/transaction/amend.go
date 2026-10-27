package transaction

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Amend struct {
	groupRepo       GroupRepository
	transactionRepo TransactionRepository
	userRepo        UserRepository

	geminiClient GeminiClient
}

func NewAmend(
	groupRepo GroupRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
	geminiClient GeminiClient,
) *Amend {
	return &Amend{
		groupRepo:       groupRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		geminiClient:    geminiClient,
	}
}

type (
	AmendOutcome int32

	AmendResult struct {
		Outcome     AmendOutcome
		Transaction *domain.Transaction
	}
)

const (
	AmendOutcomeUnknown AmendOutcome = 0
	AmendOutcomeUpdated AmendOutcome = 1
	AmendOutcomeDeleted AmendOutcome = 2
)

func (uc *Amend) Execute(ctx context.Context, rawText string, user *domain.User, group *domain.Group, messageID string) (*AmendResult, error) {
	// validate user
	isValidUser, userID := ValidateUser(user)
	if !isValidUser {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	// get and validate reply transacttion
	replyTransaction, err := uc.transactionRepo.FindByMessage(ctx, messageID)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "lookup reply transaction failed", err)
	}

	if replyTransaction == nil {
		return nil, util.WrapStd(constant.ErrTransactionNotFound, "lookup reply transaction not found", nil)
	}

	if replyTransaction.UserID != userID {
		return nil, util.WrapStd(constant.ErrInvalidRequest, "user is not match", nil)
	}

	// parse message
	message, err := uc.geminiClient.ParseAmendMessage(ctx, rawText, replyTransaction)
	if err != nil {
		return nil, util.Wrap(err, "parse message failed")
	}

	if !message.Valid || message.TransactionMessage == nil {
		return nil, util.Wrap(constant.ErrInvalidRequest, "message is not a financial transaction")
	}

	// amend action
	switch message.TransactionMessage.Action {
	case domain.TRANSACTION_ACTION_UPDATE:
		return uc.UpdateTransaction(ctx, replyTransaction, message.TransactionMessage)
	case domain.TRANSACTION_ACTION_DELETE:
		return uc.DeleteTransaction(ctx, replyTransaction)
	default:
		return uc.UnknownTransaction(ctx, replyTransaction)
	}
}

func (uc *Amend) DeleteTransaction(ctx context.Context, existing *domain.Transaction) (*AmendResult, error) {
	if err := uc.transactionRepo.Delete(ctx, existing.ID); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "delete transaction failed", err)
	}

	return &AmendResult{
		Outcome:     AmendOutcomeDeleted,
		Transaction: existing,
	}, nil
}

func (uc *Amend) UpdateTransaction(ctx context.Context, existing *domain.Transaction, message *domain.ParsedTransactionMessage) (*AmendResult, error) {
	updatedTransaction := constructUpdatedTransaction(existing, message)
	if err := uc.transactionRepo.Update(ctx, updatedTransaction); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "update transaction failed", err)
	}

	return &AmendResult{
		Outcome:     AmendOutcomeUpdated,
		Transaction: updatedTransaction,
	}, nil
}

func constructUpdatedTransaction(existing *domain.Transaction, message *domain.ParsedTransactionMessage) *domain.Transaction {
	// update transaction type
	newTransactionType := existing.Type
	if transactionType, err := domain.ParseTransactionType(message.Type); err == nil {
		newTransactionType = transactionType
	}

	// update description
	newDescription := existing.Description
	if message.Description != "" {
		newDescription = message.Description
	}

	// update amount
	newAmount := existing.Amount
	if message.Amount > 0 {
		newAmount = message.Amount
	}

	// update category
	newCategory := existing.Category
	if message.Category != "" {
		if category, err := domain.ParseCategory(message.Category); err == nil {
			newCategory = category
		}
	}

	// update transaction date
	newTransactionDate := existing.TransactionDate
	if message.Date != "" {
		if transactionDate := ConvertTransactionDate(message.Date, message.DateLayout); transactionDate != nil {
			newTransactionDate = *transactionDate
		}
	}

	return &domain.Transaction{
		ID:              existing.ID,
		Description:     newDescription,
		Type:            newTransactionType,
		Amount:          newAmount,
		Category:        newCategory,
		TransactionDate: newTransactionDate,
	}
}

func (uc *Amend) UnknownTransaction(ctx context.Context, existing *domain.Transaction) (*AmendResult, error) {
	return &AmendResult{
		Outcome:     AmendOutcomeUnknown,
		Transaction: existing,
	}, nil
}
