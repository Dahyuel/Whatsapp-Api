package api

import (
	"net/http"

	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
)

// RegisterQueueRoutes registers /queue endpoints.
func RegisterQueueRoutes(r gin.IRouter, mgr *session.Manager) {
	r.GET("/queue", func(c *gin.Context) {
		sessions := mgr.List()
		stats := make([]interface{}, 0, len(sessions))
		for _, s := range sessions {
			if s.Queue != nil {
				stats = append(stats, s.Queue.Stats())
			}
		}
		c.JSON(http.StatusOK, gin.H{"queues": stats})
	})

	r.GET("/queue/:session", func(c *gin.Context) {
		sess, err := mgr.Get(c.Param("session"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if sess.Queue == nil {
			c.JSON(http.StatusOK, gin.H{"pending": 0})
			return
		}
		c.JSON(http.StatusOK, sess.Queue.Stats())
	})
}
