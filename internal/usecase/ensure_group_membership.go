package usecase

import (
	"context"
	"fmt"

	"whatsup-bot/internal/port"
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
		return fmt.Errorf("lookup user failed: %w", err)
	}
	if user == nil {
		return nil
	}
	if err := uc.groupRepo.AddMember(ctx, groupID, user.ID); err != nil {
		return fmt.Errorf("add member failed: %w", err)
	}
	return nil
}
