package render

import (
	"context"
	"strings"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type AmendResponseSuccess struct {
	ctx         context.Context
	transaction *domain.Transaction
	isUpdated   bool
	isDeleted   bool
}

func NewAmendResponseSuccess(
	ctx context.Context,
	transaction *domain.Transaction,
	isUpdated bool,
	isDeleted bool,
) *AmendResponseSuccess {
	return &AmendResponseSuccess{
		ctx:         ctx,
		transaction: transaction,
		isUpdated:   isUpdated,
		isDeleted:   isDeleted,
	}
}

func (r *AmendResponseSuccess) RenderSuccess() []Reply {
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
