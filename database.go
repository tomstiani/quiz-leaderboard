package main

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func openDatabase(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "screenshots"), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	path, err := filepath.Abs(filepath.Join(dataDir, "leaderboard.db"))
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, entry := range entries {
		var applied bool
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = ?)", entry.Name()).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		script, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.Exec(string(script)); err == nil {
			_, err = tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", entry.Name())
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	for _, entry := range entries {
		script, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		digest := sha256.Sum256(script)
		checksum := hex.EncodeToString(digest[:])
		var stored sql.NullString
		if err := db.QueryRow("SELECT checksum FROM schema_migrations WHERE name = ?", entry.Name()).Scan(&stored); err != nil {
			return fmt.Errorf("read migration checksum %s: %w", entry.Name(), err)
		}
		if stored.Valid && stored.String != checksum {
			return fmt.Errorf("migration %s changed after it was applied", entry.Name())
		}
		if !stored.Valid {
			if _, err := db.Exec("UPDATE schema_migrations SET checksum = ? WHERE name = ?", checksum, entry.Name()); err != nil {
				return fmt.Errorf("record migration checksum %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}
