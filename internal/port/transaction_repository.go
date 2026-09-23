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
	FindByWAMessageID(ctx context.Context, waMessageID string) (*domain.Transaction, error)
	SetWAMessageID(ctx context.Context, id int64, waMessageID string) error
	Update(ctx context.Context, tx *domain.Transaction) error
	Delete(ctx context.Context, id int64) error
}
