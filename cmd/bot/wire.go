package main

import (
	"context"

	"go.mau.fi/whatsmeow"
	"google.golang.org/genai"

	"whatsup-bot/internal/config"
	"whatsup-bot/internal/external/database"
	"whatsup-bot/internal/external/gemini"
	"whatsup-bot/internal/external/whatsapp"
	"whatsup-bot/internal/usecase/group"
	"whatsup-bot/internal/usecase/persona"
	"whatsup-bot/internal/usecase/transaction"
	"whatsup-bot/internal/usecase/user"
	"whatsup-bot/internal/util"
)

// build is the composition root: it constructs every adapter and usecase and
// connects them. This package is the only place allowed to import concrete
// adapters.
func build(ctx context.Context, cfg config.Config) (*whatsmeow.Client, error) {
	appDB, err := database.Open(cfg.AppDBPath)
	if err != nil {
		return nil, util.Wrap(err, "open app db")
	}

	userRepo := database.NewUserRepo(appDB)
	txRepo := database.NewTransactionRepo(appDB)
	groupRepo := database.NewGroupRepo(appDB)
	pageRepo := database.NewQueryPageRepo(appDB)

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, util.Wrap(err, "create gemini client")
	}
	parser := gemini.NewParser(geminiClient, cfg.GeminiModel)

	client, err := whatsapp.NewClient(ctx, cfg.WAStorePath)
	if err != nil {
		return nil, err
	}

	picker := persona.New()

	whatsapp.NewHandler(
		client,
		txRepo,
		pageRepo,
		user.NewRegisterUseCase(userRepo),
		transaction.NewRecordUseCase(userRepo, txRepo, parser, picker),
		transaction.NewAmendUseCase(userRepo, txRepo, parser, picker),
		transaction.NewSummarizeUseCase(userRepo, txRepo, parser),
		transaction.NewQueryUseCase(userRepo, txRepo, pageRepo, parser),
		transaction.NewPageUseCase(userRepo, txRepo, pageRepo),
		group.NewEnsureUseCase(groupRepo),
		group.NewEnsureMembershipUseCase(userRepo, groupRepo),
		group.NewComputeSplitUseCase(groupRepo, txRepo),
		picker,
	).Register()

	return client, nil
}
