package render

import (
	"context"
	"strings"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type RecordResponseSuccess struct {
	ctx          context.Context
	transaction  *domain.Transaction
	quip         string
	isFirstToday bool
}

func NewRecordResponseSuccess(
	ctx context.Context,
	transaction *domain.Transaction,
	quip string,
	isFirstToday bool,
) *RecordResponseSuccess {
	return &RecordResponseSuccess{
		ctx:          ctx,
		transaction:  transaction,
		quip:         quip,
		isFirstToday: isFirstToday,
	}
}

func (r *RecordResponseSuccess) RenderSuccess() []Reply {
	// render record template
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

	replies := []Reply{
		{
			Text: message,
			Metadata: Metadata{
				Transaction: r.transaction,
			},
		},
	}

	// generate gimmick
	if r.quip != "" {
		if gimmickMessage, err := util.Render(templateNameGimmick, TemplateDataGimmick{Quip: r.quip}); err == nil && gimmickMessage != "" {
			replies = append(replies, Reply{Text: gimmickMessage})
		}
	}

	return replies
}
