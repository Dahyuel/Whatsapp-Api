package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"whatsapp-api/internal/antiban"
	"whatsapp-api/internal/db"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// MessageJob is an item in the queue.
type MessageJob struct {
	TrackingID string
	SessionID  string
	JID        types.JID
	Message    *waProto.Message
	MediaType  string // "", "image", "video", etc.
	Text       string // original text for typing sim
	Retries    int
}

// Stats holds queue statistics for a session.
type Stats struct {
	SessionID string `json:"session_id"`
	Pending   int    `json:"pending"`
	Workers   int    `json:"workers"`
}

// Queue is a per-session FIFO message queue with anti-ban delays.
type Queue struct {
	mu          sync.Mutex
	sessionID   string
	jobs        chan *MessageJob
	quit        chan struct{}
	wg          sync.WaitGroup
	db          *sql.DB
	client      *whatsmeow.Client
	typeSim     *antiban.TypingSimulator
	adaptive    *antiban.AdaptiveTiming
	contextual  *ContextualQueue
	pending     int
	errorCount  int
}

// NewQueue creates a new per-session queue.
func NewQueue(
	sessionID string,
	database *sql.DB,
	typeSim *antiban.TypingSimulator,
	adaptive *antiban.AdaptiveTiming,
	contextual *ContextualQueue,
) *Queue {
	return &Queue{
		sessionID:  sessionID,
		jobs:       make(chan *MessageJob, 512),
		quit:       make(chan struct{}),
		db:         database,
		typeSim:    typeSim,
		adaptive:   adaptive,
		contextual: contextual,
	}
}

// SetClient sets/updates the WhatsMeow client (called after session connects).
func (q *Queue) SetClient(client *whatsmeow.Client) {
	q.mu.Lock()
	q.client = client
	q.mu.Unlock()
}

// Start launches the queue worker goroutine.
func (q *Queue) Start() {
	q.wg.Add(1)
	go q.worker()
}

// Stop gracefully stops the worker.
func (q *Queue) Stop() {
	close(q.quit)
	q.wg.Wait()
}

// Enqueue adds a job to the queue.
func (q *Queue) Enqueue(job *MessageJob) {
	q.mu.Lock()
	q.pending++
	q.mu.Unlock()
	// Persist to DB for durability
	payload, _ := json.Marshal(job)
	_ = db.UpsertMessageStatus(q.db, job.TrackingID, job.SessionID,
		job.JID.String(), "", "queued", string(payload), "")
	q.jobs <- job
}

// Stats returns current queue statistics.
func (q *Queue) Stats() Stats {
	q.mu.Lock()
	defer q.mu.Unlock()
	return Stats{
		SessionID: q.sessionID,
		Pending:   q.pending,
		Workers:   1,
	}
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for {
		select {
		case <-q.quit:
			return
		case job := <-q.jobs:
			q.processJob(job)
		}
	}
}

func (q *Queue) processJob(job *MessageJob) {
	defer func() {
		q.mu.Lock()
		q.pending--
		q.mu.Unlock()
	}()

	ctx := context.Background()

	// Contextual delay based on contact risk level
	extraDelay := q.contextual.ExtraDelay(ctx, job.SessionID, job.JID.String())
	if extraDelay > 0 {
		time.Sleep(extraDelay)
	}

	// Adaptive delay
	hour := time.Now().Hour()
	isNew := q.contextual.IsNewContact(ctx, job.SessionID, job.JID.String())
	delay := q.adaptive.ComputeDelay(len(job.Text), hour, !isNew, isNew)
	time.Sleep(delay)

	q.mu.Lock()
	client := q.client
	q.mu.Unlock()

	if client == nil {
		log.Warn().Str("session", job.SessionID).Msg("queue: client not ready, requeueing")
		time.Sleep(5 * time.Second)
		if job.Retries < 3 {
			job.Retries++
			q.Enqueue(job)
		} else {
			_ = db.UpdateMessageStatus(q.db, job.TrackingID, "failed", "client not ready")
		}
		return
	}

	// Typing simulation
	q.typeSim.Simulate(ctx, client, job.JID, job.Text)

	_ = db.UpdateMessageStatus(q.db, job.TrackingID, "sending", "")

	resp, err := client.SendMessage(ctx, job.JID, job.Message)
	if err != nil {
		log.Error().Err(err).Str("session", job.SessionID).Msg("queue: send failed")
		q.mu.Lock()
		q.errorCount++
		q.mu.Unlock()
		// Auto-slowdown: increase base delay on errors
		q.adaptive.BaseDelay += 1 * time.Second
		if job.Retries < 3 {
			job.Retries++
			backoff := time.Duration(job.Retries*job.Retries) * 5 * time.Second
			time.Sleep(backoff)
			q.Enqueue(job)
		} else {
			_ = db.UpdateMessageStatus(q.db, job.TrackingID, "failed", err.Error())
		}
		return
	}

	_ = db.UpdateMessageStatus(q.db, job.TrackingID, "sent", "")
	_ = q.contextual.RecordSent(ctx, job.SessionID, job.JID.String())
	log.Info().Str("session", job.SessionID).Str("msg_id", fmt.Sprintf("%v", resp)).Msg("queue: sent")
}
