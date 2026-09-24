package usecase

import (
	"context"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type AmendTransactionUseCase struct {
	userRepo port.UserRepository
	txRepo   port.TransactionRepository
	parser   port.MessageParser
}

func NewAmendTransactionUseCase(userRepo port.UserRepository, txRepo port.TransactionRepository, parser port.MessageParser) *AmendTransactionUseCase {
	return &AmendTransactionUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser}
}

// Execute handles a reply to a transaction confirmation message identified
// by waMessageID. handled is false when waMessageID doesn't match a
// transaction the bot recorded, so the caller can fall back to treating the
// message as a normal command. tx is the updated transaction on "update" (so
// the caller can re-associate the new confirmation's WhatsApp message ID
// with it), and nil for "delete" or "none".
func (uc *AmendTransactionUseCase) Execute(ctx context.Context, senderJID, waMessageID, rawText string) (reply string, tx *domain.Transaction, handled bool, err error) {
	existing, err := uc.txRepo.FindByWAMessageID(ctx, waMessageID)
	if err != nil {
		return "", nil, false, utils.WrapStd(constant.ErrInternal, "lookup transaction failed", err)
	}
	if existing == nil {
		return "", nil, false, nil
	}

	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return "", nil, true, utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil || user.ID != existing.UserID {
		return message.NotTransactionOwner, nil, true, nil
	}

	amend, err := uc.parser.ParseAmend(ctx, rawText, &port.CurrentTransaction{
		Type:        existing.Type.String(),
		Amount:      existing.Amount,
		Category:    existing.Category.String(),
		Description: existing.Description,
		Date:        existing.TransactionDate.Format("2006-01-02"),
	})
	if err != nil {
		return "", nil, true, utils.Wrap(err, "parse amend failed")
	}

	switch amend.Action {
	case "delete":
		if err := uc.txRepo.Delete(ctx, existing.ID); err != nil {
			return "", nil, true, utils.WrapStd(constant.ErrInternal, "delete failed", err)
		}
		return renderDeletedReply(existing), nil, true, nil

	case "update":
		txType, typeErr := domain.ParseTransactionType(amend.Type)
		if typeErr != nil {
			txType = existing.Type
		}
		category, catErr := domain.ParseCategory(amend.Category)
		if catErr != nil {
			category = domain.CategoryOther
		}
		description := amend.Description
		if description == "" {
			description = existing.Description
		}

		existing.Type = txType
		existing.Amount = amend.Amount
		existing.Category = category
		existing.Description = description
		existing.TransactionDate = resolveTransactionDate(amend.Date, existing.TransactionDate)

		if err := uc.txRepo.Update(ctx, existing); err != nil {
			return "", nil, true, utils.WrapStd(constant.ErrInternal, "update failed", err)
		}
		if existing.Amount <= 0 {
			return renderAmountPrompt(existing), existing, true, nil
		}
		return renderTransactionReply(existing), existing, true, nil

	default:
		if existing.Amount <= 0 {
			return renderAmountPrompt(existing), existing, true, nil
		}
		return message.AmendUnclear, nil, true, nil
	}
}
