package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"example.com/app/internal/config"
	httpdelivery "example.com/app/internal/infra/delivery/http"
	"example.com/app/internal/infra/storage/sqldb"
	examplerepo "example.com/app/internal/infra/storage/sqldb/example"
	exampleuc "example.com/app/internal/usecase/example"
	"example.com/app/internal/util"
	// Register your database driver here, e.g.:
	// _ "github.com/mattn/go-sqlite3"
)

// App is everything main needs to run and stop the process.
type App struct {
	Server  *http.Server
	closers []func() error
}

// Close releases resources in reverse order of creation.
func (a *App) Close() {
	for i := len(a.closers) - 1; i >= 0; i-- {
		if err := a.closers[i](); err != nil {
			slog.Error("close resource failed", "error", err)
		}
	}
}

// Build is the composition root: the only place that knows concrete types.
// Order: storage clients -> repositories -> external clients -> usecases ->
// delivery.
func Build(ctx context.Context, cfg config.Config) (*App, error) {
	app := &App{}

	// setup storage clients
	dbClient, err := sqldb.NewClient(ctx, sqldb.Config{
		Driver: cfg.DBDriver,
		DSN:    cfg.DBDSN,
	})
	if err != nil {
		return nil, util.Wrap(err, "init database failed")
	}
	app.closers = append(app.closers, dbClient.Close)

	if _, err := dbClient.DB().ExecContext(ctx, sqldb.SchemaQuery); err != nil {
		app.Close()
		return nil, util.Wrap(err, "init schema failed")
	}

	// setup repositories
	exampleRepo := examplerepo.NewRepo(dbClient.DB())

	// setup usecase modules
	exampleModule := exampleuc.NewModule(exampleRepo)

	// setup delivery
	handler := httpdelivery.NewHandler(exampleModule.CreateUsecase())

	app.Server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           http.TimeoutHandler(handler.Routes(), cfg.RequestTimeout, "request timeout"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		WriteTimeout:      cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return app, nil
}
