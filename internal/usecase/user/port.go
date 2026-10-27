package user

import (
	"context"
	"whatsupbot/internal/domain"
)

type (
	UserRepository interface {
		Create(ctx context.Context, u *domain.User) error
		FindByID(ctx context.Context, userID int64) (*domain.User, error)
		FindByJID(ctx context.Context, jid string) (*domain.User, error)
	}
)
