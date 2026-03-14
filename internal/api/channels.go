package api

import (
	"net/http"

	"whatsapp-api/internal/channels"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"
)

// RegisterChannelRoutes registers /channels endpoints.
func RegisterChannelRoutes(r gin.IRouter, mgr *session.Manager) {
	r.GET("/channels", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		list, err := channels.List(c.Request.Context(), sess.Client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"channels": list})
	})

	r.GET("/channels/:jid", func(c *gin.Context) {
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
		info, err := channels.GetInfo(c.Request.Context(), sess.Client, jid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, info)
	})

	r.POST("/channels/:jid/message", func(c *gin.Context) {
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
			Text string `json:"text" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		msgID, err := channels.SendText(c.Request.Context(), sess.Client, jid, body.Text)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"msg_id": msgID})
	})
}
