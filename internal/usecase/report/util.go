package report

import (
	"context"
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

func ValidateUser(ctx context.Context, user *domain.User) (bool, int64) {
	if user == nil {
		return false, 0
	}

	if user.ID <= 0 {
		return false, 0
	}

	return true, user.ID
}

func ConvertTransactionDate(transactionDateStr, dateLayout string) *time.Time {
	if transactionDateStr == "" {
		return nil
	}

	transactionDate, err := time.ParseInLocation(dateLayout, transactionDateStr, util.Jakarta)
	if err != nil {
		util.LogWarn("invalid transaction date str", nil, "input", transactionDateStr, "layout", dateLayout)
		return nil
	}

	return &transactionDate
}

func ValidateTransactionRange(transactionPeriod *domain.TransactionPeriod) bool {
	if transactionPeriod == nil {
		return false
	}

	// period exceed 3 month
	if !transactionPeriod.End.Before(transactionPeriod.Start.AddDate(0, 3, 0)) {
		return false
	}

	return true
}

func generateTransactionPeriod(message *domain.ParsedReportMessage) *domain.TransactionPeriod {
	var (
		startDate = ConvertTransactionDate(message.StartDate, message.DateLayout)
		endDate   = ConvertTransactionDate(message.EndDate, message.DateLayout)
	)

	if endDate == nil || startDate == nil {
		return nil
	}

	var start, end time.Time
	if endDate.Before(*startDate) {
		start = *endDate
		end = *startDate
	} else {
		start = *startDate
		end = *endDate
	}

	return &domain.TransactionPeriod{
		Start: start,
		End:   end,
	}
}
