package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port   string
	APIKey string

	// JWT (existing API-level, optional)
	JWTEnabled bool
	JWTSecret  string

	// User-facing JWT (for admin/agent login UI)
	UserJWTSecret string

	// Admin seeding
	AdminUsername string
	AdminPassword string

	// Database
	DBDriver string // "sqlite" or "postgres"
	DBDSN    string

	// Queue / Anti-Ban
	QueueBaseDelay        time.Duration
	QueueJitterMin        time.Duration
	QueueJitterMax        time.Duration
	MaxMessagesPerMinute  int
	TypingEnabled         bool
	PresenceEnabled       bool
	TypingCharsPerSecond  float64

	// Webhook
	WebhookTimeout    time.Duration
	WebhookMaxRetries int

	// Media
	MediaStoragePath string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:                  getEnv("PORT", "3000"),
		APIKey:                getEnv("API_KEY", "changeme"),
		JWTEnabled:            getBool("JWT_ENABLED", false),
		JWTSecret:             getEnv("JWT_SECRET", "changeme-jwt-secret"),
		UserJWTSecret:         getEnv("USER_JWT_SECRET", "changeme-user-jwt-secret-32chars"),
		AdminUsername:         getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:         getEnv("ADMIN_PASSWORD", "admin123"),
		DBDriver:              getEnv("DB_DRIVER", "sqlite"),
		DBDSN:                 getEnv("DB_DSN", "/app/data/whatsapp.db"),
		QueueBaseDelay:        getDuration("QUEUE_BASE_DELAY", 4*time.Second),
		QueueJitterMin:        getDuration("QUEUE_JITTER_MIN", 1*time.Second),
		QueueJitterMax:        getDuration("QUEUE_JITTER_MAX", 3*time.Second),
		MaxMessagesPerMinute:  getInt("MAX_MESSAGES_PER_MINUTE", 20),
		TypingEnabled:         getBool("TYPING_ENABLED", true),
		PresenceEnabled:       getBool("PRESENCE_ENABLED", true),
		TypingCharsPerSecond:  getFloat("TYPING_CHARS_PER_SECOND", 12.0),
		WebhookTimeout:        getDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		WebhookMaxRetries:     getInt("WEBHOOK_MAX_RETRIES", 5),
		MediaStoragePath:      getEnv("MEDIA_STORAGE_PATH", "/app/data/media"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getFloat(key string, fallback float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
