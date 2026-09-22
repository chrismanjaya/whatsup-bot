package port

import (
	"context"
	"whatsup-bot/internal/domain"
)

type UserRepository interface {
	Save(ctx context.Context, u *domain.User) error
	FindByJID(ctx context.Context, jid string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
}
