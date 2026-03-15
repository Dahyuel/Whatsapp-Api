package api

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"

	"whatsapp-api/internal/chats"
	"whatsapp-api/internal/db"
	"whatsapp-api/internal/messaging"
	"whatsapp-api/internal/queue"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
)

// RegisterAgentRoutes registers agent-facing endpoints.
// These are protected by UserJWTMiddleware; the agent may only see their own chats.
func RegisterAgentRoutes(r gin.IRouter, mgr *session.Manager, database *sql.DB) {
	// GET /agent/chats — list chats assigned to this agent
	r.GET("/agent/chats", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)

		assigned, err := db.GetChatsForAgent(database, aid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type chatOut struct {
			AssignmentID string `json:"assignment_id"`
			SessionID    string `json:"session_id"`
			JID          string `json:"jid"`
		}
		out := make([]chatOut, 0, len(assigned))
		for _, ac := range assigned {
			out = append(out, chatOut{
				AssignmentID: ac.ID,
				SessionID:    ac.SessionID,
				JID:          ac.JID,
			})
		}
		c.JSON(http.StatusOK, gin.H{"chats": out})
	})

	// GET /agent/chats/:jid/messages — get messages for an assigned chat
	r.GET("/agent/chats/:jid/messages", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		sessID := c.Query("session")
		jidParam := c.Param("jid")

		// Verify the agent is actually assigned to this chat
		ac, err := db.GetAgentForChat(database, sessID, jidParam)
		if err != nil || ac.AgentID != aid {
			c.JSON(http.StatusForbidden, gin.H{"error": "chat not assigned to you"})
			return
		}

		countStr := c.DefaultQuery("count", "50")
		count, _ := strconv.Atoi(countStr)

		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := types.ParseJID(jidParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		msgs, err := chats.GetMessages(c.Request.Context(), sess.Client, jid, count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"messages": msgs})
	})

	// POST /agent/chats/:jid/send/text
	r.POST("/agent/chats/:jid/send/text", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		jidParam := c.Param("jid")

		var body struct {
			SessionID string `json:"session_id" binding:"required"`
			Text      string `json:"text" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify assignment
		ac, err := db.GetAgentForChat(database, body.SessionID, jidParam)
		if err != nil || ac.AgentID != aid {
			c.JSON(http.StatusForbidden, gin.H{"error": "chat not assigned to you"})
			return
		}

		sess, err := mgr.Get(body.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := messaging.ParseJID(jidParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		trackID := uuid.New().String()
		msg := messaging.BuildTextMessage(body.Text)
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  body.SessionID,
			JID:        jid,
			Message:    msg,
			Text:       body.Text,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	// POST /agent/chats/:jid/send/media — multipart (file upload)
	r.POST("/agent/chats/:jid/send/media", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		jidParam := c.Param("jid")
		sessID := c.PostForm("session_id")
		mediaType := c.PostForm("media_type")
		caption := c.PostForm("caption")
		filename := c.PostForm("filename")

		// Verify assignment
		ac, err := db.GetAgentForChat(database, sessID, jidParam)
		if err != nil || ac.AgentID != aid {
			c.JSON(http.StatusForbidden, gin.H{"error": "chat not assigned to you"})
			return
		}

		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := messaging.ParseJID(jidParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing file: " + err.Error()})
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		mimeType := http.DetectContentType(data)
		msg, err := messaging.UploadAndBuildMedia(c.Request.Context(), sess.Client,
			data, messaging.MediaType(mediaType), mimeType, caption, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  sessID,
			JID:        jid,
			Message:    msg,
			MediaType:  mediaType,
			Text:       caption,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})
}
