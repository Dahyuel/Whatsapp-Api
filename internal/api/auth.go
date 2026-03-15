package api

import (
	"database/sql"
	"net/http"

	"whatsapp-api/internal/auth"
	"whatsapp-api/internal/config"
	"whatsapp-api/internal/db"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers the /auth endpoints (no auth required).
func RegisterAuthRoutes(r gin.IRouter, database *sql.DB, cfg *config.Config) {
	r.POST("/auth/login", func(c *gin.Context) {
		var body struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
			return
		}

		user, err := db.GetUserByUsername(database, body.Username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if !auth.CheckPassword(user.PasswordHash, body.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := auth.GenerateToken(user.ID, user.Username, user.Role, cfg.UserJWTSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":    token,
			"role":     user.Role,
			"user_id":  user.ID,
			"username": user.Username,
		})
	})
}
