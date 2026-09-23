package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"whatsup-bot/internal/utils"
)

// Open opens (and initializes, if needed) the app's SQLite database at path.
// Foreign keys are enforced and writer waits are bounded via DSN params.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, utils.Wrap(err, "open sqlite db")
	}

	if _, err := db.Exec(initSchemaQuery); err != nil {
		db.Close()
		return nil, utils.Wrap(err, "init schema")
	}

	return db, nil
}
