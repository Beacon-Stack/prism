package dbutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
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

func TestIsUniqueViolation_PgError(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	if !IsUniqueViolation(err) {
		t.Error("23505 PgError should be a unique violation")
	}
}

func TestIsUniqueViolation_PlainError(t *testing.T) {
	err := errors.New("some random error")
	if IsUniqueViolation(err) {
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
