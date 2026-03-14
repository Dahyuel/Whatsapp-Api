package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens the database connection and runs migrations.
func Open(driver, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driver, dsn+"?_journal=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return db, nil
}
