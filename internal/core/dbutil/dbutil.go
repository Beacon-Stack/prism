// Package dbutil provides shared helpers for database-layer operations.
package dbutil

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// NullString converts a *string to sql.NullString. A nil or empty pointer
// yields a null value.
func NullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

// NullStringFromString converts a string to sql.NullString. An empty string
// yields a null value.
func NullStringFromString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullStringPtr converts a sql.NullString back to a *string. Invalid values
// yield nil.
func NullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// NullStringValue returns the string value of a sql.NullString, or "" if invalid.
func NullStringValue(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

// NullInt32 converts a *int to sql.NullInt32. A nil pointer yields a null value.
func NullInt32(p *int) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*p), Valid: true}
}

// NullInt32FromInt64Ptr converts a *int64 to sql.NullInt32.
func NullInt32FromInt64Ptr(p *int64) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*p), Valid: true}
}

// NullInt32Value returns the int value of a sql.NullInt32, or 0 if invalid.
func NullInt32Value(ns sql.NullInt32) int {
	if !ns.Valid {
		return 0
	}
	return int(ns.Int32)
}

// NullTime converts a *time.Time to sql.NullTime. A nil pointer yields a null value.
func NullTime(p *time.Time) sql.NullTime {
	if p == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *p, Valid: true}
}

// NullTimePtr converts a sql.NullTime back to a *time.Time. Invalid values yield nil.
func NullTimePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
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

// IsUniqueViolation reports whether err is a PostgreSQL unique constraint violation
// (error code 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
