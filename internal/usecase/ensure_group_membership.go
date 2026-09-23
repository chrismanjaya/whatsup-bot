package usecase

import (
	"context"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type EnsureGroupMembershipUseCase struct {
	userRepo  port.UserRepository
	groupRepo port.GroupRepository
}

func NewEnsureGroupMembershipUseCase(userRepo port.UserRepository, groupRepo port.GroupRepository) *EnsureGroupMembershipUseCase {
	return &EnsureGroupMembershipUseCase{userRepo: userRepo, groupRepo: groupRepo}
}

// Execute is a no-op if the sender isn't registered yet — nothing to track.
func (uc *EnsureGroupMembershipUseCase) Execute(ctx context.Context, senderJID string, groupID int64) error {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return nil
	}
	if err := uc.groupRepo.AddMember(ctx, groupID, user.ID); err != nil {
		return utils.WrapStd(constant.ErrInternal, "add member failed", err)
	}
	return nil
}
