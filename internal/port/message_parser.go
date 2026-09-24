package port

import (
	"context"
)

type ParsedMessage struct {
	Valid       bool
	Type        string
	Amount      int64
	Category    string
	Description string
	Date        string // "YYYY-MM-DD", resolved by the model relative to today
}

// CurrentTransaction is the state of a previously recorded transaction, given
// to ParseAmend as context for interpreting a reply to its confirmation.
type CurrentTransaction struct {
	Type        string
	Amount      int64
	Category    string
	Description string
	Date        string // "YYYY-MM-DD"
}

// AmendResult is the model's interpretation of a reply to a transaction
// confirmation. For Action "update", it carries the full new field set
// (unchanged fields copied over from CurrentTransaction); for "delete" and
// "none" the fields echo CurrentTransaction unchanged.
type AmendResult struct {
	Action      string // "update", "delete", or "none"
	Type        string
	Amount      int64
	Category    string
	Description string
	Date        string
}

// QueryIntent is the model's reading of a request to list past transactions.
// Dates are "YYYY-MM-DD", both inclusive.
type QueryIntent struct {
	IsQuery   bool
	StartDate string
	EndDate   string
}

type MessageParser interface {
	Parse(ctx context.Context, rawText string) (*ParsedMessage, error)
	ParseQuery(ctx context.Context, rawText string) (*QueryIntent, error)
	ParseAmend(ctx context.Context, rawText string, current *CurrentTransaction) (*AmendResult, error)
}
