package domain

import "time"

const (
	Income  TransactionType = "CR"
	Outcome TransactionType = "DB"
)

type (
	TransactionType string

	Transaction struct {
		ID          int64
		UserID      int64
		Description string
		Type        TransactionType
		Amount      int64
		Category    string
		IsShared    bool
		GroupID     int64
		CreatedAt   time.Time
	}
)
