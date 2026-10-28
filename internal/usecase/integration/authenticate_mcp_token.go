package integration

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type AuthenticateMCPToken struct {
	mcpTokenRepo MCPTokenRepository
	userRepo     UserRepository
}

func NewAuthenticateMCPToken(mcpTokenRepo MCPTokenRepository, userRepo UserRepository) *AuthenticateMCPToken {
	return &AuthenticateMCPToken{
		mcpTokenRepo: mcpTokenRepo,
		userRepo:     userRepo,
	}
}

// Execute returns the user that owns token.
func (uc *AuthenticateMCPToken) Execute(ctx context.Context, token string) (*domain.User, error) {
	if token == "" {
		return nil, util.Wrap(constant.ErrInvalidMCPToken, "empty mcp token")
	}

	mcpToken, err := uc.mcpTokenRepo.FindByHash(ctx, hashMCPToken(token))
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "find mcp token failed", err)
	}

	if mcpToken == nil {
		return nil, util.Wrap(constant.ErrInvalidMCPToken, "mcp token not found")
	}

	user, err := uc.userRepo.FindByID(ctx, mcpToken.UserID)
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "find user failed", err)
	}

	if user == nil {
		return nil, util.Wrap(constant.ErrInvalidMCPToken, "mcp token user not found")
	}

	return user, nil
}
