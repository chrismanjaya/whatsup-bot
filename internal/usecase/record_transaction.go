package usecase

import (
	"context"
	"fmt"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
)

type RecordTransactionUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
}

func NewRecordTransactionUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser) *RecordTransactionUseCase {
	return &RecordTransactionUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser}
}

func (uc *RecordTransactionUseCase) Execute(ctx context.Context, senderJID, rawText string, isShared bool, groupID int64) (string, error) {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil || user == nil {
		return "Please register first: register <name> <email>", nil
	}

	parsed, err := uc.parser.Parse(ctx, rawText)
	if err != nil {
		return "", fmt.Errorf("parse failed: %w", err)
	}
	if !parsed.Valid {
		return "I can only help track income and expenses. Try: \"spent 50k on lunch\"", nil
	}

	txType, err := domain.ParseTransactionType(parsed.Type)
	if err != nil {
		return "", fmt.Errorf("invalid type from parser: %w", err)
	}

	tx := &domain.Transaction{
		UserID:      user.ID,
		Description: rawText,
		Type:        txType,
		Amount:      parsed.Amount,
		Category:    parsed.Category,
		IsShared:    isShared,
		GroupID:     groupID,
	}

	if err := uc.txRepo.Save(ctx, tx); err != nil {
		return "", fmt.Errorf("save failed: %w", err)
	}

	return fmt.Sprintf("Recorded: %s - Rp%d (%s)", tx.Type, tx.Amount, tx.Category), nil
}
