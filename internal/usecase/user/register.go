package user

import (
	"context"
	"strings"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type Register struct {
	userRepo UserRepository
}

func NewRegister(userRepo UserRepository) *Register {
	return &Register{userRepo: userRepo}
}

func (uc *Register) Execute(ctx context.Context, jid, rawText string) (*domain.User, error) {
	// find existing user
	existing, err := uc.userRepo.FindByJID(ctx, jid)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}

	if existing != nil {
		return existing, nil
	}

	// register new user
	newUser := constractUser(jid, rawText)
	if newUser == nil {
		return nil, util.WrapStd(constant.ErrInternal, "invalid user message", nil)
	}

	if err = uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "failed to create user", err)
	}

	return newUser, nil
}

func constractUser(jid, rawText string) *domain.User {
	parts := strings.Fields(rawText)
	if len(parts) < 3 {
		return nil
	}

	return &domain.User{
		JID:   jid,
		Name:  parts[1],
		Email: parts[2],
	}
}
