package api

import (
	"net/http"

	"whatsapp-api/internal/groups"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type createGroupReq struct {
	Session      string   `json:"session" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Participants []string `json:"participants"`
}

type updateParticipantsReq struct {
	Session      string   `json:"session" binding:"required"`
	Participants []string `json:"participants" binding:"required"`
	Action       string   `json:"action" binding:"required"` // add|remove|promote|demote
}

// RegisterGroupRoutes registers /groups endpoints.
func RegisterGroupRoutes(r gin.IRouter, mgr *session.Manager) {
	r.POST("/groups", func(c *gin.Context) {
		var req createGroupReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := groups.Create(c.Request.Context(), sess.Client, req.Name, req.Participants)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"jid": jid})
	})

	r.GET("/groups", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		list, err := groups.ListJoined(c.Request.Context(), sess.Client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"groups": list})
	})

	r.GET("/groups/:jid", func(c *gin.Context) {
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
		info, err := groups.GetInfo(c.Request.Context(), sess.Client, jid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, info)
	})

	r.POST("/groups/:jid/participants", func(c *gin.Context) {
		var req updateParticipantsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := types.ParseJID(c.Param("jid"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		actionMap := map[string]whatsmeow.ParticipantChange{
			"add":     whatsmeow.ParticipantChangeAdd,
			"remove":  whatsmeow.ParticipantChangeRemove,
			"promote": whatsmeow.ParticipantChangePromote,
			"demote":  whatsmeow.ParticipantChangeDemote,
		}
		action, ok := actionMap[req.Action]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action: use add|remove|promote|demote"})
			return
		}
		if err := groups.UpdateParticipants(c.Request.Context(), sess.Client, jid, req.Participants, action); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.DELETE("/groups/:jid/leave", func(c *gin.Context) {
		sessID := c.Query("session")
		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := types.ParseJID(c.Param("jid"))
		if err := groups.Leave(c.Request.Context(), sess.Client, jid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "left"})
	})
}
