package sqlite

const (
	initSchemaQuery = `
		CREATE TABLE IF NOT EXISTS users (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			jid         TEXT NOT NULL UNIQUE,
			name        TEXT NOT NULL,
			email       TEXT,
			register_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS groups (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			jid        TEXT NOT NULL UNIQUE,
			name       TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS group_members (
			group_id INTEGER NOT NULL,
			user_id  INTEGER NOT NULL,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id)  REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS transactions (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id          INTEGER NOT NULL,
			description      TEXT NOT NULL,
			type             TEXT NOT NULL CHECK (type IN ('CR', 'DB')),
			amount           INTEGER NOT NULL,
			category         TEXT,
			is_shared        INTEGER NOT NULL DEFAULT 0,
			group_id         INTEGER,
			transaction_date DATETIME NOT NULL,
			created_at       DATETIME NOT NULL,
			wa_message_id    TEXT,
			FOREIGN KEY (user_id)  REFERENCES users(id),
			FOREIGN KEY (group_id) REFERENCES groups(id)
		);

		CREATE INDEX IF NOT EXISTS idx_transactions_user_date  ON transactions(user_id, transaction_date);
		CREATE INDEX IF NOT EXISTS idx_transactions_group_date ON transactions(group_id, transaction_date);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_wa_message_id ON transactions(wa_message_id);
		`
	insertGroupQuery = ``
)
