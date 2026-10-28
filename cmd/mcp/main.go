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

	// load config
	cfg, err := config.LoadMCP()
	if err != nil {
		util.Fatal("load config failed", err)
	}

	// build server
	handler, err := Build(ctx, cfg)
	if err != nil {
		util.Fatal("build app failed", err)
	}

	// shutdown on signal
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		if err := handler.Shutdown(ctx); err != nil {
			util.LogError("shutdown mcp failed", err)
		}
	}()

	fmt.Printf("MCP server is running on %s/mcp. Press Ctrl+C to exit.\n", cfg.Addr)

	if err := handler.Serve(); err != nil {
		util.Fatal("serve mcp failed", err)
	}
}
