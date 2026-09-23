package domain

import (
	"fmt"
	"time"
)

const (
	Income  TransactionType = "CR"
	Outcome TransactionType = "DB"
)

type (
	TransactionType string

	Transaction struct {
		ID              int64           `json:"id"`
		UserID          int64           `json:"user_id"`
		Description     string          `json:"description"`
		Type            TransactionType `json:"type"`
		Amount          int64           `json:"amount"`
		Category        string          `json:"category"`
		IsShared        bool            `json:"is_shared"`
		GroupID         int64           `json:"group_id"`
		TransactionDate time.Time       `json:"transaction_date"`
		CreatedAt       time.Time       `json:"created_at"`
	}
)

// String satisfies fmt.Stringer, so TransactionType prints cleanly in logs.
func (t TransactionType) String() string {
	return string(t)
}

// Valid reports whether t is one of the defined enum values.
func (t TransactionType) Valid() bool {
	switch t {
	case Income, Outcome:
		return true
	default:
		return false
	}
}

// ParseTransactionType safely converts an external string (e.g. from the
// Gemini adapter or a DB row) into a TransactionType, rejecting anything
// that isn't a known value instead of silently accepting garbage.
func ParseTransactionType(s string) (TransactionType, error) {
	t := TransactionType(s)
	if !t.Valid() {
		return "", fmt.Errorf("invalid transaction type: %q", s)
	}
	return t, nil
}
