package api

import (
	"net/http"

	"whatsapp-api/internal/session"
	"whatsapp-api/internal/webhook"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type registerWebhookReq struct {
	SessionID string   `json:"session_id" binding:"required"`
	URL       string   `json:"url" binding:"required"`
	Secret    string   `json:"secret"`
	Events    []string `json:"events"` // e.g. ["message.received", "session.connected"]
}

// RegisterWebhookRoutes registers /webhooks endpoints.
func RegisterWebhookRoutes(r gin.IRouter, mgr *session.Manager, dispatcher *webhook.Dispatcher) {
	r.POST("/webhooks", func(c *gin.Context) {
		var req registerWebhookReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := mgr.Get(req.SessionID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		id := uuid.New().String()
		cfg := &webhook.Config{
			ID:        id,
			SessionID: req.SessionID,
			URL:       req.URL,
			Secret:    req.Secret,
			Events:    req.Events,
		}
		if err := dispatcher.Register(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "url": req.URL, "events": req.Events})
	})

	r.DELETE("/webhooks/:id", func(c *gin.Context) {
		if err := dispatcher.Unregister(c.Param("id")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})
}
