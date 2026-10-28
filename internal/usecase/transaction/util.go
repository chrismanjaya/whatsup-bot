package transaction

import (
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

func ValidateUser(user *domain.User) (bool, int64) {
	if user == nil {
		return false, 0
	}

	if user.ID <= 0 {
		return false, 0
	}

	return true, user.ID
}

func ValidateGroup(group *domain.Group) (bool, int64) {
	if group == nil {
		return false, 0
	}

	if group.ID <= 0 {
		return false, 0
	}

	return true, group.ID
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

// isAfterDay reports whether a falls on a later calendar day than b in Jakarta.
func isAfterDay(a, b time.Time) bool {
	ay, am, ad := a.In(util.Jakarta).Date()
	by, bm, bd := b.In(util.Jakarta).Date()
	return time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC).After(time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC))
}
