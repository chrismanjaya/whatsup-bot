package port

import (
	"context"
	"whatsup-bot/internal/domain"
)

type GroupRepository interface {
	Save(ctx context.Context, g *domain.Group) error
	FindByJID(ctx context.Context, jid string) (*domain.Group, error)
	FindByID(ctx context.Context, id int64) (*domain.Group, error)

	AddMember(ctx context.Context, groupID, userID int64) error
	ListMembers(ctx context.Context, groupID int64) ([]*domain.User, error)
}
