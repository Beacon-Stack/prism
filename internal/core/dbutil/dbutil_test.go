package dbutil

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMergeSettings(t *testing.T) {
	existing := json.RawMessage(`{"url":"http://localhost","password":"secret"}`)
	update := json.RawMessage(`{"url":"http://newhost"}`)

	merged := MergeSettings(existing, update)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(merged, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["url"]) != `"http://newhost"` {
		t.Errorf("url should be updated, got %s", m["url"])
	}
	if string(m["password"]) != `"secret"` {
		t.Errorf("password should be preserved, got %s", m["password"])
	}
}

func TestMergeSettingsEmptyNew(t *testing.T) {
	existing := json.RawMessage(`{"url":"http://localhost"}`)
	merged := MergeSettings(existing, nil)
	if string(merged) != string(existing) {
		t.Errorf("empty new should return existing, got %s", merged)
	}
}

// TestIsUniqueViolation_RealConstraint verifies the helper against a real
// SQLite UNIQUE-constraint failure: nothing else reliably distinguishes the
// extended-result-code path we depend on.
func TestIsUniqueViolation_RealConstraint(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, k TEXT UNIQUE)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO t (k) VALUES ('a')"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO t (k) VALUES ('a')")
	if err == nil {
		t.Fatal("expected a unique-constraint error")
	}
	if !IsUniqueViolation(err) {
		t.Errorf("IsUniqueViolation(%v) = false; want true", err)
	}
}

func TestIsUniqueViolation_PlainError(t *testing.T) {
	if IsUniqueViolation(errors.New("some random error")) {
		t.Error("plain error should not be a unique violation")
	}
}

func TestIsUniqueViolation_Nil(t *testing.T) {
	if IsUniqueViolation(nil) {
		t.Error("nil error should not be a unique violation")
	}
}

func TestIsUniqueViolation_WrappedPlainError(t *testing.T) {
	err := fmt.Errorf("inserting: %w", errors.New("something else"))
	if IsUniqueViolation(err) {
		t.Error("wrapped plain error should not be a unique violation")
	}
}
