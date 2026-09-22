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
		ID              int64
		UserID          int64
		Description     string
		Type            TransactionType
		Amount          int64
		Category        string
		IsShared        bool
		GroupID         int64
		TransactionDate time.Time
		CreatedAt       time.Time
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
