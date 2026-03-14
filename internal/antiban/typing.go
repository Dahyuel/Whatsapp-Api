package antiban

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// TypingSimulator simulates human typing before sending a message.
type TypingSimulator struct {
	Enabled          bool
	CharsPerSecond   float64
}

// NewTypingSimulator creates a simulator with given config.
func NewTypingSimulator(enabled bool, charsPerSec float64) *TypingSimulator {
	if charsPerSec <= 0 {
		charsPerSec = 12.0
	}
	return &TypingSimulator{Enabled: enabled, CharsPerSecond: charsPerSec}
}

// Simulate sends composing presence, waits proportional to message length, then clears.
// The caller should send the message after this returns.
func (t *TypingSimulator) Simulate(ctx context.Context, client *whatsmeow.Client, jid types.JID, messageText string) {
	if !t.Enabled || client == nil {
		return
	}
	// Send composing presence
	_ = client.SendChatPresence(jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	// Calculate typing duration: chars / chars_per_second, capped at 8s
	chars := float64(len(messageText))
	seconds := chars / t.CharsPerSecond
	if seconds < 0.5 {
		seconds = 0.5
	}
	if seconds > 8.0 {
		seconds = 8.0
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(seconds * float64(time.Second))):
	}

	// Clear typing presence
	_ = client.SendChatPresence(jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
}

// SimulateVoice simulates recording presence for voice notes.
func (t *TypingSimulator) SimulateVoice(ctx context.Context, client *whatsmeow.Client, jid types.JID, durationSec float64) {
	if !t.Enabled || client == nil {
		return
	}
	_ = client.SendChatPresence(jid, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
	simulatedDuration := durationSec * 0.5 // simulate half the note duration
	if simulatedDuration > 6 {
		simulatedDuration = 6
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(simulatedDuration * float64(time.Second))):
	}
	_ = client.SendChatPresence(jid, types.ChatPresencePaused, types.ChatPresenceMediaAudio)
}
