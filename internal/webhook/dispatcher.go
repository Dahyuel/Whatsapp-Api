package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"whatsapp-api/internal/db"

	"github.com/rs/zerolog/log"
)

// Config holds webhook registration data.
type Config struct {
	ID        string   `json:"id"`
	SessionID string   `json:"session_id"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"`
	Events    []string `json:"events"`
}

// Dispatcher dispatches events to registered webhooks.
type Dispatcher struct {
	mu         sync.RWMutex
	hooks      map[string][]*Config // sessionID → []Config
	db         *sql.DB
	timeout    time.Duration
	maxRetries int
	httpClient *http.Client
}

// NewDispatcher creates a webhook dispatcher.
func NewDispatcher(database *sql.DB, timeout time.Duration, maxRetries int) *Dispatcher {
	d := &Dispatcher{
		hooks:      make(map[string][]*Config),
		db:         database,
		timeout:    timeout,
		maxRetries: maxRetries,
		httpClient: &http.Client{Timeout: timeout},
	}
	d.loadFromDB()
	return d
}

// Register adds a new webhook for a session.
func (d *Dispatcher) Register(cfg *Config) error {
	events, _ := json.Marshal(cfg.Events)
	if err := db.InsertWebhook(d.db, cfg.ID, cfg.SessionID, cfg.URL, cfg.Secret, string(events)); err != nil {
		return err
	}
	d.mu.Lock()
	d.hooks[cfg.SessionID] = append(d.hooks[cfg.SessionID], cfg)
	d.mu.Unlock()
	return nil
}

// Unregister removes a webhook by ID.
func (d *Dispatcher) Unregister(id string) error {
	if err := db.DeleteWebhook(d.db, id); err != nil {
		return err
	}
	d.mu.Lock()
	for sessID, hooks := range d.hooks {
		filtered := make([]*Config, 0, len(hooks))
		for _, h := range hooks {
			if h.ID != id {
				filtered = append(filtered, h)
			}
		}
		d.hooks[sessID] = filtered
	}
	d.mu.Unlock()
	return nil
}

// Dispatch fires an event to all matching webhooks for a session.
func (d *Dispatcher) Dispatch(sessionID, event string, payload interface{}) {
	d.mu.RLock()
	hooks := append([]*Config{}, d.hooks[sessionID]...)
	d.mu.RUnlock()

	body := map[string]interface{}{
		"event":   event,
		"session": sessionID,
		"data":    payload,
		"ts":      time.Now().Unix(),
	}
	data, _ := json.Marshal(body)

	for _, hook := range hooks {
		if !eventMatches(hook.Events, event) {
			continue
		}
		go d.deliver(hook, data)
	}
}

// deliver sends the webhook payload with retries.
func (d *Dispatcher) deliver(hook *Config, data []byte) {
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}
		if err := d.post(hook, data); err != nil {
			log.Warn().Err(err).Str("url", hook.URL).Int("attempt", attempt+1).Msg("webhook delivery failed")
			continue
		}
		return
	}
	log.Error().Str("url", hook.URL).Msg("webhook: all retries exhausted")
}

// post makes a single HTTP POST to the webhook URL.
func (d *Dispatcher) post(hook *Config, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsApp-API/1.0")

	// HMAC signature
	if hook.Secret != "" {
		mac := hmac.New(sha256.New, []byte(hook.Secret))
		mac.Write(data)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-WhatsApp-Signature", fmt.Sprintf("sha256=%s", sig))
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// loadFromDB restores persisted webhooks at startup.
func (d *Dispatcher) loadFromDB() {
	// Load all sessions' webhooks using a wildcard query approach
	// We'll iterate all sessions via the webhooks table directly
	rows, err := d.db.Query(`SELECT id, session_id, url, COALESCE(secret,''), events FROM webhooks`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, sessID, url, secret, eventsJSON string
		if err := rows.Scan(&id, &sessID, &url, &secret, &eventsJSON); err != nil {
			continue
		}
		var evts []string
		_ = json.Unmarshal([]byte(eventsJSON), &evts)
		cfg := &Config{ID: id, SessionID: sessID, URL: url, Secret: secret, Events: evts}
		d.hooks[sessID] = append(d.hooks[sessID], cfg)
	}
}

func eventMatches(events []string, event string) bool {
	for _, e := range events {
		if e == "*" || e == event || strings.HasPrefix(event, strings.TrimSuffix(e, "*")) {
			return true
		}
	}
	return len(events) == 0
}
