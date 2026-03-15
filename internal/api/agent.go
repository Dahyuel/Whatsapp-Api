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
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
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
			Name         string `json:"name"`
		}
		out := make([]chatOut, 0, len(assigned))
		for _, ac := range assigned {
			out = append(out, chatOut{
				AssignmentID: ac.ID,
				SessionID:    ac.SessionID,
				JID:          ac.JID,
				Name:         ac.Name,
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
		jid, err := types.ParseJID(jidParam)
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

	// GET /agent/chats/:jid/avatar
	r.GET("/agent/chats/:jid/avatar", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		sessID := c.Query("session")
		jidParam := c.Param("jid")

		// Verify assignment
		ac, err := db.GetAgentForChat(database, sessID, jidParam)
		if err != nil || ac.AgentID != aid {
			c.JSON(http.StatusForbidden, gin.H{"error": "chat not assigned to you"})
			return
		}

		sess, err := mgr.Get(sessID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		jid, err := messaging.ParseJID(jidParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JID"})
			return
		}

		info, err := sess.Client.GetProfilePictureInfo(c.Request.Context(), jid, nil)
		if err != nil {
			// If none, return a placeholder or 404
			c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": info.URL})
	})

	// GET /agent/chats/:jid/messages/:msgid/media
	r.GET("/agent/chats/:jid/messages/:msgid/media", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		sessID := c.Query("session")
		jidParam := c.Param("jid")
		msgID := c.Param("msgid")

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

		// Look up the message in our local DB to see what type of media it is
		// and importantly, grab its raw payload.
		rawBytes, err := db.GetRawMessage(database, sessID, msgID)
		if err != nil || len(rawBytes) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "media payload not found locally"})
			return
		}

		// Unmarshal the raw bytes back into a waE2E.Message
		parsedMsg := &waE2E.Message{}
		if err := proto.Unmarshal(rawBytes, parsedMsg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse original message"})
			return
		}

		// Extract the specific DownloadableMessage interface
		var downloadable whatsmeow.DownloadableMessage
		var mimeType string

		if parsedMsg.GetImageMessage() != nil {
			downloadable = parsedMsg.GetImageMessage()
			mimeType = parsedMsg.GetImageMessage().GetMimetype()
		} else if parsedMsg.GetVideoMessage() != nil {
			downloadable = parsedMsg.GetVideoMessage()
			mimeType = parsedMsg.GetVideoMessage().GetMimetype()
		} else if parsedMsg.GetAudioMessage() != nil {
			downloadable = parsedMsg.GetAudioMessage()
			mimeType = parsedMsg.GetAudioMessage().GetMimetype()
		} else if parsedMsg.GetDocumentMessage() != nil {
			downloadable = parsedMsg.GetDocumentMessage()
			mimeType = parsedMsg.GetDocumentMessage().GetMimetype()
			fileName := parsedMsg.GetDocumentMessage().GetFileName()
			if fileName == "" {
				fileName = "document.bin"
			}
			c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message does not contain supported media"})
			return
		}

		// Use WhatsMeow's Download method which internally handles the decryption keys
		data, err := sess.Client.Download(c.Request.Context(), downloadable)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download/decrypt media: " + err.Error()})
			return
		}

		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}

		c.Data(http.StatusOK, mimeType, data)
	})

	// DELETE /agent/chats/:jid/messages/:msgid — Revoke a message
	r.DELETE("/agent/chats/:jid/messages/:msgid", func(c *gin.Context) {
		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		sessID := c.Query("session")
		jidParam := c.Param("jid")
		msgID := c.Param("msgid")

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

		// Look up the message locally to get the sender
		msgs, err := db.GetChatMessages(database, sessID, jidParam, 50)
		var targetMsg *db.ChatMessageRow
		if err == nil {
			for _, m := range msgs {
				if m.MessageID == msgID {
					targetMsg = m
					break
				}
			}
		}

		if targetMsg == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}

		senderJid, _ := messaging.ParseJID(targetMsg.SenderJID)
		
		msg := sess.Client.BuildRevoke(jid, senderJid, msgID)
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  sessID,
			JID:        jid,
			Message:    msg,
			Text:       "Revoke Message " + msgID,
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})

	// POST /agent/chats/:jid/messages/:msgid/forward — Forward a message
	type forwardReq struct {
		ToJID string `json:"to_jid" binding:"required"`
	}
	r.POST("/agent/chats/:jid/messages/:msgid/forward", func(c *gin.Context) {
		var req forwardReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		agentID, _ := c.Get("user_id")
		aid := agentID.(string)
		sessID := c.Query("session")
		jidParam := c.Param("jid")
		msgID := c.Param("msgid")

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

		targetJid, err := messaging.ParseJID(req.ToJID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target JID"})
			return
		}

		// Grab raw bytes
		rawBytes, err := db.GetRawMessage(database, sessID, msgID)
		if err != nil || len(rawBytes) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "historical message not found locally"})
			return
		}

		parsedMsg := &waE2E.Message{}
		if err := proto.Unmarshal(rawBytes, parsedMsg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse message"})
			return
		}

		// What meow needs is just sending the parsed message to a different JID
		// However, typical forwarding wraps it in an ExtendedTextMessage ContextInfo etc.
		// For simplicity, we just submit the unmarshalled message as is.
		trackID := uuid.New().String()
		sess.Queue.Enqueue(&queue.MessageJob{
			TrackingID: trackID,
			SessionID:  sessID,
			JID:        targetJid,
			Message:    parsedMsg,
			Text:       "Forwarded Message",
		})
		c.JSON(http.StatusAccepted, gin.H{"tracking_id": trackID, "status": "queued"})
	})
}
