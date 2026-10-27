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
