package usecase

import (
	"context"
	"fmt"

	"whatsup-bot/internal/port"
)

type AddGroupMemberUseCase struct {
	groupRepo port.GroupRepository
}

func NewAddGroupMemberUseCase(groupRepo port.GroupRepository) *AddGroupMemberUseCase {
	return &AddGroupMemberUseCase{groupRepo: groupRepo}
}

// Execute is idempotent — the repository implementation should treat a
// duplicate (groupID, userID) pair as a no-op rather than an error.
func (uc *AddGroupMemberUseCase) Execute(ctx context.Context, groupID, userID int64) error {
	if err := uc.groupRepo.AddMember(ctx, groupID, userID); err != nil {
		return fmt.Errorf("add member failed: %w", err)
	}
	return nil
}
