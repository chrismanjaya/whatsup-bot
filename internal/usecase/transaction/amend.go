package transaction

import (
	"context"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/parser"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/usecase/message"
	"whatsup-bot/internal/usecase/persona"
	"whatsup-bot/internal/usecase/response"
	"whatsup-bot/internal/util"
)

type AmendUseCase struct {
	userRepo repository.UserRepository
	txRepo   repository.TransactionRepository
	parser   parser.MessageParser
	persona  *persona.Picker
}

func NewAmendUseCase(userRepo repository.UserRepository, txRepo repository.TransactionRepository, parser parser.MessageParser, picker *persona.Picker) *AmendUseCase {
	return &AmendUseCase{userRepo: userRepo, txRepo: txRepo, parser: parser, persona: picker}
}

// Execute handles a reply to a transaction confirmation message identified
// by waMessageID. handled is false when waMessageID doesn't match a
// transaction the bot recorded, so the caller can fall back to treating the
// message as a normal command. tx is the updated transaction on "update" (so
// the caller can re-associate the new confirmation's WhatsApp message ID
// with it, via the first reply's Tx), and unlinked for "delete" or "none".
// A reply that fills in a missing amount may be followed by a quip message.
func (uc *AmendUseCase) Execute(ctx context.Context, senderJID, waMessageID, rawText string) (replies []response.Reply, handled bool, err error) {
	existing, err := uc.txRepo.FindByWAMessageID(ctx, waMessageID)
	if err != nil {
		return nil, false, util.WrapStd(constant.ErrInternal, "lookup transaction failed", err)
	}
	if existing == nil {
		return nil, false, nil
	}

	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return nil, true, util.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil || user.ID != existing.UserID {
		return []response.Reply{{Text: message.NotTransactionOwner}}, true, nil
	}

	amend, err := uc.parser.ParseAmend(ctx, rawText, &parser.CurrentTransaction{
		Type:        existing.Type.String(),
		Amount:      existing.Amount,
		Category:    existing.Category.String(),
		Description: existing.Description,
		Date:        existing.TransactionDate.Format("2006-01-02"),
	})
	if err != nil {
		return nil, true, util.Wrap(err, "parse amend failed")
	}

	switch amend.Action {
	case "delete":
		if err := uc.txRepo.Delete(ctx, existing.ID); err != nil {
			return nil, true, util.WrapStd(constant.ErrInternal, "delete failed", err)
		}
		return []response.Reply{{Text: renderDeletedReply(existing)}}, true, nil

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
		// A reply that fills in the missing amount completes a new
		// transaction, so it's treated like one (it can get a quip); any
		// other update is just a correction and doesn't.
		completesNew := existing.Amount <= 0 && amend.Amount > 0

		existing.Type = txType
		existing.Amount = amend.Amount
		existing.Category = category
		existing.Description = description
		existing.TransactionDate = resolveTransactionDate(amend.Date, existing.TransactionDate)

		if err := uc.txRepo.Update(ctx, existing); err != nil {
			return nil, true, util.WrapStd(constant.ErrInternal, "update failed", err)
		}
		if existing.Amount <= 0 {
			return []response.Reply{{Text: renderAmountPrompt(existing), Tx: existing}}, true, nil
		}
		quip := ""
		if completesNew {
			quip = newTransactionQuip(uc.persona, "", existing, senderJID)
		}
		return transactionReplies(existing, quip), true, nil

	default:
		if existing.Amount <= 0 {
			return []response.Reply{{Text: renderAmountPrompt(existing), Tx: existing}}, true, nil
		}
		return []response.Reply{{Text: message.AmendUnclear}}, true, nil
	}
}
