package group

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

	GroupRepository interface {
		Create(ctx context.Context, group *domain.Group) error
		FindByID(ctx context.Context, groupID int64) (*domain.Group, error)
		FindByJID(ctx context.Context, jid string) (*domain.Group, error)

		AddMember(ctx context.Context, groupID, userID int64) error
		GetListMember(ctx context.Context, groupID int64) ([]*domain.User, error)
	}
)
