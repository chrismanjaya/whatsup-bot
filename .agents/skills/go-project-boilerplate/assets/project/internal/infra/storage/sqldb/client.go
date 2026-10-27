// Package sqldb opens and owns the database/sql connection pool. Repositories
// live in one subpackage per aggregate and receive the *sql.DB from wire.go.
package sqldb

import (
	"context"
	"database/sql"
	"time"

	"example.com/app/internal/util"
)

type (
	Client struct {
		db *sql.DB
	}
	Config struct {
		Driver string
		DSN    string
	}
)

const (
	maxOpenConns    = 10
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
	pingTimeout     = 5 * time.Second
)

// NewClient opens the pool and verifies it with a ping. The driver must be
// registered by a blank import in cmd/wire.go.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, util.Wrap(err, "open database failed")
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, util.Wrap(err, "ping database failed")
	}

	return &Client{db: db}, nil
}

func (c *Client) DB() *sql.DB {
	return c.db
}

func (c *Client) Close() error {
	return c.db.Close()
}
