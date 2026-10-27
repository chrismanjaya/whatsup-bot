package render

import (
	"context"
	"strings"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"

	"golang.org/x/text/language"
)

type (
	ReportSummaryResponseSuccess struct {
		ctx              context.Context
		transactions     []*domain.Transaction
		transacionPeriod domain.TransactionPeriod
		expenseSummary   *SummaryInfo
		incomeSummary    *SummaryInfo
	}

	SummaryInfo struct {
		TotalTransaction   int64
		CountTransaction   int
		TopCategory        domain.Category
		HighestTransaction int64
	}
)

func NewReportSummaryResponseSuccess(
	ctx context.Context,
	transactions []*domain.Transaction,
	transacionPeriod domain.TransactionPeriod,
	expenseSummary *SummaryInfo,
	incomeSummary *SummaryInfo,
) *ReportSummaryResponseSuccess {
	return &ReportSummaryResponseSuccess{
		ctx:              ctx,
		transactions:     transactions,
		transacionPeriod: transacionPeriod,
		expenseSummary:   expenseSummary,
		incomeSummary:    incomeSummary,
	}
}

func NewSummaryInfo(
	TotalTransaction int64,
	CountTransaction int,
	TopCategory domain.Category,
	HighestTransaction int64,
) *SummaryInfo {
	return &SummaryInfo{
		TotalTransaction:   TotalTransaction,
		CountTransaction:   CountTransaction,
		TopCategory:        TopCategory,
		HighestTransaction: HighestTransaction,
	}
}
func (r *ReportSummaryResponseSuccess) RenderSuccess() []Reply {
	// generate summary transactions
	var summaryTransactionMessage []string
	for _, transaction := range r.transactions {
		amountFormatted := util.FormatCurrency(transaction.Amount, "IDR", language.English)

		switch transaction.Type {
		case domain.Expense:
			amountFormatted = "-" + amountFormatted
		case domain.Income:
			amountFormatted = "+" + amountFormatted
		}

		message, err := util.Render(templateNameReportSummaryList, TemplateDataReportSummaryList{
			Date:            ReportDateFormatted(transaction.TransactionDate),
			AmountFormatted: amountFormatted,
			Description:     transaction.Description,
		})
		if err != nil {
			util.LogError("Render transaction template failed", err)
			return []Reply{}
		}

		summaryTransactionMessage = append(summaryTransactionMessage, message)
	}

	var reportSummaryTemplateName string
	templateData := TemplateDataReportSummary{
		StartDate:        ReportDateFormatted(r.transacionPeriod.Start),
		EndDate:          ReportDateFormatted(r.transacionPeriod.End),
		TransactionCount: util.FormatNumber(len(r.transactions), language.English),
		TransactionList:  strings.Join(summaryTransactionMessage, "\n"),
	}

	if r.expenseSummary != nil && r.expenseSummary.TotalTransaction > 0 {
		reportSummaryTemplateName = templateNameReportSummaryExpenseOnly

		templateData.TotalExpense = util.FormatCurrency(r.expenseSummary.TotalTransaction, "IDR", language.English)
		templateData.ExpenseCount = util.FormatNumber(r.expenseSummary.CountTransaction, language.English)
		templateData.TopExpenseCategory = r.expenseSummary.TopCategory.String()
		templateData.BiggestExpense = util.FormatCurrency(r.expenseSummary.HighestTransaction, "IDR", language.English)
	}

	if r.incomeSummary != nil && r.incomeSummary.TotalTransaction > 0 {
		reportSummaryTemplateName = templateNameReportSummaryIncomeOnly

		templateData.TotalIncome = util.FormatCurrency(r.incomeSummary.TotalTransaction, "IDR", language.English)
		templateData.IncomeCount = util.FormatNumber(r.incomeSummary.CountTransaction, language.English)
		templateData.TopIncomeCategory = r.incomeSummary.TopCategory.String()
		templateData.BiggestIncome = util.FormatCurrency(r.incomeSummary.HighestTransaction, "IDR", language.English)
	}

	if r.incomeSummary != nil && r.expenseSummary != nil && r.incomeSummary.TotalTransaction > 0 && r.expenseSummary.TotalTransaction > 0 {
		reportSummaryTemplateName = templateNameReportSummary
	}

	// generate summary info
	message, err := util.Render(reportSummaryTemplateName, templateData)
	if err != nil {
		util.LogError("Render report summary template failed", err)
		return []Reply{}
	}

	return []Reply{
		{
			Text: message,
			Metadata: Metadata{
				ListTransaction: r.transactions,
			},
		},
	}
}
