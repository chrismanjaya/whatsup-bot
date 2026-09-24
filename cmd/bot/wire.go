package main

import (
	"context"

	"go.mau.fi/whatsmeow"
	"google.golang.org/genai"

	"whatsup-bot/internal/adapter/gemini"
	"whatsup-bot/internal/adapter/sqlite"
	"whatsup-bot/internal/adapter/whatsapp"
	"whatsup-bot/internal/config"
	"whatsup-bot/internal/usecase"
	"whatsup-bot/internal/utils"
)

// build is the composition root: it constructs every adapter and usecase and
// connects them. This package is the only place allowed to import concrete
// adapters.
func build(ctx context.Context, cfg config.Config) (*whatsmeow.Client, error) {
	appDB, err := sqlite.Open(cfg.AppDBPath)
	if err != nil {
		return nil, utils.Wrap(err, "open app db")
	}

	userRepo := sqlite.NewUserRepo(appDB)
	txRepo := sqlite.NewTransactionRepo(appDB)
	groupRepo := sqlite.NewGroupRepo(appDB)
	pageRepo := sqlite.NewQueryPageRepo(appDB)

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, utils.Wrap(err, "create gemini client")
	}
	parser := gemini.NewParser(geminiClient, cfg.GeminiModel)

	client, err := whatsapp.NewClient(ctx, cfg.WAStorePath)
	if err != nil {
		return nil, err
	}

	whatsapp.NewHandler(
		client,
		txRepo,
		pageRepo,
		usecase.NewRegisterUserUseCase(userRepo),
		usecase.NewRecordTransactionUseCase(userRepo, txRepo, parser),
		usecase.NewAmendTransactionUseCase(userRepo, txRepo, parser),
		usecase.NewQueryTransactionsUseCase(userRepo, txRepo, pageRepo, parser),
		usecase.NewPageTransactionsUseCase(userRepo, txRepo, pageRepo),
		usecase.NewEnsureGroupUseCase(groupRepo),
		usecase.NewEnsureGroupMembershipUseCase(userRepo, groupRepo),
		usecase.NewComputeSplitUseCase(groupRepo, txRepo),
	).Register()

	return client, nil
}
