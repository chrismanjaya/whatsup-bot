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

type MessageParser interface {
	Parse(ctx context.Context, rawText string) (*ParsedMessage, error)
}
