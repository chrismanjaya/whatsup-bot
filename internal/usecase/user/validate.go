package user

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Validate struct {
	userRepo UserRepository
}

func NewValidate(userRepo UserRepository) *Validate {
	return &Validate{userRepo: userRepo}
}

func (uc *Validate) Execute(ctx context.Context, jid string) (*domain.User, error) {
	user, err := uc.userRepo.FindByJID(ctx, jid)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInvalidUser, "lookup user failed", err)
	}

	if user == nil {
		return nil, util.Wrap(constant.ErrUserNotRegister, "user not register")
	}

	return user, nil
}
