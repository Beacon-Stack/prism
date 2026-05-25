package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/beacon-stack/prism/internal/config"
	"github.com/beacon-stack/prism/internal/db"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
)

// NewTestDB creates a fresh SQLite database with all migrations applied.
// The database is registered with t.Cleanup to be closed after the test
// completes. Each call gets its own file under t.TempDir() so tests never
// share state.
func NewTestDB(t *testing.T) *dbgen.Queries {
	t.Helper()
	q, _ := newTestDBInternal(t)
	return q
}

// NewTestDBWithSQL returns both the Queries and the underlying *sql.DB.
// Use this when you need to execute raw SQL in tests (e.g. for low-level
// assertions).
func NewTestDBWithSQL(t *testing.T) (*dbgen.Queries, *sql.DB) {
	t.Helper()
	return newTestDBInternal(t)
}

func newTestDBInternal(t *testing.T) (*dbgen.Queries, *sql.DB) {
	t.Helper()

	database, err := db.Open(config.DatabaseConfig{
		Path: filepath.Join(t.TempDir(), "prism-test.db"),
	})
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("testutil.NewTestDB: close db: %v", err)
		}
	})

	if err := db.Migrate(database.SQL); err != nil {
		t.Fatalf("testutil.NewTestDB: migrate: %v", err)
	}

	return dbgen.New(database.SQL), database.SQL
}
