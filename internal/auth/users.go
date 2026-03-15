package auth

import (
	"database/sql"

	"whatsapp-api/internal/db"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword bcrypt-hashes a plain-text password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword returns true if plain matches the bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// SeedAdminUser ensures at least one admin account exists. It uses the
// provided username/password only if no admin exists yet.
func SeedAdminUser(database *sql.DB, username, password string) error {
	// Check if admin already exists
	users, err := db.ListUsers(database)
	if err != nil {
		return err
	}
	for _, u := range users {
		if u.Role == "admin" {
			return nil // already seeded
		}
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return db.CreateUser(database, uuid.New().String(), username, hash, "admin")
}
