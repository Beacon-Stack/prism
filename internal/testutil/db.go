package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/beacon-stack/prism/internal/db"
	dbsqlite "github.com/beacon-stack/prism/internal/db/generated/sqlite"
)

const testDSN = ":memory:?_foreign_keys=ON"

// NewTestDB creates a fresh in-memory SQLite database with all migrations applied.
// The database is registered with t.Cleanup to be closed after the test completes.
// Each call returns an independent database — tests never share state.
func NewTestDB(t *testing.T) *dbsqlite.Queries {
	t.Helper()
	q, _ := newTestDBInternal(t)
	return q
}

// NewTestDBWithSQL returns both the Queries and the underlying *sql.DB.
// Use this when you need to execute raw SQL in tests (e.g. for low-level assertions).
func NewTestDBWithSQL(t *testing.T) (*dbsqlite.Queries, *sql.DB) {
	t.Helper()
	return newTestDBInternal(t)
}

func newTestDBInternal(t *testing.T) (*dbsqlite.Queries, *sql.DB) {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", testDSN)
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open sqlite: %v", err)
	}
	// In-memory SQLite gives each connection its own database.
	// Limit to one connection so migrations and queries share the same DB.
	sqlDB.SetMaxOpenConns(1)

	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("testutil.NewTestDB: close db: %v", err)
		}
	})

	if err := db.Migrate(sqlDB, "sqlite"); err != nil {
		t.Fatalf("testutil.NewTestDB: migrate: %v", err)
	}

	return dbsqlite.New(sqlDB), sqlDB
}
