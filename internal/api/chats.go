package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"whatsapp-api/internal/chats"
	"whatsapp-api/internal/db"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"
)

// RegisterChatRoutes registers /chats endpoints.
func RegisterChatRoutes(r gin.IRouter, mgr *session.Manager, database *sql.DB) {
	r.GET("/chats", func(c *gin.Context) {
		sessID := c.Query("session")
		result, err := chats.ListChats(c.Request.Context(), database, sessID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"chats": result})
	})

	r.GET("/chats/:jid/messages", func(c *gin.Context) {
		sessID := c.Query("session")
		countStr := c.DefaultQuery("count", "50")
		count, _ := strconv.Atoi(countStr)
		jid, err := types.ParseJID(c.Param("jid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		msgs, err := chats.GetMessages(c.Request.Context(), database, sessID, jid, count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"messages": msgs})
	})

	r.POST("/chats/:jid/read", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := types.ParseJID(c.Param("jid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		var body struct {
			MsgIDs []string `json:"msg_ids"`
		}
		_ = c.ShouldBindJSON(&body)
		msgIDs := make([]types.MessageID, len(body.MsgIDs))
		for i, id := range body.MsgIDs {
			msgIDs[i] = types.MessageID(id)
		}
		if err := chats.MarkRead(c.Request.Context(), sess.Client, jid, msgIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = db.MarkChatRead(database, sessID, jid.String())
		c.JSON(http.StatusOK, gin.H{"status": "marked_read"})
	})

	r.POST("/chats/:jid/mute", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		var body struct{ Muted bool `json:"muted"` }
		_ = c.ShouldBindJSON(&body)
		_ = chats.SetMuted(c.Request.Context(), sess.Client, jid, body.Muted)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/chats/:jid/pin", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		var body struct{ Pinned bool `json:"pinned"` }
		_ = c.ShouldBindJSON(&body)
		_ = chats.SetPinned(c.Request.Context(), sess.Client, jid, body.Pinned)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/chats/:jid/archive", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		var body struct{ Archived bool `json:"archived"` }
		_ = c.ShouldBindJSON(&body)
		_ = chats.SetArchived(c.Request.Context(), sess.Client, jid, body.Archived)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
