package render

import (
	"context"
	"strings"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type AmendResponseSuccess struct {
	ctx            context.Context
	transaction    *domain.Transaction
	isUpdated      bool
	isDeleted      bool
	deletedMessage string
}

func NewAmendResponseSuccess(
	ctx context.Context,
	transaction *domain.Transaction,
	isUpdated bool,
	isDeleted bool,
	deletedMessage string,
) *AmendResponseSuccess {
	return &AmendResponseSuccess{
		ctx:            ctx,
		transaction:    transaction,
		isUpdated:      isUpdated,
		isDeleted:      isDeleted,
		deletedMessage: deletedMessage,
	}
}

func (r *AmendResponseSuccess) RenderSuccess() []Reply {
	// deleted: a canned line quoted to the confirmation the user replied to
	if r.isDeleted {
		return []Reply{{Text: r.deletedMessage, IsQuotedReplied: true}}
	}

	// render amend template
	message, err := util.Render(templateNameTransaction, TemplateDataTransaction{
		Type:            strings.ToUpper(convertTransactionType(r.transaction.Type)),
		Description:     r.transaction.Description,
		Category:        r.transaction.Category.String(),
		AmountFormatted: util.FormatCurrency(r.transaction.Amount, "IDR", language.English),
		DateFormatted:   TransactionDateFormatted(r.transaction.TransactionDate),
	})
	if err != nil {
		util.LogError("Render transaction template failed", err)
		return []Reply{}
	}

	metadata := Metadata{
		Transaction: r.transaction,
	}

	return []Reply{
		{
			Text:     message,
			Metadata: metadata,
		},
	}
}
