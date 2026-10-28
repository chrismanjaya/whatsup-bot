package domain

import "strings"

type ParsedMessage struct {
	Valid bool
	Quip  string

	TransactionMessage *ParsedTransactionMessage
	ReportMessage      *ParsedReportMessage
}

type (
	TransactionAction string

	ParsedTransactionMessage struct {
		Action      TransactionAction
		Type        string
		Amount      int64
		Category    string
		Description string
		Date        string
		DateLayout  string
	}

	ParsedReportMessage struct {
		StartDate  string
		EndDate    string
		Offset     int
		DateLayout string
		Type       TransactionType // empty means all transaction types
	}
)

const (
	TRANSACTION_ACTION_NONE   = "none"
	TRANSACTION_ACTION_UPDATE = "update"
	TRANSACTION_ACTION_DELETE = "delete"
)

func (a TransactionAction) String() string {
	return string(a)
}

func TransactionActionEnum(action string) TransactionAction {
	switch strings.ToLower(action) {
	case TRANSACTION_ACTION_UPDATE:
		return TRANSACTION_ACTION_UPDATE
	case TRANSACTION_ACTION_DELETE:
		return TRANSACTION_ACTION_DELETE
	default:
		return TRANSACTION_ACTION_NONE
	}
}
