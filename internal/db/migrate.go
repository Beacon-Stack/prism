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
func Migrate(sqlDB *sql.DB, driver string) error {
	// goose global state (SetBaseFS, SetDialect) is not goroutine-safe,
	// so initialise it exactly once.
	var initErr error
	gooseInit.Do(func() {
		goose.SetBaseFS(migrationsFS)

		// goose dialect name for SQLite is "sqlite3".
		dialect := driver
		if driver == "sqlite" {
			dialect = "sqlite3"
		}

		initErr = goose.SetDialect(dialect)
	})
	if initErr != nil {
		return fmt.Errorf("setting goose dialect: %w", initErr)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
