package group

import (
	"context"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/util"
)

type EnsureUseCase struct {
	groupRepo repository.GroupRepository
}

func NewEnsureUseCase(groupRepo repository.GroupRepository) *EnsureUseCase {
	return &EnsureUseCase{groupRepo: groupRepo}
}

// Execute returns the group's internal ID, creating the group on first sight.
func (uc *EnsureUseCase) Execute(ctx context.Context, jid, name string) (int64, error) {
	existing, err := uc.groupRepo.FindByJID(ctx, jid)
	if err != nil {
		return 0, util.WrapStd(constant.ErrInternal, "lookup failed", err)
	}
	if existing != nil {
		return existing.ID, nil
	}

	group := &domain.Group{JID: jid, Name: name, CreatedAt: time.Now()}
	if err := uc.groupRepo.Save(ctx, group); err != nil {
		return 0, util.WrapStd(constant.ErrInternal, "save failed", err)
	}
	return group.ID, nil
}
