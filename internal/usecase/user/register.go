package user

import (
	"context"
	"fmt"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/usecase/message"
	"whatsup-bot/internal/util"
)

type RegisterUseCase struct {
	repo repository.UserRepository
}

func NewRegisterUseCase(repo repository.UserRepository) *RegisterUseCase {
	return &RegisterUseCase{repo: repo}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, jid, name, email string) (string, error) {
	existing, err := uc.repo.FindByJID(ctx, jid)
	if err != nil {
		return "", util.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if existing != nil {
		return "", util.Wrap(constant.ErrAlreadyExists, "user already registered")
	}

	user := &domain.User{
		JID:        jid,
		Name:       name,
		Email:      email,
		RegisterAt: time.Now(),
	}

	if err := uc.repo.Save(ctx, user); err != nil {
		return "", util.WrapStd(constant.ErrInternal, "failed to save user", err)
	}

	return fmt.Sprintf(message.Welcome, name), nil
}
