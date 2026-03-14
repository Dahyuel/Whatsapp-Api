package queue

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"whatsapp-api/internal/db"
)

// RiskLevel represents the contact risk classification.
type RiskLevel string

const (
	RiskNew      RiskLevel = "new"
	RiskUnknown  RiskLevel = "unknown"
	RiskNormal   RiskLevel = "normal"
	RiskFrequent RiskLevel = "frequent"
)

// ContextualQueue classifies contacts and computes per-contact extra delays.
type ContextualQueue struct {
	database *sql.DB
}

// NewContextualQueue creates a ContextualQueue.
func NewContextualQueue(database *sql.DB) *ContextualQueue {
	return &ContextualQueue{database: database}
}

// IsNewContact returns true if this contact has never been messaged before.
func (c *ContextualQueue) IsNewContact(ctx context.Context, sessionID, jid string) bool {
	count, _, _, err := db.GetCooldown(ctx, c.database, sessionID, jid)
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return count == 0
}

// ExtraDelay returns additional delay for high-risk contacts.
func (c *ContextualQueue) ExtraDelay(ctx context.Context, sessionID, jid string) time.Duration {
	count, risk, lastSent, err := db.GetCooldown(ctx, c.database, sessionID, jid)
	if errors.Is(err, sql.ErrNoRows) {
		// Brand new contact — maximum caution
		return 6 * time.Second
	}
	if err != nil {
		return 0
	}

	// Duplicate message spam prevention: if sent within last 5s return 10s penalty
	if time.Since(lastSent) < 5*time.Second && count > 0 {
		return 10 * time.Second
	}

	switch RiskLevel(risk) {
	case RiskNew:
		return 5 * time.Second
	case RiskUnknown:
		return 3 * time.Second
	case RiskFrequent:
		return 0
	default:
		return 1 * time.Second
	}
}

// RecordSent updates the contact cooldown after a successful send.
func (c *ContextualQueue) RecordSent(ctx context.Context, sessionID, jid string) error {
	count, _, _, err := db.GetCooldown(ctx, c.database, sessionID, jid)
	if errors.Is(err, sql.ErrNoRows) {
		return db.UpsertCooldown(ctx, c.database, sessionID, jid, 1, string(RiskNew))
	}
	if err != nil {
		return err
	}
	count++
	newRisk := classifyRisk(count)
	return db.UpsertCooldown(ctx, c.database, sessionID, jid, count, string(newRisk))
}

// classifyRisk assigns a risk level based on total message count.
func classifyRisk(count int) RiskLevel {
	switch {
	case count == 0:
		return RiskNew
	case count < 3:
		return RiskUnknown
	case count < 15:
		return RiskNormal
	default:
		return RiskFrequent
	}
}
