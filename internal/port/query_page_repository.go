package port

import (
	"context"
	"whatsup-bot/internal/domain"
)

type QueryPageRepository interface {
	Save(ctx context.Context, p *domain.QueryPage) error
	FindByWAMessageID(ctx context.Context, waMessageID string) (*domain.QueryPage, error)
	SetWAMessageID(ctx context.Context, id int64, waMessageID string) error
}
