package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sync"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var gooseInit sync.Once

// Migrate runs all pending database migrations.
// It is safe to call on every startup — goose is idempotent.
func Migrate(sqlDB *sql.DB) error {
	// goose global state (SetBaseFS, SetDialect) is not goroutine-safe,
	// so initialise it exactly once.
	var initErr error
	gooseInit.Do(func() {
		goose.SetBaseFS(migrationsFS)
		initErr = goose.SetDialect("sqlite3")
	})
	if initErr != nil {
		return fmt.Errorf("setting goose dialect: %w", initErr)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
