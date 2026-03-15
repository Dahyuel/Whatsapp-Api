package chats

import (
	"database/sql"

	"whatsapp-api/internal/db"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// NewChatEvent is sent to the notify callback when a chat gets auto-assigned.
type NewChatEvent struct {
	AgentID       string
	SessionID     string
	JID           string
	AssignmentID  string
}

// NotifyFunc is called after a successful auto-assignment.
type NotifyFunc func(ev NewChatEvent)

// AutoAssignNewChat picks a random agent and assigns the given session+jid to them.
// It is a no-op if the chat is already assigned or if there are no agents.
// The optional notify callback (may be nil) is called with the assignment details.
// Returns the agentID that was assigned (empty string if no assignment happened).
func AutoAssignNewChat(database *sql.DB, sessionID, jid string, notify NotifyFunc) string {
	// Already assigned?
	existing, err := db.GetAgentForChat(database, sessionID, jid)
	if err == nil && existing != nil && existing.AgentID != "" {
		return existing.AgentID
	}

	agent, err := db.GetRandomAgent(database)
	if err != nil {
		// No agents yet — silently skip
		return ""
	}

	id := uuid.New().String()
	if err := db.AssignChatToAgent(database, id, agent.ID, sessionID, jid); err != nil {
		log.Error().Err(err).Msg("auto-assign chat")
		return ""
	}

	log.Info().Str("agent", agent.Username).Str("jid", jid).Msg("auto-assigned new chat")

	if notify != nil {
		notify(NewChatEvent{
			AgentID:      agent.ID,
			SessionID:    sessionID,
			JID:          jid,
			AssignmentID: id,
		})
	}

	return agent.ID
}
