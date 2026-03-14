package api

import (
	"net/http"

	"whatsapp-api/internal/contacts"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"
)

// RegisterContactRoutes registers /contacts endpoints.
func RegisterContactRoutes(r gin.IRouter, mgr *session.Manager) {
	r.GET("/contacts", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		result, err := contacts.GetContacts(c.Request.Context(), sess.Client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"contacts": result})
	})

	r.GET("/contacts/:jid", func(c *gin.Context) {
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
		info, err := contacts.GetContactInfo(c.Request.Context(), sess.Client, jid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, info)
	})

	r.POST("/contacts/exists", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		var body struct {
			Phones []string `json:"phones" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := contacts.CheckExists(c.Request.Context(), sess.Client, body.Phones)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	r.POST("/contacts/:jid/block", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		if err := contacts.BlockContact(c.Request.Context(), sess.Client, jid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "blocked"})
	})

	r.POST("/contacts/:jid/unblock", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		if err := contacts.UnblockContact(c.Request.Context(), sess.Client, jid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "unblocked"})
	})

	r.GET("/contacts/:jid/photo", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		url, err := contacts.GetProfilePhoto(c.Request.Context(), sess.Client, jid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
	})
}
