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

// ── User management ────────────────────────────────────────────────────────

// UserRow represents a row in the users table.
type UserRow struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

// CreateUser inserts a new user.
func CreateUser(db *sql.DB, id, username, passwordHash, role string) error {
	_, err := db.Exec(`INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)`,
		id, username, passwordHash, role)
	return err
}

// GetUserByUsername returns a user by username.
func GetUserByUsername(db *sql.DB, username string) (*UserRow, error) {
	u := &UserRow{}
	err := db.QueryRow(`SELECT id, username, password_hash, role, created_at FROM users WHERE username=?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByID returns a user by ID.
func GetUserByID(db *sql.DB, id string) (*UserRow, error) {
	u := &UserRow{}
	err := db.QueryRow(`SELECT id, username, password_hash, role, created_at FROM users WHERE id=?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ListUsers returns all users.
func ListUsers(db *sql.DB) ([]*UserRow, error) {
	rows, err := db.Query(`SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*UserRow
	for rows.Next() {
		u := &UserRow{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ListAgents returns all users with role='agent'.
func ListAgents(db *sql.DB) ([]*UserRow, error) {
	rows, err := db.Query(`SELECT id, username, password_hash, role, created_at FROM users WHERE role='agent' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*UserRow
	for rows.Next() {
		u := &UserRow{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// DeleteUser deletes a user by ID.
func DeleteUser(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

// CountUsers returns the total number of users matching role (empty string = all).
func CountUsers(db *sql.DB, role string) (int, error) {
	var count int
	var err error
	if role == "" {
		err = db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	} else {
		err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE role=?`, role).Scan(&count)
	}
	return count, err
}

// ── Agent chat assignment ──────────────────────────────────────────────────

// AgentChatRow represents a row in the agent_chats table.
type AgentChatRow struct {
	ID         string
	AgentID    string
	SessionID  string
	JID        string
	Name       string // Added to join from chats
	AssignedAt time.Time
}

// AssignChatToAgent creates an assignment record.
func AssignChatToAgent(db *sql.DB, id, agentID, sessionID, jid string) error {
	_, err := db.Exec(`
		INSERT INTO agent_chats (id, agent_id, session_id, jid)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(agent_id, session_id, jid) DO NOTHING`,
		id, agentID, sessionID, jid)
	return err
}

// UnassignChat removes an assignment by ID.
func UnassignChat(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM agent_chats WHERE id=?`, id)
	return err
}

// UnassignChatByJID removes assignment by agent+session+jid.
func UnassignChatByJID(db *sql.DB, agentID, sessionID, jid string) error {
	_, err := db.Exec(`DELETE FROM agent_chats WHERE agent_id=? AND session_id=? AND jid=?`, agentID, sessionID, jid)
	return err
}

// GetChatsForAgent returns all chat assignments for an agent along with chat names.
func GetChatsForAgent(db *sql.DB, agentID string) ([]*AgentChatRow, error) {
	rows, err := db.Query(`
		SELECT a.id, a.agent_id, a.session_id, a.jid, a.assigned_at, COALESCE(c.name, '')
		FROM agent_chats a
		LEFT JOIN chats c ON a.session_id = c.session_id AND a.jid = c.jid
		WHERE a.agent_id=? 
		ORDER BY a.assigned_at
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []*AgentChatRow
	for rows.Next() {
		ac := &AgentChatRow{}
		if err := rows.Scan(&ac.ID, &ac.AgentID, &ac.SessionID, &ac.JID, &ac.AssignedAt, &ac.Name); err != nil {
			return nil, err
		}
		chats = append(chats, ac)
	}
	return chats, nil
}

// GetAgentForChat returns the agent assigned to a given session+jid, or nil if none.
func GetAgentForChat(db *sql.DB, sessionID, jid string) (*AgentChatRow, error) {
	ac := &AgentChatRow{}
	err := db.QueryRow(`SELECT id, agent_id, session_id, jid, assigned_at FROM agent_chats WHERE session_id=? AND jid=?`, sessionID, jid).
		Scan(&ac.ID, &ac.AgentID, &ac.SessionID, &ac.JID, &ac.AssignedAt)
	if err != nil {
		return nil, err
	}
	return ac, nil
}

// GetRandomAgent returns a random agent user, used for auto-assignment.
func GetRandomAgent(db *sql.DB) (*UserRow, error) {
	u := &UserRow{}
	err := db.QueryRow(`SELECT id, username, password_hash, role, created_at FROM users WHERE role='agent' ORDER BY RANDOM() LIMIT 1`).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ── Chat & Message History ─────────────────────────────────────────────────

// ChatRow represents a conversation.
type ChatRow struct {
	SessionID       string
	JID             string
	Name            string
	UnreadCount     int
	LastMessageTime time.Time
}

// UpsertChat creates or updates a chat conversation record.
func UpsertChat(db *sql.DB, sessionID, jid, name string, unreadInc int, lastMsgTime time.Time) error {
	_, err := db.Exec(`
		INSERT INTO chats (session_id, jid, name, unread_count, last_message_time)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(session_id, jid) DO UPDATE SET
			name = CASE WHEN excluded.name != '' AND chats.name = '' THEN excluded.name ELSE chats.name END,
			unread_count = chats.unread_count + excluded.unread_count,
			last_message_time = CASE WHEN excluded.last_message_time > chats.last_message_time THEN excluded.last_message_time ELSE chats.last_message_time END
	`, sessionID, jid, name, unreadInc, lastMsgTime)
	return err
}

// ListChats returns all chats for a given session.
func ListChats(db *sql.DB, sessionID string) ([]*ChatRow, error) {
	rows, err := db.Query(`SELECT session_id, jid, name, unread_count, last_message_time FROM chats WHERE session_id=? ORDER BY last_message_time DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []*ChatRow
	for rows.Next() {
		c := &ChatRow{}
		if err := rows.Scan(&c.SessionID, &c.JID, &c.Name, &c.UnreadCount, &c.LastMessageTime); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}

// MarkChatRead resets the unread count for a chat.
func MarkChatRead(db *sql.DB, sessionID, jid string) error {
	_, err := db.Exec(`UPDATE chats SET unread_count=0 WHERE session_id=? AND jid=?`, sessionID, jid)
	return err
}

// ChatMessageRow represents a single message in a chat history.
type ChatMessageRow struct {
	ID         int
	SessionID  string
	ChatJID    string
	SenderJID  string
	MessageID  string
	IsFromMe   bool
	Text       string
	MsgType    string
	Status     string
	Timestamp  time.Time
	RawMessage []byte
}

// InsertChatMessage stores an incoming or synced message.
func InsertChatMessage(db *sql.DB, sessionID, chatJID, senderJID, messageID string, isFromMe bool, text, msgType, status string, timestamp time.Time, rawMessage []byte) error {
	isFromMeInt := 0
	if isFromMe {
		isFromMeInt = 1
	}
	_, err := db.Exec(`
		INSERT INTO chat_messages (session_id, chat_jid, sender_jid, message_id, is_from_me, text, msg_type, status, timestamp, raw_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, chat_jid, message_id) DO NOTHING
	`, sessionID, chatJID, senderJID, messageID, isFromMeInt, text, msgType, status, timestamp, rawMessage)
	return err
}

// GetChatMessages retrieves the recent messages for a chat.
// GetChatMessages retrieves the recent messages for a chat.
func GetChatMessages(db *sql.DB, sessionID, chatJID string, limit int) ([]*ChatMessageRow, error) {
	rows, err := db.Query(`
		SELECT id, session_id, chat_jid, sender_jid, message_id, is_from_me, text, msg_type, status, timestamp, raw_message
		FROM chat_messages
		WHERE session_id=? AND chat_jid=?
		ORDER BY timestamp DESC
		LIMIT ?
	`, sessionID, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []*ChatMessageRow
	for rows.Next() {
		m := &ChatMessageRow{}
		var isFromMeInt int
		if err := rows.Scan(&m.ID, &m.SessionID, &m.ChatJID, &m.SenderJID, &m.MessageID, &isFromMeInt, &m.Text, &m.MsgType, &m.Status, &m.Timestamp, &m.RawMessage); err != nil {
			return nil, err
		}
		m.IsFromMe = isFromMeInt == 1
		msgs = append(msgs, m)
	}
	
	// Reverse to return in chronological order (oldest first, which UIs typically expect)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	
	return msgs, nil
}

// GetRawMessage retrieves the raw serialized protobuf bytes for a specific message.
func GetRawMessage(db *sql.DB, sessionID, msgID string) ([]byte, error) {
	var raw []byte
	err := db.QueryRow(`SELECT raw_message FROM chat_messages WHERE session_id=? AND message_id=?`, sessionID, msgID).Scan(&raw)
	return raw, err
}

// UpdateChatMessageStatus updates the delivery status of a message.
func UpdateChatMessageStatus(db *sql.DB, sessionID, msgID, status string) error {
	_, err := db.Exec(`UPDATE chat_messages SET status=? WHERE session_id=? AND message_id=?`, status, sessionID, msgID)
	return err
}

