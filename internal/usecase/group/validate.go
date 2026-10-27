package group

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Validate struct {
	groupRepo GroupRepository
	userRepo  UserRepository
}

func NewValidate(groupRepo GroupRepository, userRepo UserRepository) *Validate {
	return &Validate{
		groupRepo: groupRepo,
		userRepo:  userRepo,
	}
}

func (uc *Validate) Execute(ctx context.Context, jid string, groupName string) (*domain.Group, error) {
	group, err := uc.getGroup(ctx, jid, groupName)
	if err != nil {
		return nil, util.Wrap(err, "lookup group failed")
	}

	if err = uc.addMember(ctx, jid, group.ID); err != nil {
		return nil, util.Wrap(err, "add member failed")
	}

	return group, nil
}

func (uc *Validate) getGroup(ctx context.Context, jid, name string) (*domain.Group, error) {
	// check existing group
	existing, err := uc.groupRepo.FindByJID(ctx, jid)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "lookup group failed", err)
	}

	if existing != nil {
		return existing, nil
	}

	// not found, create new group
	newGroup, err := uc.createGroup(ctx, jid, name)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "create group failed", err)
	}

	return newGroup, nil
}

func (uc *Validate) createGroup(ctx context.Context, jid, name string) (*domain.Group, error) {
	newGroup := &domain.Group{
		JID:  jid,
		Name: name,
	}

	if err := uc.groupRepo.Create(ctx, newGroup); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "create group failed", err)
	}

	return newGroup, nil
}

func (uc *Validate) addMember(ctx context.Context, jid string, groupID int64) error {
	user, err := uc.userRepo.FindByJID(ctx, jid)
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
