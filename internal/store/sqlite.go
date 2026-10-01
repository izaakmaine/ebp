package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Sentinel errors.
var (
	ErrNotFound           = errors.New("not found")
	ErrDebtAlreadyRetired = errors.New("debt item already retired")
	ErrDebtNotPresent     = errors.New("debt item not present on idea")
	ErrInvalidDebtItem    = errors.New("invalid debt item")
	ErrPromotionBlocked   = errors.New("promotion blocked: outstanding debt or final truth claim")
)

// Querier is the common interface for *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// DB wraps a Querier (either *sql.DB or *sql.Tx) with migration support.
type DB struct {
	Querier
}

// Open opens a SQLite database and applies migrations.
func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	wrapped := &DB{db}
	if err := wrapped.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return wrapped, nil
}

// OpenTest opens an in-memory SQLite database for testing.
func OpenTest() (*DB, error) {
	db, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open test database: %w", err)
	}
	db.SetMaxOpenConns(1) // in-memory databases are per-connection
	wrapped := &DB{db}
	if err := wrapped.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate test database: %w", err)
	}
	return wrapped, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	if conn, ok := db.Querier.(*sql.DB); ok {
		return conn.Close()
	}
	return nil
}

// migrate applies all embedded SQL migrations.
func (db *DB) migrate() error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, entry := range entries {
		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := db.ExecContext(context.Background(), string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// Transaction executes fn within an exclusive transaction. Auto-rollback on error or panic.
// Uses sql.LevelSerializable which maps to BEGIN EXCLUSIVE in SQLite,
// serializing all writes and preventing read-then-write races between
// concurrent transactions (e.g., Promote ↔ MarkFinalTruthClaim).
func (db *DB) Transaction(fn func(tx *DB) error) error {
	conn, ok := db.Querier.(*sql.DB)
	if !ok {
		return fmt.Errorf("cannot begin transaction on non-root connection")
	}
	stdTx, err := conn.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return fmt.Errorf("begin serializable transaction: %w", err)
	}
	txDB := &DB{stdTx}
	defer stdTx.Rollback() // no-op after successful commit
	if err := fn(txDB); err != nil {
		return err
	}
	return stdTx.Commit()
}
