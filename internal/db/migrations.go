package db

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	id          TEXT PRIMARY KEY,
	status      TEXT NOT NULL DEFAULT 'initializing',
	jid         TEXT,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhooks (
	id          TEXT PRIMARY KEY,
	session_id  TEXT NOT NULL,
	url         TEXT NOT NULL,
	secret      TEXT,
	events      TEXT NOT NULL DEFAULT '[]',
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
	id          TEXT PRIMARY KEY,
	session_id  TEXT NOT NULL,
	remote_jid  TEXT NOT NULL,
	message_id  TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'queued',
	payload     TEXT,
	error       TEXT,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS queue_items (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id  TEXT NOT NULL,
	message_id  TEXT NOT NULL,
	payload     TEXT NOT NULL,
	retries     INTEGER NOT NULL DEFAULT 0,
	scheduled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS contact_cooldowns (
	session_id  TEXT NOT NULL,
	jid         TEXT NOT NULL,
	last_sent   DATETIME NOT NULL,
	msg_count   INTEGER NOT NULL DEFAULT 0,
	risk_level  TEXT NOT NULL DEFAULT 'normal',
	PRIMARY KEY (session_id, jid),
	FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_queue_session ON queue_items(session_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_session ON webhooks(session_id);
CREATE INDEX IF NOT EXISTS idx_cooldowns_session ON contact_cooldowns(session_id);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
