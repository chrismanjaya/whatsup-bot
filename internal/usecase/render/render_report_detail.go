package render

import (
	"context"
	"strings"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type ReportDetailResponseSuccess struct {
	ctx          context.Context
	transactions []*domain.Transaction
	totalPage    int
	currentPage  int
	period       domain.TransactionPeriod
	start        int
	end          int
	totalCount   int
}

func NewReportDetailResponseSuccess(
	ctx context.Context,
	transactions []*domain.Transaction,
	totalPage int,
	currentPage int,
	period domain.TransactionPeriod,
	start int,
	end int,
	totalCount int,
) *ReportDetailResponseSuccess {
	return &ReportDetailResponseSuccess{
		ctx:          ctx,
		transactions: transactions,
		totalPage:    totalPage,
		currentPage:  currentPage,
		period:       period,
		start:        start,
		end:          end,
		totalCount:   totalCount,
	}
}

func (r *ReportDetailResponseSuccess) RenderSuccess() []Reply {
	// render detail transaction template
	var transactionMessage []Reply
	for _, transaction := range r.transactions {
		// render transaction template
		message, err := util.Render(templateNameTransaction, TemplateDataTransaction{
			Type:            strings.ToUpper(convertTransactionType(transaction.Type)),
			Description:     transaction.Description,
			Category:        transaction.Category.String(),
			AmountFormatted: util.FormatCurrency(transaction.Amount, "IDR", language.English),
			DateFormatted:   TransactionDateFormatted(transaction.TransactionDate),
		})
		if err != nil {
			util.LogError("Render transaction template failed", err)
			return []Reply{}
		}

		transactionMessage = append(transactionMessage, Reply{
			Text: message,
			Metadata: Metadata{
				Transaction: transaction,
			},
		})
	}

	// render report detail page template
	message, err := util.Render(templateNameReportDetailPage, TemplateDataReportDetailPage{
		CurrentPage: util.FormatNumber(r.currentPage, language.English),
		TotalPage:   util.FormatNumber(r.totalPage, language.English),
		StartDate:   ReportDateFormatted(r.period.Start),
		EndDate:     ReportDateFormatted(r.period.End),
		Start:       util.FormatNumber(r.start, language.English),
		End:         util.FormatNumber(r.end, language.English),
		Total:       util.FormatNumber(r.totalCount, language.English),
	})
	if err != nil {
		util.LogError("Render report detail page template failed", err)
		return []Reply{}
	}

	return append(transactionMessage, Reply{
		Text: message,
		Metadata: Metadata{
			ListTransaction: r.transactions,
		},
	})
}
