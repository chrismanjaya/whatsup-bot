package port

import (
	"context"
	"time"
	"whatsup-bot/internal/domain"
)

type TransactionRepository interface {
	Save(ctx context.Context, tx *domain.Transaction) error
	FindBySender(ctx context.Context, userID int64, since time.Time) ([]*domain.Transaction, error)
	FindByGroup(ctx context.Context, groupID int64, since time.Time) ([]*domain.Transaction, error)
}
