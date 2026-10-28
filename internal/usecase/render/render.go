package render

import (
	"time"
	"whatsupbot/internal/domain"
)

type (
	Reply struct {
		Text     string
		Metadata Metadata
	}

	Metadata struct {
		Transaction     *domain.Transaction   // for record or amend
		User            *domain.User          // for registration
		ListTransaction []*domain.Transaction // for report
	}
)

var transactionTypeMapping = map[domain.TransactionType]string{
	domain.Income:  "Income",
	domain.Expense: "Expense",
}

const (
	templateNameRegistration = "registration.tmpl"

	templateNameSplitTransaction  = "split_transaction.tmpl"
	templateNameSplitMemberShares = "split_member_shares.tmpl"
	templateNameSplitMemberOwes   = "split_member_owes.tmpl"

	templateNameTransaction = "transaction.tmpl"

	templateNameReportDetailPage = "report_detail_page.tmpl"

	templateNameReportSummary            = "report_summary.tmpl"
	templateNameReportSummaryList        = "report_summary_list.tmpl"
	templateNameReportSummaryExpenseOnly = "report_summary_expense_only.tmpl"
	templateNameReportSummaryIncomeOnly  = "report_summary_income_only.tmpl"

	templateNameGimmick = "gimmick.tmpl"

	templateNameMCPToken = "mcp_token.tmpl"
)

type (
	// Registration template data
	TemplateDataRegistration struct {
		Name string
	}

	// Split transaction template data
	TemplateDataSplitTransaction struct {
		MemberShares         string
		MemberOwes           string
		TotalAmountFormatted string
		ShareAmountFormatted string
	}

	TemplateDataSplitMemberShares struct {
		MemberName      string
		AmountFormatted string
	}

	TemplateDataSplitMemberOwes struct {
		MemberName      string
		AmountFormatted string
	}

	// Transaction template data
	TemplateDataTransaction struct {
		Type            string
		Description     string
		Category        string
		AmountFormatted string
		DateFormatted   string
	}

	// Report detail template data
	TemplateDataReportDetailPage struct {
		CurrentPage string
		TotalPage   string
		StartDate   string
		EndDate     string
		Start       string
		End         string
		Total       string
	}

	// Report summary template data
	TemplateDataReportSummary struct {
		StartDate            string
		EndDate              string
		TotalExpense         string
		ExpenseCount         string
		TotalIncome          string
		IncomeCount          string
		TopExpenseCategory   string
		TopIncomeCategory    string
		BiggestExpense       string
		BiggestExpenseAmount string
		BiggestIncome        string
		BiggestIncomeAmount  string
		TransactionCount     string
		TransactionList      string
	}

	TemplateDataReportSummaryList struct {
		Date            string
		AmountFormatted string
		Description     string
	}

	// Gimmick template data
	TemplateDataGimmick struct {
		Quip string
	}

	// MCP token template data
	TemplateDataMCPToken struct {
		MCPURL string
	}
)

func TransactionDateFormatted(date time.Time) string {
	return date.Format("2 Jan 2006")
}

func ReportDateFormatted(date time.Time) string {
	return date.Format("02/01/06")
}

func convertTransactionType(transactionType domain.TransactionType) string {
	if typeStr, ok := transactionTypeMapping[transactionType]; !ok {
		return "Expense"
	} else {
		return typeStr
	}
}
