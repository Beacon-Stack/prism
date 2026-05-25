package v1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSQLiteDB_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// A real SQLite file begins with the 16-byte magic header.
	data := append([]byte(sqliteMagic), make([]byte, 256)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteDB(path); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateSQLiteDB_InvalidHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.db")

	if err := os.WriteFile(path, []byte("-- PostgreSQL database dump\nSET statement_timeout = 0;\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteDB(path); err == nil {
		t.Fatal("expected error for non-SQLite file, got nil")
	}
}

func TestValidateSQLiteDB_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteDB(path); err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestValidateSQLiteDB_TooSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.db")

	if err := os.WriteFile(path, []byte("SQLite"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteDB(path); err == nil {
		t.Fatal("expected error for truncated file, got nil")
	}
}

func TestValidateSQLiteDB_NonExistent(t *testing.T) {
	if err := validateSQLiteDB("/tmp/definitely-does-not-exist-abcxyz.db"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
