package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver

	"github.com/beacon-stack/prism/internal/config"
)

// DB wraps the underlying sql.DB and tracks which driver is in use.
type DB struct {
	SQL    *sql.DB
	Driver string
}

// Open opens a database connection based on the provided configuration.
// The caller is responsible for calling Close when done.
func Open(cfg config.DatabaseConfig) (*DB, error) {
	switch cfg.Driver {
	case "postgres", "":
		return openPostgres(cfg.DSN.Value())
	default:
		return nil, fmt.Errorf("unsupported database driver: %q (must be postgres)", cfg.Driver)
	}
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.SQL.Close()
}

func openPostgres(dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres DSN must not be empty")
	}

	// pgx/stdlib registers the "pgx" driver name for database/sql compatibility.
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres database: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("pinging postgres database: %w", err)
	}

	return &DB{SQL: sqlDB, Driver: "postgres"}, nil
}
