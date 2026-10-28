package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
)

type fakeMCPTokenRepo struct {
	byUser map[int64]string // user id -> token hash
}

func (r *fakeMCPTokenRepo) Upsert(ctx context.Context, token *domain.MCPToken) error {
	r.byUser[token.UserID] = token.TokenHash
	return nil
}

func (r *fakeMCPTokenRepo) FindByHash(ctx context.Context, tokenHash string) (*domain.MCPToken, error) {
	for userID, hash := range r.byUser {
		if hash == tokenHash {
			return &domain.MCPToken{UserID: userID, TokenHash: hash}, nil
		}
	}
	return nil, nil
}

type fakeUserRepo struct{}

func (fakeUserRepo) FindByID(ctx context.Context, userID int64) (*domain.User, error) {
	return &domain.User{ID: userID}, nil
}

func TestMCPToken(t *testing.T) {
	ctx := context.Background()
	repo := &fakeMCPTokenRepo{byUser: map[int64]string{}}
	module := NewModule(repo, fakeUserRepo{}, "https://example.com/mcp")
	user := &domain.User{ID: 7}

	first, err := module.GenerateMCPTokenUsecase().Execute(ctx, user, false)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(first.Token, mcpTokenPrefix) || first.MCPURL != "https://example.com/mcp" {
		t.Fatalf("unexpected result: %+v", first)
	}
	if repo.byUser[7] == first.Token {
		t.Fatal("plain token must not be stored")
	}

	got, err := module.AuthenticateMCPTokenUsecase().Execute(ctx, first.Token)
	if err != nil || got.ID != 7 {
		t.Fatalf("authenticate = %+v, %v; want user 7", got, err)
	}

	// regenerating revokes the previous token
	if _, err := module.GenerateMCPTokenUsecase().Execute(ctx, user, false); err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if _, err := module.AuthenticateMCPTokenUsecase().Execute(ctx, first.Token); !errors.Is(err, constant.ErrInvalidMCPToken) {
		t.Fatalf("old token err = %v, want ErrInvalidMCPToken", err)
	}

	if _, err := module.GenerateMCPTokenUsecase().Execute(ctx, user, true); !errors.Is(err, constant.ErrPrivateChatOnly) {
		t.Fatalf("group err = %v, want ErrPrivateChatOnly", err)
	}
}
