package integration

import (
	"context"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type GenerateMCPToken struct {
	mcpTokenRepo MCPTokenRepository
	mcpURL       string
}

func NewGenerateMCPToken(mcpTokenRepo MCPTokenRepository, mcpURL string) *GenerateMCPToken {
	return &GenerateMCPToken{
		mcpTokenRepo: mcpTokenRepo,
		mcpURL:       mcpURL,
	}
}

type GenerateMCPTokenResult struct {
	Token  string
	MCPURL string
}

// Execute issues a new MCP token for user, replacing any token issued before.
func (uc *GenerateMCPToken) Execute(ctx context.Context, user *domain.User, isGroup bool) (*GenerateMCPTokenResult, error) {
	if user == nil || user.ID <= 0 {
		return nil, util.Wrap(constant.ErrInvalidUser, "invalid user")
	}

	// a token sent to a group would be readable by every member
	if isGroup {
		return nil, util.Wrap(constant.ErrPrivateChatOnly, "mcp token requested in group")
	}

	token, err := newMCPToken()
	if err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "generate mcp token failed", err)
	}

	if err := uc.mcpTokenRepo.Upsert(ctx, &domain.MCPToken{
		UserID:    user.ID,
		TokenHash: hashMCPToken(token),
	}); err != nil {
		return nil, util.WrapStd(constant.ErrInternal, "save mcp token failed", err)
	}

	return &GenerateMCPTokenResult{
		Token:  token,
		MCPURL: uc.mcpURL,
	}, nil
}
