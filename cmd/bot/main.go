package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"whatsupbot/internal/config"
	"whatsupbot/internal/util"
)

func main() {
	ctx := context.Background()

	// setup logger
	util.SetupLogger()

	// init template
	util.InitTemplate()

	// load config
	cfg, err := config.Load()
	if err != nil {
		util.Fatal("load config failed", err)
	}

	// build client
	handler, err := Build(ctx, cfg)
	if err != nil {
		util.Fatal("build app failed", err)
	}

	// api connect
	if err := handler.Connect(ctx); err != nil {
		util.Fatal("connect api failed", err)
	}
	defer handler.Disconnect()

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}
