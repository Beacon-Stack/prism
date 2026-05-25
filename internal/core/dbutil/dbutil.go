// Package dbutil provides shared helpers for database-layer operations.
package dbutil

import (
	"encoding/json"
	"errors"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// SQLite/sqlc note: with `emit_pointers_for_null_types: true`, sqlc-sqlite
// generates `*string` / `*int64` / `*time.Time` for nullable columns rather
// than `sql.NullXxx` structs. The helpers below operate on those pointer
// shapes directly so call sites do not have to mix types.

// NullString returns p unchanged. It exists as a no-op for callers that built
// against the older sql.NullString-based API; keep it so adoption is grep-free.
// Empty strings are preserved (they are not coerced to nil) — callers that
// want "" treated as NULL should call NullStringFromString.
func NullString(p *string) *string {
	return p
}

// NullStringFromString converts a string to a nullable *string. An empty
// string yields nil (so it stores as SQL NULL).
func NullStringFromString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NullStringPtr is the identity for *string. Keeps the call-site name stable
// against the pre-SQLite signature, which converted sql.NullString → *string.
func NullStringPtr(p *string) *string {
	return p
}

// NullStringValue returns the dereferenced value of p, or "" if nil.
func NullStringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// NullInt32 converts a *int to a *int64. A nil pointer yields nil.
func NullInt32(p *int) *int64 {
	if p == nil {
		return nil
	}
	v := int64(*p)
	return &v
}

// NullInt32FromInt64Ptr returns p unchanged; SQLite-sqlc emits *int64 for
// nullable integer columns. Kept under the old name so existing call sites
// continue to compile against the migrated API.
func NullInt32FromInt64Ptr(p *int64) *int64 {
	return p
}

// NullInt32Value returns the dereferenced value of p, or 0 if nil.
func NullInt32Value(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// NullTime converts a *time.Time to its RFC3339 UTC string representation.
// A nil pointer yields nil so the column stores as SQL NULL.
//
// TIMESTAMPTZ columns are stored as TEXT in SQLite (see migration squash);
// using RFC3339 keeps lexicographic ordering identical to chronological
// ordering, which several queries rely on.
func NullTime(p *time.Time) *string {
	if p == nil {
		return nil
	}
	s := p.UTC().Format(time.RFC3339Nano)
	return &s
}

// NullTimePtr parses an RFC3339-encoded *string back to *time.Time. Nil or
// unparseable input yields nil.
func NullTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		// Some legacy callers may pass plain RFC3339 (no fractional secs).
		t, err = time.Parse(time.RFC3339, *s)
		if err != nil {
			return nil
		}
	}
	return &t
}

// MergeSettings returns newSettings with any keys absent from newSettings
// filled in from existing. Keys present in newSettings always win.
// This is used to preserve secret fields (passwords, API keys) when the
// frontend omits them from an update request.
func MergeSettings(existing, newSettings json.RawMessage) json.RawMessage {
	if len(newSettings) == 0 {
		return existing
	}
	var existingMap, newMap map[string]json.RawMessage
	if json.Unmarshal(existing, &existingMap) != nil {
		return newSettings
	}
	if json.Unmarshal(newSettings, &newMap) != nil {
		return newSettings
	}
	for k, v := range existingMap {
		if _, ok := newMap[k]; !ok {
			newMap[k] = v
		}
	}
	merged, err := json.Marshal(newMap)
	if err != nil {
		return newSettings
	}
	return merged
}

// IsUniqueViolation reports whether err is a SQLite unique constraint violation
// (extended result codes SQLITE_CONSTRAINT_UNIQUE / SQLITE_CONSTRAINT_PRIMARYKEY).
func IsUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	switch sqliteErr.Code() {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
		return true
	}
	return false
}
