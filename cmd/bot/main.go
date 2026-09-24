package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"whatsup-bot/internal/adapter/whatsapp"
	"whatsup-bot/internal/config"
	"whatsup-bot/internal/utils"
)

func main() {
	ctx := context.Background()

	utils.SetupLogger()

	cfg, err := config.Load()
	if err != nil {
		utils.Fatal("load config failed", err)
	}

	client, err := build(ctx, cfg)
	if err != nil {
		utils.Fatal("build app failed", err)
	}

	if err := whatsapp.Connect(ctx, client); err != nil {
		utils.Fatal("connect failed", err)
	}
	defer client.Disconnect()

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
