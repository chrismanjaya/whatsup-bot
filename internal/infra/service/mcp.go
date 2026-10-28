package service

import (
	"context"
	"whatsupbot/internal/domain"
	integrationuc "whatsupbot/internal/usecase/integration"
	reportuc "whatsupbot/internal/usecase/report"
)

type MCP struct {
	// integration usecase
	authenticateMCPToken *integrationuc.AuthenticateMCPToken

	// report usecase
	getTransactionList *reportuc.GetTransactionList
}

func NewMCP(
	authenticateMCPToken *integrationuc.AuthenticateMCPToken,
	getTransactionList *reportuc.GetTransactionList,
) *MCP {
	return &MCP{
		authenticateMCPToken: authenticateMCPToken,
		getTransactionList:   getTransactionList,
	}
}

// Authenticate returns the user that owns the MCP token.
func (s *MCP) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	return s.authenticateMCPToken.Execute(ctx, token)
}

func (s *MCP) GetTransactionList(ctx context.Context, req reportuc.TransactionListRequest) (*reportuc.TransactionListResult, error) {
	return s.getTransactionList.Execute(ctx, req)
}
