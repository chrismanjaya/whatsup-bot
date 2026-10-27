package sqldb

// SchemaQuery is the bootstrap schema. Once real data exists, replace it with
// versioned migrations (e.g. golang-migrate or goose) under /migrations.
const SchemaQuery = `
	CREATE TABLE IF NOT EXISTS examples (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL UNIQUE,
		status     TEXT NOT NULL CHECK (status IN ('active', 'archived')),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
`
