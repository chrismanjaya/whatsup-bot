package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"example.com/app/internal/config"
	"example.com/app/internal/util"
)

const shutdownTimeout = 15 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// load config
	cfg, err := config.Load()
	if err != nil {
		util.Fatal("load config failed", err)
	}

	// setup logger
	util.SetupLogger(cfg.LogLevel, cfg.LogFormat)

	// build app
	app, err := Build(ctx, cfg)
	if err != nil {
		util.Fatal("build app failed", err)
	}
	defer app.Close()

	// serve until a shutdown signal arrives
	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		serveErr <- app.Server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			util.Fatal("server failed", err)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	// drain in-flight requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
