package integration

import (
	"context"
	"whatsupbot/internal/domain"
)

type (
	MCPTokenRepository interface {
		Upsert(ctx context.Context, token *domain.MCPToken) error
		FindByHash(ctx context.Context, tokenHash string) (*domain.MCPToken, error)
	}

	UserRepository interface {
		FindByID(ctx context.Context, userID int64) (*domain.User, error)
	}
)
