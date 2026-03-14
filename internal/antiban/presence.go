package antiban

import (
	"context"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// PresenceManager tracks and throttles presence state updates per session.
// It prevents spamming WhatsApp servers with rapid presence changes.
type PresenceManager struct {
	mu            sync.Mutex
	client        *whatsmeow.Client
	lastState     types.ChatPresence
	lastUpdate    time.Time
	minInterval   time.Duration
	enabled       bool
}

// NewPresenceManager creates a PresenceManager.
func NewPresenceManager(client *whatsmeow.Client, enabled bool) *PresenceManager {
	return &PresenceManager{
		client:      client,
		minInterval: 2 * time.Second,
		enabled:     enabled,
	}
}

// SetAvailable marks the session as available.
func (p *PresenceManager) SetAvailable() {
	p.send(types.ChatPresencePaused, types.ChatPresenceMediaText)
}

// SetComposing marks the session as typing.
func (p *PresenceManager) SetComposing(jid types.JID) {
	if !p.enabled {
		return
	}
	_ = p.client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

// SetRecording marks the session as recording audio.
func (p *PresenceManager) SetRecording(jid types.JID) {
	if !p.enabled {
		return
	}
	_ = p.client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
}

// SetPaused marks the session as paused.
func (p *PresenceManager) SetPaused(jid types.JID) {
	if !p.enabled {
		return
	}
	_ = p.client.SendChatPresence(context.Background(), jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
}

// send is an internal throttled presence sender.
func (p *PresenceManager) send(state types.ChatPresence, media types.ChatPresenceMedia) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if state == p.lastState && time.Since(p.lastUpdate) < p.minInterval {
		return // throttle duplicate state
	}
	p.lastState = state
	p.lastUpdate = time.Now()
}
