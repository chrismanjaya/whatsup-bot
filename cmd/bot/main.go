package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"whatsup-bot/internal/config"
	"whatsup-bot/internal/external/whatsapp"
	"whatsup-bot/internal/util"
)

func main() {
	ctx := context.Background()

	util.SetupLogger()

	cfg, err := config.Load()
	if err != nil {
		util.Fatal("load config failed", err)
	}

	client, err := build(ctx, cfg)
	if err != nil {
		util.Fatal("build app failed", err)
	}

	if err := whatsapp.Connect(ctx, client); err != nil {
		util.Fatal("connect failed", err)
	}
	defer client.Disconnect()

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
