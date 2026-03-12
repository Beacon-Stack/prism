package v1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSQLiteMagic_ValidDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Write a valid SQLite header (16 bytes) followed by some junk.
	data := append([]byte("SQLite format 3\000"), make([]byte, 100)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteMagic(path); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateSQLiteMagic_InvalidHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.db")

	if err := os.WriteFile(path, []byte("not a sqlite file!!"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validateSQLiteMagic(path)
	if err == nil {
		t.Fatal("expected error for non-SQLite file, got nil")
	}
}

func TestValidateSQLiteMagic_TooSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.db")

	// Only 5 bytes — less than the 16-byte header.
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := validateSQLiteMagic(path)
	if err == nil {
		t.Fatal("expected error for too-small file, got nil")
	}
}

func TestValidateSQLiteMagic_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := validateSQLiteMagic(path)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestValidateSQLiteMagic_NonExistent(t *testing.T) {
	err := validateSQLiteMagic("/tmp/definitely-does-not-exist-abcxyz.db")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidateSQLiteMagic_ExactlyHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exact.db")

	// Exactly 16 bytes — valid header with no trailing data.
	if err := os.WriteFile(path, []byte("SQLite format 3\000"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := validateSQLiteMagic(path); err != nil {
		t.Errorf("expected nil for exact header, got %v", err)
	}
}
