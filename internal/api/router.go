package api

import (
	"database/sql"
	"time"

	"whatsapp-api/internal/auth"
	"whatsapp-api/internal/config"
	"whatsapp-api/internal/session"
	"whatsapp-api/internal/webhook"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// NewRouter constructs and returns the fully wired Gin router.
func NewRouter(cfg *config.Config, db *sql.DB, mgr *session.Manager, dispatcher *webhook.Dispatcher) *gin.Engine {
	if cfg.Port != "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS initialization
	r.Use(cors.New(cors.Config{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "X-API-Key", "Authorization", "X-WhatsApp-Signature", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiter: max 60 req/min per IP
	rate := limiter.Rate{Period: time.Minute, Limit: 60}
	store := memory.NewStore()
	lmt := limiter.New(store, rate)
	r.Use(mgin.NewMiddleware(lmt))

	// System routes (no auth)
	RegisterSystemRoutes(r)

	// Authenticated routes
	var authed gin.IRouter
	if cfg.JWTEnabled {
		authed = r.Group("/", auth.JWTMiddleware(cfg.JWTSecret), auth.APIKeyMiddleware(cfg.APIKey))
	} else {
		authed = r.Group("/", auth.APIKeyMiddleware(cfg.APIKey))
	}

	RegisterSessionRoutes(authed, mgr, dispatcher)
	RegisterMessageRoutes(authed, mgr)
	RegisterQueueRoutes(authed, mgr)
	RegisterChatRoutes(authed, mgr)
	RegisterGroupRoutes(authed, mgr)
	RegisterContactRoutes(authed, mgr)
	RegisterWebhookRoutes(authed, mgr, dispatcher)
	RegisterChannelRoutes(authed, mgr)

	return r
}
