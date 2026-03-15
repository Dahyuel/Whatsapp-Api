package api

import (
	"database/sql"
	"net/http"

	"whatsapp-api/internal/auth"
	"whatsapp-api/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterUserRoutes registers admin-only /admin/users endpoints.
func RegisterUserRoutes(r gin.IRouter, database *sql.DB) {
	// List all users
	r.GET("/admin/users", func(c *gin.Context) {
		users, err := db.ListUsers(database)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		type userOut struct {
			ID        string `json:"id"`
			Username  string `json:"username"`
			Role      string `json:"role"`
			CreatedAt string `json:"created_at"`
		}
		out := make([]userOut, 0, len(users))
		for _, u := range users {
			out = append(out, userOut{ID: u.ID, Username: u.Username, Role: u.Role, CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		c.JSON(http.StatusOK, gin.H{"users": out})
	})

	// Create a user (admin or agent)
	r.POST("/admin/users", func(c *gin.Context) {
		var body struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Role     string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Role != "admin" && body.Role != "agent" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or agent"})
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
			return
		}
		id := uuid.New().String()
		if err := db.CreateUser(database, id, body.Username, hash, body.Role); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "username": body.Username, "role": body.Role})
	})

	// Delete a user
	r.DELETE("/admin/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DeleteUser(database, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	})

	// Assign chats to an agent
	r.POST("/admin/users/:id/chats", func(c *gin.Context) {
		agentID := c.Param("id")
		var body struct {
			SessionID string   `json:"session_id" binding:"required"`
			JIDs      []string `json:"jids" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, jid := range body.JIDs {
			id := uuid.New().String()
			if err := db.AssignChatToAgent(database, id, agentID, body.SessionID, jid); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "assigned", "count": len(body.JIDs)})
	})

	// Get chats assigned to an agent
	r.GET("/admin/users/:id/chats", func(c *gin.Context) {
		agentID := c.Param("id")
		chats, err := db.GetChatsForAgent(database, agentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		type chatOut struct {
			ID        string `json:"id"`
			SessionID string `json:"session_id"`
			JID       string `json:"jid"`
		}
		out := make([]chatOut, 0, len(chats))
		for _, ch := range chats {
			out = append(out, chatOut{ID: ch.ID, SessionID: ch.SessionID, JID: ch.JID})
		}
		c.JSON(http.StatusOK, gin.H{"chats": out})
	})

	// Remove a specific chat assignment by its assignment ID
	r.DELETE("/admin/users/:id/chats/:chatid", func(c *gin.Context) {
		if err := db.UnassignChat(database, c.Param("chatid")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "unassigned"})
	})
}
