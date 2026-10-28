package main

import (
	"context"

	"whatsupbot/internal/config"
	mcpHandler "whatsupbot/internal/infra/delivery/mcp"
	"whatsupbot/internal/infra/service"
	"whatsupbot/internal/infra/storage/sqlite"

	mcpTokenRepo "whatsupbot/internal/infra/storage/sqlite/mcptoken"
	transactionRepo "whatsupbot/internal/infra/storage/sqlite/transaction"
	userRepo "whatsupbot/internal/infra/storage/sqlite/user"

	integrationusecase "whatsupbot/internal/usecase/integration"
	reportusecase "whatsupbot/internal/usecase/report"

	"whatsupbot/internal/util"
)

func Build(ctx context.Context, cfg config.MCPConfig) (*mcpHandler.Handler, error) {
	// setup persistence clients
	sqliteClient, err := sqlite.NewClient(ctx, sqlite.Config{
		DBPath: cfg.AppDBPath,
	})
	if err != nil {
		return nil, util.Wrap(err, "init sqlite failed")
	}

	// setup repository
	userRepo := userRepo.NewRepo(sqliteClient.Client())
	transactionRepo := transactionRepo.NewRepo(sqliteClient.Client())
	mcpTokenRepo := mcpTokenRepo.NewRepo(sqliteClient.Client())

	// setup usecase module
	reportModule := reportusecase.NewModule(
		transactionRepo,
		userRepo,
		nil, // the mcp server does not parse chat messages
	)

	integrationModule := integrationusecase.NewModule(
		mcpTokenRepo,
		userRepo,
		"", // the mcp server does not issue tokens
	)

	// setup service
	mcpService := service.NewMCP(
		integrationModule.AuthenticateMCPTokenUsecase(),
		reportModule.GetTransactionListUsecase(),
	)

	// setup delivery
	handler := mcpHandler.NewHandler(
		mcpHandler.Config{
			Addr: cfg.Addr,
		},
		mcpService,
	)

	return handler, nil
}
