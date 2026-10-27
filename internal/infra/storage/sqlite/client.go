package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"whatsupbot/internal/util"

	_ "github.com/mattn/go-sqlite3"
)

type (
	Client struct {
		sqliteClient *sql.DB
		config       Config
	}
	Config struct {
		DBPath string
	}
)

const (
	driverName  = "sqlite3"
	busyTimeout = 5000

	dataSourceNameTemplate = "file:%s?_foreign_keys=on&_busy_timeout=%d"
)

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	dataSourceName := fmt.Sprintf(dataSourceNameTemplate, cfg.DBPath, busyTimeout)

	// open connection
	sqliteClient, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, util.Wrap(err, "open sqlite failed")
	}

	// init schema
	if _, err := sqliteClient.Exec(initSchemaQuery); err != nil {
		sqliteClient.Close()
		return nil, util.Wrap(err, "init schema failed")
	}

	return &Client{
		sqliteClient: sqliteClient,
		config:       cfg,
	}, nil
}

func (s *Client) Client() *sql.DB {
	return s.sqliteClient
}
