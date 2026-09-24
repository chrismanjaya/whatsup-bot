package port

import (
	"context"
	"time"
	"whatsup-bot/internal/domain"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *domain.Transaction) error
	FindByUser(ctx context.Context, userID int64, since time.Time) ([]*domain.Transaction, error)
	FindByGroup(ctx context.Context, groupID int64, since time.Time) ([]*domain.Transaction, error)
	// ListByUserBetween returns the user's transactions dated in [from, to)
	// (compared by calendar date), oldest first, plus the total match count.
	ListByUserBetween(ctx context.Context, userID int64, from, to time.Time, limit, offset int) (txs []*domain.Transaction, total int, err error)
	FindByWAMessageID(ctx context.Context, waMessageID string) (*domain.Transaction, error)
	SetWAMessageID(ctx context.Context, id int64, waMessageID string) error
	Update(ctx context.Context, tx *domain.Transaction) error
	Delete(ctx context.Context, id int64) error
}
