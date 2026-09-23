package usecase

import (
	"context"
	"fmt"
	"time"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/port"
	"whatsup-bot/internal/utils"
)

type RegisterUserUseCase struct {
	repo port.UserRepository
}

func NewRegisterUserUseCase(repo port.UserRepository) *RegisterUserUseCase {
	return &RegisterUserUseCase{repo: repo}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, jid, name, email string) (string, error) {
	existing, err := uc.repo.FindByJID(ctx, jid)
	if err != nil {
		return "", utils.WrapStd(constant.ErrInternal, "lookup user failed", err)
	}
	if existing != nil {
		return "", utils.Wrap(constant.ErrAlreadyExists, "user already registered")
	}

	user := &domain.User{
		JID:        jid,
		Name:       name,
		Email:      email,
		RegisterAt: time.Now(),
	}

	if err := uc.repo.Save(ctx, user); err != nil {
		return "", utils.WrapStd(constant.ErrInternal, "failed to save user", err)
	}

	return fmt.Sprintf("Welcome, %s! You can now log your income/expenses.", name), nil
}
