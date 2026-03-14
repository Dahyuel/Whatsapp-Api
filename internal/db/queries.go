package db

import (
	"context"
	"database/sql"
	"time"
)

// SessionRow represents a row in the sessions table.
type SessionRow struct {
	ID        string
	Status    string
	JID       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UpsertSession inserts or updates a session.
func UpsertSession(db *sql.DB, id, status, jid string) error {
	_, err := db.Exec(`
		INSERT INTO sessions (id, status, jid, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, jid=excluded.jid, updated_at=CURRENT_TIMESTAMP`,
		id, status, jid)
	return err
}

// GetSession returns a session by ID.
func GetSession(db *sql.DB, id string) (*SessionRow, error) {
	row := &SessionRow{}
	err := db.QueryRow(`SELECT id, status, COALESCE(jid,''), created_at, updated_at FROM sessions WHERE id=?`, id).
		Scan(&row.ID, &row.Status, &row.JID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return row, nil
}

// ListSessions returns all sessions.
func ListSessions(db *sql.DB) ([]*SessionRow, error) {
	rows, err := db.Query(`SELECT id, status, COALESCE(jid,''), created_at, updated_at FROM sessions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*SessionRow
	for rows.Next() {
		s := &SessionRow{}
		if err := rows.Scan(&s.ID, &s.Status, &s.JID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// DeleteSession removes a session.
func DeleteSession(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

// UpsertMessageStatus inserts or updates message tracking record.
func UpsertMessageStatus(db *sql.DB, id, sessionID, remoteJID, msgID, status, payload, errMsg string) error {
	_, err := db.Exec(`
		INSERT INTO messages (id, session_id, remote_jid, message_id, status, payload, error, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status, error=excluded.error, updated_at=CURRENT_TIMESTAMP`,
		id, sessionID, remoteJID, msgID, status, payload, errMsg)
	return err
}

// UpdateMessageStatus updates the status of a tracked message.
func UpdateMessageStatus(db *sql.DB, id, status, errMsg string) error {
	_, err := db.Exec(`UPDATE messages SET status=?, error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, status, errMsg, id)
	return err
}

// GetMessageStatus retrieves message status by tracking ID.
func GetMessageStatus(db *sql.DB, id string) (string, error) {
	var status string
	err := db.QueryRow(`SELECT status FROM messages WHERE id=?`, id).Scan(&status)
	return status, err
}

// WebhookRow represents a webhook configuration.
type WebhookRow struct {
	ID        string
	SessionID string
	URL       string
	Secret    string
	Events    string
}

// InsertWebhook inserts a new webhook.
func InsertWebhook(db *sql.DB, id, sessionID, url, secret, events string) error {
	_, err := db.Exec(`INSERT INTO webhooks (id, session_id, url, secret, events) VALUES (?, ?, ?, ?, ?)`,
		id, sessionID, url, secret, events)
	return err
}

// DeleteWebhook deletes a webhook by ID.
func DeleteWebhook(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM webhooks WHERE id=?`, id)
	return err
}

// ListWebhooksForSession returns all webhooks for a session.
func ListWebhooksForSession(db *sql.DB, sessionID string) ([]*WebhookRow, error) {
	rows, err := db.Query(`SELECT id, session_id, url, COALESCE(secret,''), events FROM webhooks WHERE session_id=?`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var webhooks []*WebhookRow
	for rows.Next() {
		w := &WebhookRow{}
		if err := rows.Scan(&w.ID, &w.SessionID, &w.URL, &w.Secret, &w.Events); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

// UpsertCooldown upserts a contact cooldown record.
func UpsertCooldown(ctx context.Context, db *sql.DB, sessionID, jid string, count int, risk string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO contact_cooldowns (session_id, jid, last_sent, msg_count, risk_level)
		VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?)
		ON CONFLICT(session_id, jid) DO UPDATE SET last_sent=CURRENT_TIMESTAMP, msg_count=excluded.msg_count, risk_level=excluded.risk_level`,
		sessionID, jid, count, risk)
	return err
}

// GetCooldown returns the cooldown record for a contact.
func GetCooldown(ctx context.Context, db *sql.DB, sessionID, jid string) (count int, riskLevel string, lastSent time.Time, err error) {
	err = db.QueryRowContext(ctx, `SELECT msg_count, risk_level, last_sent FROM contact_cooldowns WHERE session_id=? AND jid=?`, sessionID, jid).
		Scan(&count, &riskLevel, &lastSent)
	return
}
