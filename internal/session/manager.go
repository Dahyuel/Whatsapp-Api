package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"whatsapp-api/internal/antiban"
	"whatsapp-api/internal/chats"
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
	"google.golang.org/protobuf/proto"
)

// SSEPublisher is a minimal interface so the session package does not
// import the api package (avoiding a circular dependency).
type SSEPublisher interface {
	Publish(agentID string, eventType string, data interface{})
	PublishAll(eventType string, data interface{})
}

// Manager manages all WhatsApp sessions.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	db         *sql.DB
	cfg        *config.Config
	dispatcher *webhook.Dispatcher
	hub        SSEPublisher
	dataDir    string
}

// NewManager creates a session manager and restores persisted sessions.
func NewManager(database *sql.DB, cfg *config.Config, dispatcher *webhook.Dispatcher, hub SSEPublisher) (*Manager, error) {
	m := &Manager{
		sessions:   make(map[string]*Session),
		db:         database,
		cfg:        cfg,
		dispatcher: dispatcher,
		hub:        hub,
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

		case *events.HistorySync:
			log.Info().Str("session", sess.ID).Str("type", string(v.Data.GetSyncType())).Msg("history sync received")
			for _, conv := range v.Data.GetConversations() {
				chatJID := conv.GetID()
				name := conv.GetName()
				if name == "" {
					name = chatJID
				}
				unreadCount := int(conv.GetUnreadCount())
				// We don't have the last message time directly here, but we will sort it out below
				lastMsgTime := time.Unix(0, 0)
				
				for _, msgInfo := range conv.GetMessages() {
					msgWrapper := msgInfo.GetMessage()
					if msgWrapper == nil {
						continue
					}
					
					// msgWrapper is a *waWeb.WebMessageInfo
					key := msgWrapper.GetKey()
					msgID := key.GetID()
					isFromMe := key.GetFromMe()
					senderJID := chatJID
					if key.GetParticipant() != "" {
						senderJID = key.GetParticipant()
					} else if isFromMe {
						senderJID = sess.JID // me
					}

					realMsg := msgWrapper.GetMessage()
					text := realMsg.GetConversation()
					if text == "" && realMsg.GetExtendedTextMessage() != nil {
						text = realMsg.GetExtendedTextMessage().GetText()
					}
					msgType := "text"
					if realMsg.GetImageMessage() != nil {
						msgType = "image"
					} else if realMsg.GetVideoMessage() != nil {
						msgType = "video"
					} else if realMsg.GetDocumentMessage() != nil {
						msgType = "document"
					} else if realMsg.GetAudioMessage() != nil {
						msgType = "audio"
					}

					timestamp := time.Unix(int64(msgWrapper.GetMessageTimestamp()), 0)
					if timestamp.After(lastMsgTime) {
						lastMsgTime = timestamp
					}

					status := ""
					if isFromMe {
						status = "SERVER_ACK"
					}

					rawBytes, _ := proto.Marshal(msgWrapper)
					_ = db.InsertChatMessage(m.db, sess.ID, chatJID, senderJID, msgID, isFromMe, text, msgType, status, timestamp, rawBytes)
				}
				_ = db.UpsertChat(m.db, sess.ID, chatJID, name, unreadCount, lastMsgTime)
				
				// Auto-assign the chat to an agent if not yet assigned.
				chats.AutoAssignNewChat(m.db, sess.ID, chatJID, func(ev chats.NewChatEvent) {
					if m.hub != nil {
						m.hub.Publish(ev.AgentID, "new_chat", map[string]interface{}{
							"assignment_id": ev.AssignmentID,
							"session_id":    ev.SessionID,
							"jid":           ev.JID,
						})
					}
				})
			}
			log.Info().Str("session", sess.ID).Msg("history sync processed")

		case *events.Message:
			payload := buildMessagePayload(sess.ID, v)
			m.dispatcher.Dispatch(sess.ID, "message.received", payload)

			chatJID := v.Info.Chat.String()
			msgID := v.Info.ID
			senderJID := ""
			if v.Info.Sender.IsEmpty() == false {
				senderJID = v.Info.Sender.String()
			}
			
			// Save in local chat_messages storage
			text := v.Message.GetConversation()
			if text == "" && v.Message.GetExtendedTextMessage() != nil {
				text = v.Message.GetExtendedTextMessage().GetText()
			}
			msgType := "text"
			if v.Message.GetImageMessage() != nil {
				msgType = "image"
			} else if v.Message.GetVideoMessage() != nil {
				msgType = "video"
			} else if v.Message.GetDocumentMessage() != nil {
				msgType = "document"
			} else if v.Message.GetAudioMessage() != nil {
				msgType = "audio"
			} else if v.Message.GetStickerMessage() != nil {
				msgType = "sticker"
			}
			
			status := ""
			if v.Info.IsFromMe {
				status = "SERVER_ACK"
			}

			rawBytes, _ := proto.Marshal(v.Message)
			_ = db.InsertChatMessage(m.db, sess.ID, chatJID, senderJID, msgID, v.Info.IsFromMe, text, msgType, status, v.Info.Timestamp, rawBytes)
			
			// Also upsert chat
			unreadInc := 1
			if v.Info.IsFromMe {
				unreadInc = 0
			}
			_ = db.UpsertChat(m.db, sess.ID, chatJID, v.Info.PushName, unreadInc, v.Info.Timestamp)

			// Skip messages sent by us for auto assignment
			if v.Info.IsFromMe {
				break
			}


			// Auto-assign the chat to an agent if not yet assigned.
			// The notify callback pushes a "new_chat" SSE event to the assigned agent.
			assignedAgentID := chats.AutoAssignNewChat(m.db, sess.ID, chatJID, func(ev chats.NewChatEvent) {
				if m.hub != nil {
					m.hub.Publish(ev.AgentID, "new_chat", map[string]interface{}{
						"assignment_id": ev.AssignmentID,
						"session_id":    ev.SessionID,
						"jid":           ev.JID,
					})
				}
			})

			// If no agent was assigned at all, skip SSE
			if assignedAgentID == "" {
				break
			}

			// Push a "new_message" event so the assigned agent's chat updates in real time.
			if m.hub != nil {
				m.hub.Publish(assignedAgentID, "new_message", payload)
			}

		case *events.Receipt:
			m.handleReceipt(sess.ID, v)

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

func (m *Manager) handleReceipt(sessionID string, v *events.Receipt) {
	event := ""
	status := ""
	switch v.Type {
	case types.ReceiptTypeDelivered:
		event = "message.delivered"
		status = "DELIVERY_ACK"
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		event = "message.read"
		status = "READ"
	case types.ReceiptTypePlayed:
		event = "message.played"
		status = "PLAYED"
	default:
		return
	}

	// Update DB for each message ID
	for _, msgID := range v.MessageIDs {
		_ = db.UpdateChatMessageStatus(m.db, sessionID, msgID, status)
	}

	// Broadcast via SSE and Webhooks
	m.dispatcher.Dispatch(sessionID, event, map[string]interface{}{
		"session":  sessionID,
		"ids":      v.MessageIDs,
		"from":     v.MessageSource.Sender.String(),
		"chat":     v.MessageSource.Chat.String(),
		"time":     v.Timestamp,
	})

	m.hub.PublishAll("message_status", map[string]interface{}{
		"session": sessionID,
		"chat":   v.MessageSource.Chat.String(),
		"ids":    v.MessageIDs,
		"status": status,
	})
}
