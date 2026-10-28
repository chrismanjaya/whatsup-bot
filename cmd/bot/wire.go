package main

import (
	"context"

	"whatsupbot/internal/config"
	whatsappHandler "whatsupbot/internal/infra/delivery/whatsapp"

	"whatsupbot/internal/infra/external/gemini"
	"whatsupbot/internal/infra/external/whatsapp"
	"whatsupbot/internal/infra/service"
	"whatsupbot/internal/infra/storage/sqlite"

	groupRepo "whatsupbot/internal/infra/storage/sqlite/group"
	mcpTokenRepo "whatsupbot/internal/infra/storage/sqlite/mcptoken"
	transactionRepo "whatsupbot/internal/infra/storage/sqlite/transaction"
	userRepo "whatsupbot/internal/infra/storage/sqlite/user"

	groupusecase "whatsupbot/internal/usecase/group"
	integrationusecase "whatsupbot/internal/usecase/integration"
	personausecase "whatsupbot/internal/usecase/persona"
	reportusecase "whatsupbot/internal/usecase/report"
	transactionusecase "whatsupbot/internal/usecase/transaction"
	userusecase "whatsupbot/internal/usecase/user"

	"whatsupbot/internal/util"

	"google.golang.org/genai"
)

func Build(ctx context.Context, cfg config.Config) (*whatsappHandler.Handler, error) {
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
	groupRepo := groupRepo.NewRepo(sqliteClient.Client())
	mcpTokenRepo := mcpTokenRepo.NewRepo(sqliteClient.Client())

	// setup external clients
	whatsappClient, err := whatsapp.NewClient(ctx, whatsapp.Config{
		StorePath: cfg.WAStorePath,
	})
	if err != nil {
		return nil, util.Wrap(err, "init whatsapp failed")
	}

	geminiClient, err := gemini.NewClient(ctx, gemini.Config{
		ClientConfig: &genai.ClientConfig{
			APIKey:  cfg.GeminiAPIKey,
			Backend: genai.BackendGeminiAPI,
		},
		Model: cfg.GeminiModel,
	})
	if err != nil {
		return nil, util.Wrap(err, "init gemini failed")
	}

	// setup usecase module
	transactionModule := transactionusecase.NewModule(
		groupRepo,
		transactionRepo,
		userRepo,
		geminiClient,
	)

	groupModule := groupusecase.NewModule(
		groupRepo,
		userRepo,
	)

	userModule := userusecase.NewModule(userRepo)

	reportModule := reportusecase.NewModule(
		transactionRepo,
		userRepo,
		geminiClient,
	)

	integrationModule := integrationusecase.NewModule(
		mcpTokenRepo,
		userRepo,
		cfg.MCPURL,
	)

	personaModule := personausecase.NewModule()

	// setup service
	recordBatcher := service.NewRecordBatcher(
		transactionModule.RecordBatchUsecase(),
		personaModule.GetPickerUsecase(),
		service.RecordBatcherConfig{
			FlushInterval: cfg.RecordBatchInterval,
			MaxBatchSize:  cfg.RecordBatchMaxSize,
		},
	)

	chatbotService := service.NewChatbot(
		groupModule.ValidateUsecase(),
		transactionModule.RecordUsecase(),
		transactionModule.AmendUsecase(),
		transactionModule.SplitUsecase(),
		recordBatcher,
		userModule.RegisterUsecase(),
		userModule.ValidateUsecase(),
		reportModule.GetDetailUsecase(),
		reportModule.GetSummarylUsecase(),
		integrationModule.GenerateMCPTokenUsecase(),
		personaModule.GetPickerUsecase(),
	)

	// setup delivery
	handler := whatsappHandler.NewHandler(
		whatsappClient,
		chatbotService,
	)

	handler.Register()

	return handler, nil
}
