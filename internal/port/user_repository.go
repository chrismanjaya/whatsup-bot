package port

import (
	"context"
	"whatsup-bot/internal/domain"
)

type UserRepository interface {
	Save(ctx context.Context, userData *domain.User) error
}
