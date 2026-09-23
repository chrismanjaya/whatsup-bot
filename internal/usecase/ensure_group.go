package usecase

import (
	"context"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type EnsureGroupUseCase struct {
	groupRepo port.GroupRepository
}

func NewEnsureGroupUseCase(groupRepo port.GroupRepository) *EnsureGroupUseCase {
	return &EnsureGroupUseCase{groupRepo: groupRepo}
}

// Execute returns the group's internal ID, creating the group on first sight.
func (uc *EnsureGroupUseCase) Execute(ctx context.Context, jid, name string) (int64, error) {
	existing, err := uc.groupRepo.FindByJID(ctx, jid)
	if err != nil {
		return 0, utils.WrapStd(constant.ErrInternal, "lookup failed", err)
	}
	if existing != nil {
		return existing.ID, nil
	}

	group := &domain.Group{JID: jid, Name: name, CreatedAt: time.Now()}
	if err := uc.groupRepo.Save(ctx, group); err != nil {
		return 0, utils.WrapStd(constant.ErrInternal, "save failed", err)
	}
	return group.ID, nil
}
