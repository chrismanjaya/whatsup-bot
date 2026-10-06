package group

import (
	"context"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/util"
)

type EnsureMembershipUseCase struct {
	userRepo  repository.UserRepository
	groupRepo repository.GroupRepository
}

func NewEnsureMembershipUseCase(userRepo repository.UserRepository, groupRepo repository.GroupRepository) *EnsureMembershipUseCase {
	return &EnsureMembershipUseCase{userRepo: userRepo, groupRepo: groupRepo}
}

// Execute is a no-op if the sender isn't registered yet — nothing to track.
func (uc *EnsureMembershipUseCase) Execute(ctx context.Context, senderJID string, groupID int64) error {
	user, err := uc.userRepo.FindByJID(ctx, senderJID)
	if err != nil {
		return util.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if user == nil {
		return nil
	}
	if err := uc.groupRepo.AddMember(ctx, groupID, user.ID); err != nil {
		return util.WrapStd(constant.ErrInternal, "add member failed", err)
	}
	return nil
}
