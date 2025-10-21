package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Driver string

const (
	SQLite     Driver = "sqlite"
	PostgreSQL Driver = "postgres"
	MariaDB    Driver = "mariadb"
	MySQL      Driver = "mysql"
)

type Engine struct {
	driver Driver
	db     *sql.DB
	path   string
}

func New(driver Driver, connectionString string) (*Engine, error) {
	var db *sql.DB
	var err error

	switch driver {
	case SQLite:
		// SQLite with optimized settings
		db, err = sql.Open("sqlite3", connectionString+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=10000")
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}

		// Set connection pool settings for SQLite
		db.SetMaxOpenConns(1) // SQLite doesn't benefit from multiple connections in WAL mode
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)

	case PostgreSQL, MariaDB, MySQL:
		// TODO: Implement other database drivers
		return nil, fmt.Errorf("database driver %s not yet implemented", driver)

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	engine := &Engine{
		driver: driver,
		db:     db,
		path:   connectionString,
	}

	// Initialize schema
	if err := engine.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Database initialized successfully (%s)", driver)
	return engine, nil
}

func (e *Engine) initializeSchema() error {
	// Check if this is a new database
	var tableCount int
	err := e.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	if err != nil {
		return err
	}

	// If no tables exist, create the schema
	if tableCount == 0 {
		log.Println("Initializing database schema...")
		if err := e.createSchema(); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		if err := e.insertDefaultData(); err != nil {
			return fmt.Errorf("failed to insert default data: %w", err)
		}
	} else {
		// Check and apply migrations
		if err := e.checkMigrations(); err != nil {
			return fmt.Errorf("failed to check migrations: %w", err)
		}
	}

	return nil
}

func (e *Engine) createSchema() error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute schema in transaction
	if _, err := tx.Exec(Schema); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return tx.Commit()
}

func (e *Engine) insertDefaultData() error {
	// All default data is now in the Schema constant
	// This function is kept for backward compatibility but does nothing
	return nil
}


func (e *Engine) checkMigrations() error {
	// TODO: Implement migration system
	return nil
}

func (e *Engine) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

func (e *Engine) IsFirstRun() (bool, error) {
	var count int
	err := e.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (e *Engine) Begin() (*sql.Tx, error) {
	return e.db.Begin()
}

func (e *Engine) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return e.db.Query(query, args...)
}

func (e *Engine) QueryRow(query string, args ...interface{}) *sql.Row {
	return e.db.QueryRow(query, args...)
}

func (e *Engine) Exec(query string, args ...interface{}) (sql.Result, error) {
	return e.db.Exec(query, args...)
}

func (e *Engine) GetSetting(key string) (string, error) {
	var value string
	err := e.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}

func (e *Engine) SetSetting(key, value string, userID *int) error {
	_, err := e.Exec(`
		UPDATE settings
		SET value = ?, updated_at = ?, updated_by = ?
		WHERE key = ?
	`, value, time.Now(), userID, key)
	return err
}