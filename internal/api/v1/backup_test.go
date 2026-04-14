package v1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSQLDump_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")

	data := []byte("-- PostgreSQL database dump\n\nSET statement_timeout = 0;\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLDump(path); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateSQLDump_InvalidHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.sql")

	if err := os.WriteFile(path, []byte("SQLite format 3\000not a sql dump"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validateSQLDump(path)
	if err == nil {
		t.Fatal("expected error for non-SQL-dump file, got nil")
	}
}

func TestValidateSQLDump_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.sql")

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := validateSQLDump(path)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestValidateSQLDump_NonExistent(t *testing.T) {
	err := validateSQLDump("/tmp/definitely-does-not-exist-abcxyz.sql")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidateSQLDump_PgDumpHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pgdump.sql")

	data := []byte("--\n-- PostgreSQL database dump\n-- Dumped from pg_dump version 16.2\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLDump(path); err != nil {
		t.Errorf("expected nil for pg_dump header, got %v", err)
	}
}
