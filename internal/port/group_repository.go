package port

import (
	"context"
	"whatsup-bot/internal/domain"
)

type GroupRepository interface {
	Save(ctx context.Context, groupData *domain.Group) error
}
