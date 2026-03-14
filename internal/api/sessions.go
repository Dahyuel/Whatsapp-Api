package api

import (
	"encoding/base64"
	"net/http"
	"time"

	"whatsapp-api/internal/session"
	"whatsapp-api/internal/webhook"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

type createSessionReq struct {
	ID string `json:"id" binding:"required"`
}

// RegisterSessionRoutes registers all /sessions endpoints.
func RegisterSessionRoutes(r gin.IRouter, mgr *session.Manager, dispatcher *webhook.Dispatcher) {
	r.POST("/sessions", func(c *gin.Context) {
		var req createSessionReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Create(req.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, sessionToJSON(sess))
	})

	r.GET("/sessions", func(c *gin.Context) {
		sessions := mgr.List()
		out := make([]interface{}, 0, len(sessions))
		for _, s := range sessions {
			out = append(out, sessionToJSON(s))
		}
		c.JSON(http.StatusOK, gin.H{"sessions": out})
	})

	r.GET("/sessions/:id", func(c *gin.Context) {
		sess, err := mgr.Get(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sessionToJSON(sess))
	})

	r.POST("/sessions/:id/login", func(c *gin.Context) {
		id := c.Param("id")
		qrCh, err := mgr.StartLogin(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return the first QR code as a base64 PNG
		select {
		case code := <-qrCh:
			png, err := qrcode.Encode(code, qrcode.Medium, 256)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "qr encode: " + err.Error()})
				return
			}
			b64 := base64.StdEncoding.EncodeToString(png)
			c.JSON(http.StatusOK, gin.H{
				"qr_code":     b64,
				"qr_raw":      code,
				"expires_in":  60,
				"session":     id,
			})
		case <-time.After(30 * time.Second):
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "qr generation timed out"})
		}
	})

	r.POST("/sessions/:id/logout", func(c *gin.Context) {
		if err := mgr.Logout(c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
	})

	r.DELETE("/sessions/:id", func(c *gin.Context) {
		if err := mgr.Delete(c.Param("id")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})
}

func sessionToJSON(s *session.Session) gin.H {
	m := gin.H{
		"id":     s.ID,
		"status": string(s.Status),
		"jid":    s.JID,
	}
	if s.Device != nil {
		m["device"] = gin.H{
			"model":    s.Device.Model,
			"os":       s.Device.OSName,
			"version":  s.Device.OSVersion,
		}
	}
	return m
}
