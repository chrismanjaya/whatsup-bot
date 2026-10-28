package domain

import (
	"strings"
	"time"
)

type ParsedMessage struct {
	Index int // position of the item in a batch request (1-based); 0 outside a batch
	Valid bool
	Quip  string

	TransactionMessage *ParsedTransactionMessage
	ReportMessage      *ParsedReportMessage
}

// ParseBatchItem is one record line sent to the parser in a batch. It deliberately carries no user, chat or message identity: the parser only sees the neutral Index.
type ParseBatchItem struct {
	Index      int
	Text       string
	ReceivedAt time.Time
	NeedQuip   bool
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
