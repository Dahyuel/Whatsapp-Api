package session

import (
	"whatsapp-api/internal/antiban"
	"whatsapp-api/internal/fingerprint"
	"whatsapp-api/internal/queue"

	"go.mau.fi/whatsmeow"
)

// Status represents the connection status of a session.
type Status string

const (
	StatusInitializing Status = "initializing"
	StatusQRPending    Status = "qr_pending"
	StatusConnected    Status = "connected"
	StatusDisconnected Status = "disconnected"
	StatusLoggedOut    Status = "logged_out"
)

// Session represents a single WhatsApp session.
type Session struct {
	ID          string
	Status      Status
	JID         string
	Client      *whatsmeow.Client
	Queue       *queue.Queue
	Device      *fingerprint.DeviceInfo
	Presence    *antiban.PresenceManager
	QRChannel   chan string
}
