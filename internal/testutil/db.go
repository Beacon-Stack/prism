package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/beacon-stack/prism/internal/db"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
)

// testDSN returns the Postgres DSN for tests. It reads from the
// PRISM_TEST_DSN environment variable. If unset, it falls back to a
// sensible local default.
func testDSN() string {
	if dsn := os.Getenv("PRISM_TEST_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://localhost:5432/prism_test?sslmode=disable"
}

// NewTestDB creates a fresh Postgres database with all migrations applied.
// The database is registered with t.Cleanup to be closed after the test completes.
// Each call creates an isolated schema so tests never share state.
func NewTestDB(t *testing.T) *dbgen.Queries {
	t.Helper()
	q, _ := newTestDBInternal(t)
	return q
}

// NewTestDBWithSQL returns both the Queries and the underlying *sql.DB.
// Use this when you need to execute raw SQL in tests (e.g. for low-level assertions).
func NewTestDBWithSQL(t *testing.T) (*dbgen.Queries, *sql.DB) {
	t.Helper()
	return newTestDBInternal(t)
}

var schemaCounter atomic.Uint64

func newTestDBInternal(t *testing.T) (*dbgen.Queries, *sql.DB) {
	t.Helper()

	// Generate a unique schema name. Uses a process-wide atomic counter plus
	// the test name so parallel tests get distinct schemas.
	schema := fmt.Sprintf("test_%s_%d_%d", sanitizeSchemaName(t.Name()), os.Getpid(), schemaCounter.Add(1))

	// First connection: create the schema. This uses a short-lived pool
	// pointed at the bare database, then we re-open with search_path baked
	// into the DSN so every connection in the pool sees the right schema.
	bootstrap, err := sql.Open("pgx", testDSN())
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open bootstrap: %v", err)
	}
	if _, err := bootstrap.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		bootstrap.Close()
		t.Fatalf("testutil.NewTestDB: create schema: %v", err)
	}
	bootstrap.Close()

	// Re-open with search_path in the DSN options so every connection in
	// the pool automatically uses this test's schema. pgx honors
	// ?search_path= as a connection parameter.
	dsn := testDSN()
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsnWithSchema := fmt.Sprintf("%s%soptions=-c+search_path=%s", dsn, sep, schema)

	sqlDB, err := sql.Open("pgx", dsnWithSchema)
	if err != nil {
		t.Fatalf("testutil.NewTestDB: open postgres: %v", err)
	}

	t.Cleanup(func() {
		// Drop the schema from a fresh connection (not the test's pool, which may
		// have connections still open).
		cleanup, err := sql.Open("pgx", testDSN())
		if err == nil {
			_, _ = cleanup.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
			cleanup.Close()
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("testutil.NewTestDB: close db: %v", err)
		}
	})

	if err := db.Migrate(sqlDB, "postgres"); err != nil {
		t.Fatalf("testutil.NewTestDB: migrate: %v", err)
	}

	return dbgen.New(sqlDB), sqlDB
}

// sanitizeSchemaName replaces non-alphanumeric characters with underscores.
func sanitizeSchemaName(name string) string {
	out := make([]byte, len(name))
	for i, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			out[i] = c
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}
