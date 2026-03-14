package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"whatsapp-api/internal/antiban"
	"whatsapp-api/internal/config"
	"whatsapp-api/internal/db"
	"whatsapp-api/internal/fingerprint"
	"whatsapp-api/internal/queue"
	"whatsapp-api/internal/webhook"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Manager manages all WhatsApp sessions.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	db         *sql.DB
	cfg        *config.Config
	dispatcher *webhook.Dispatcher
	dataDir    string
}

// NewManager creates a session manager and restores persisted sessions.
func NewManager(database *sql.DB, cfg *config.Config, dispatcher *webhook.Dispatcher) (*Manager, error) {
	m := &Manager{
		sessions:   make(map[string]*Session),
		db:         database,
		cfg:        cfg,
		dispatcher: dispatcher,
		dataDir:    filepath.Dir(cfg.DBDSN),
	}
	if err := m.restoreSessions(); err != nil {
		return nil, err
	}
	return m, nil
}

// Create creates a new session with an isolated WhatsMeow client.
func (m *Manager) Create(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %q already exists", id)
	}
	sess, err := m.buildSession(id)
	if err != nil {
		return nil, err
	}
	m.sessions[id] = sess
	_ = db.UpsertSession(m.db, id, string(StatusInitializing), "")
	return sess, nil
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return sess, nil
}

// List returns all sessions.
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

// Delete disconnects and removes a session.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	if sess.Client != nil {
		sess.Client.Disconnect()
	}
	if sess.Queue != nil {
		sess.Queue.Stop()
	}
	delete(m.sessions, id)
	return db.DeleteSession(m.db, id)
}

// StartLogin generates the QR code for a session.
func (m *Manager) StartLogin(id string) (<-chan string, error) {
	sess, err := m.Get(id)
	if err != nil {
		return nil, err
	}
	qrCh, err := sess.Client.GetQRChannel(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get qr channel: %w", err)
	}
	out := make(chan string, 2)
	go func() {
		defer close(out)
		for evt := range qrCh {
			if evt.Event == "code" {
				out <- evt.Code
			}
		}
	}()
	sess.Status = StatusQRPending
	_ = db.UpsertSession(m.db, id, string(StatusQRPending), "")
	if err := sess.Client.Connect(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return out, nil
}

// Logout logs out a session from WhatsApp.
func (m *Manager) Logout(id string) error {
	sess, err := m.Get(id)
	if err != nil {
		return err
	}
	if sess.Client == nil {
		return errors.New("client not initialized")
	}
	return sess.Client.Logout(context.Background())
}

// buildSession constructs a Session with isolated WhatsMeow client and queue.
func (m *Manager) buildSession(id string) (*Session, error) {
	// Each session gets its own SQLite store under /app/data/sessions/{id}.db
	sessionDBPath := filepath.Join(m.dataDir, "sessions", id+".db")
	if err := os.MkdirAll(filepath.Dir(sessionDBPath), 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", sessionDBPath+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("create sqlstore: %w", err)
	}

	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	devInfo := fingerprint.NewDeviceInfo()
	store.DeviceProps.PlatformType = nil // let whatsmeow pick
	_ = devInfo // device info stored on session for reference

	client := whatsmeow.NewClient(device, waLog.Noop)

	typSim := antiban.NewTypingSimulator(m.cfg.TypingEnabled, m.cfg.TypingCharsPerSecond)
	adaptive := antiban.NewAdaptiveTiming(m.cfg.QueueBaseDelay, m.cfg.QueueJitterMin, m.cfg.QueueJitterMax)
	contextual := queue.NewContextualQueue(m.db)
	q := queue.NewQueue(id, m.db, typSim, adaptive, contextual)
	q.SetClient(client)
	q.Start()

	presence := antiban.NewPresenceManager(client, m.cfg.PresenceEnabled)

	sess := &Session{
		ID:       id,
		Status:   StatusInitializing,
		Client:   client,
		Queue:    q,
		Device:   devInfo,
		Presence: presence,
	}

	// Register event handler
	client.AddEventHandler(m.makeEventHandler(sess))

	return sess, nil
}

// makeEventHandler returns a WhatsMeow event handler for a session.
func (m *Manager) makeEventHandler(sess *Session) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			sess.Status = StatusConnected
			sess.JID = sess.Client.Store.ID.String()
			_ = db.UpsertSession(m.db, sess.ID, string(StatusConnected), sess.JID)
			sess.Queue.SetClient(sess.Client)
			log.Info().Str("session", sess.ID).Str("jid", sess.JID).Msg("session connected")
			m.dispatcher.Dispatch(sess.ID, "session.connected", map[string]interface{}{
				"session": sess.ID,
				"jid":     sess.JID,
			})

		case *events.Disconnected:
			sess.Status = StatusDisconnected
			_ = db.UpsertSession(m.db, sess.ID, string(StatusDisconnected), sess.JID)
			log.Warn().Str("session", sess.ID).Msg("session disconnected")
			m.dispatcher.Dispatch(sess.ID, "session.disconnected", map[string]interface{}{
				"session": sess.ID,
			})

		case *events.LoggedOut:
			sess.Status = StatusLoggedOut
			_ = db.UpsertSession(m.db, sess.ID, string(StatusLoggedOut), "")
			log.Warn().Str("session", sess.ID).Msg("session logged out")

		case *events.Message:
			m.dispatcher.Dispatch(sess.ID, "message.received", buildMessagePayload(sess.ID, v))

		case *events.Receipt:
			handleReceipt(m.dispatcher, sess.ID, v)

		case *events.GroupInfo:
			m.dispatcher.Dispatch(sess.ID, "group.updated", map[string]interface{}{
				"session": sess.ID,
				"group":   v.JID.String(),
			})
		}
	}
}

// restoreSessions reloads all persisted sessions on startup.
func (m *Manager) restoreSessions() error {
	rows, err := db.ListSessions(m.db)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Status == string(StatusLoggedOut) {
			continue
		}
		sess, err := m.buildSession(row.ID)
		if err != nil {
			log.Error().Err(err).Str("session", row.ID).Msg("restore: failed to build session")
			continue
		}
		sess.JID = row.JID
		m.sessions[row.ID] = sess
		// Attempt auto-reconnect
		go func(s *Session) {
			if err := s.Client.Connect(); err != nil {
				log.Warn().Err(err).Str("session", s.ID).Msg("restore: auto-connect failed")
			}
		}(sess)
		log.Info().Str("session", row.ID).Msg("restore: session reconnecting")
	}
	return nil
}

func buildMessagePayload(sessionID string, v *events.Message) map[string]interface{} {
	from := ""
	if v.Info.Sender.IsEmpty() == false {
		from = v.Info.Sender.String()
	}
	return map[string]interface{}{
		"session":    sessionID,
		"id":         v.Info.ID,
		"from":       from,
		"chat":       v.Info.Chat.String(),
		"timestamp":  v.Info.Timestamp,
		"is_from_me": v.Info.IsFromMe,
		"type":       v.Info.Type,
	}
}

func handleReceipt(d *webhook.Dispatcher, sessionID string, v *events.Receipt) {
	event := ""
	switch v.Type {
	case types.ReceiptTypeDelivered:
		event = "message.delivered"
	case types.ReceiptTypeRead:
		event = "message.read"
	default:
		return
	}
	d.Dispatch(sessionID, event, map[string]interface{}{
		"session":  sessionID,
		"ids":      v.MessageIDs,
		"from":     v.MessageSource.Sender.String(),
		"chat":     v.MessageSource.Chat.String(),
		"time":     v.Timestamp,
	})
}
