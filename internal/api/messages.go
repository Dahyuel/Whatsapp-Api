package api

import (
	"encoding/base64"
	"io"
	"net/http"

	"whatsapp-api/internal/messaging"
	"whatsapp-api/internal/queue"
	"whatsapp-api/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type sendTextReq struct {
	Session string `json:"session" binding:"required"`
	To      string `json:"to" binding:"required"`
	Text    string `json:"text" binding:"required"`
}

type sendMediaReq struct {
	Session   string `json:"session" binding:"required"`
	To        string `json:"to" binding:"required"`
	MediaType string `json:"media_type" binding:"required"` // image|video|audio|document|sticker
	URL       string `json:"url"`
	Base64    string `json:"base64"`
	MimeType  string `json:"mime_type"`
	Caption   string `json:"caption"`
	Filename  string `json:"filename"`
}

type sendLocationReq struct {
	Session   string  `json:"session" binding:"required"`
	To        string  `json:"to" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
}

type sendContactReq struct {
	Session     string `json:"session" binding:"required"`
	To          string `json:"to" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	VCard       string `json:"vcard" binding:"required"`
}

type sendReactionReq struct {
	Session      string `json:"session" binding:"required"`
	To           string `json:"to" binding:"required"`
	TargetMsgID  string `json:"target_msg_id" binding:"required"`
	TargetSender string `json:"target_sender" binding:"required"`
	Reaction     string `json:"reaction" binding:"required"`
}

type sendPollReq struct {
	Session         string   `json:"session" binding:"required"`
	To              string   `json:"to" binding:"required"`
	Question        string   `json:"question" binding:"required"`
	Options         []string `json:"options" binding:"required"`
	SelectableCount uint32   `json:"selectable_count"`
}

type deleteMessageReq struct {
	Session string `json:"session" binding:"required"`
	Chat    string `json:"chat" binding:"required"`
	MsgID   string `json:"msg_id" binding:"required"`
	FromMe  bool   `json:"from_me"`
}

type editMessageReq struct {
	Session string `json:"session" binding:"required"`
	Chat    string `json:"chat" binding:"required"`
	MsgID   string `json:"msg_id" binding:"required"`
	Text    string `json:"text" binding:"required"`
}

// RegisterMessageRoutes registers all /messages endpoints.
func RegisterMessageRoutes(r gin.IRouter, mgr *session.Manager) {
	r.POST("/messages/text", func(c *gin.Context) {
		var req sendTextReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := messaging.ParseJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID: " + err.Error()})
			return
		}
		trackID := uuid.New().String()
		msg := messaging.BuildTextMessage(req.Text)
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			Text:       req.Text,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.POST("/messages/media", func(c *gin.Context) {
		// Try multipart first, then JSON
		if c.ContentType() == "multipart/form-data" {
			handleMultipartMedia(c, mgr)
			return
		}
		var req sendMediaReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, err := messaging.ParseJID(req.To)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}
		var data []byte
		if req.URL != "" {
			data, err = messaging.FetchURL(req.URL)
		} else if req.Base64 != "" {
			data, err = base64.StdEncoding.DecodeString(req.Base64)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provide url or base64"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		mime := req.MimeType
		if mime == "" {
			mime = http.DetectContentType(data)
		}
		msg, err := messaging.UploadAndBuildMedia(c.Request.Context(), sess.Client,
			data, messaging.MediaType(req.MediaType), mime, req.Caption, req.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			MediaType:  req.MediaType,
			Text:       req.Caption,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.POST("/messages/location", func(c *gin.Context) {
		var req sendLocationReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.To)
		msg := messaging.BuildLocationMessage(req.Latitude, req.Longitude, req.Name, req.Address)
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			Text:       req.Name,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.POST("/messages/contact", func(c *gin.Context) {
		var req sendContactReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.To)
		msg := messaging.BuildContactMessage(req.DisplayName, req.VCard)
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			Text:       req.DisplayName,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.POST("/messages/reaction", func(c *gin.Context) {
		var req sendReactionReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.To)
		msg := messaging.BuildReactionMessage(req.TargetMsgID, req.TargetSender, req.Reaction)
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			Text:       req.Reaction,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.POST("/messages/poll", func(c *gin.Context) {
		var req sendPollReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.To)
		msg := messaging.BuildPollMessage(req.Question, req.Options, req.SelectableCount)
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  req.Session,
			JID:        jid,
			Message:    msg,
			Text:       req.Question,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	r.DELETE("/messages", func(c *gin.Context) {
		var req deleteMessageReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.Chat)
		if err := messaging.DeleteMessage(c.Request.Context(), sess.Client, jid, req.MsgID, req.FromMe); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})

	r.PATCH("/messages", func(c *gin.Context) {
		var req editMessageReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sess, err := mgr.Get(req.Session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		jid, _ := messaging.ParseJID(req.Chat)
		if err := messaging.EditMessage(c.Request.Context(), sess.Client, jid, req.MsgID, req.Text); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "edited"})
	})

	// Status check
	r.GET("/messages/:tracking_id/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tracking_id": c.Param("tracking_id"), "note": "check DB for status"})
	})
}

func handleMultipartMedia(c *gin.Context, mgr *session.Manager) {
	sessID := c.PostForm("session")
	to := c.PostForm("to")
	mediaType := c.PostForm("media_type")
	caption := c.PostForm("caption")
	filename := c.PostForm("filename")

	sess, err := mgr.Get(sessID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	jid, err := messaging.ParseJID(to)
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
	mime := http.DetectContentType(data)
	msg, err := messaging.UploadAndBuildMedia(c.Request.Context(), sess.Client,
		data, messaging.MediaType(mediaType), mime, caption, filename)
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
}
