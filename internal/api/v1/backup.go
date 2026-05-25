package v1

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// sqliteMagic is the 16-byte header every SQLite 3 database file begins with.
const sqliteMagic = "SQLite format 3\x00"

// BackupHandler returns an http.HandlerFunc that streams a consistent snapshot
// of the SQLite database as a file download.
//
// Registered directly on the chi router (not via huma) because huma wraps all
// responses in JSON, which is unsuitable for file downloads.
//
// The snapshot is produced with `VACUUM INTO`, which writes a fully consistent
// copy even while the live database is in WAL mode and being written to.
func BackupHandler(db *sql.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmp, err := os.CreateTemp("", "prism-backup-*.db")
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to create temp file", slog.Any("error", err))
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}
		tmpPath := tmp.Name()
		tmp.Close()
		// VACUUM INTO requires the target not to exist.
		_ = os.Remove(tmpPath)
		defer func() { _ = os.Remove(tmpPath) }()

		// SQLite VACUUM INTO accepts a bind parameter for the destination
		// path; tmpPath comes from os.CreateTemp, not from request data,
		// but parameterising it sidesteps any quoting concerns and the
		// gosec G202 SQL-injection lint.
		if _, err := db.ExecContext(r.Context(), "VACUUM INTO ?", tmpPath); err != nil {
			logger.ErrorContext(r.Context(), "backup: VACUUM INTO failed", slog.Any("error", err))
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}

		f, err := os.Open(tmpPath)
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to open snapshot file", slog.Any("error", err))
			http.Error(w, "failed to read backup", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		filename := fmt.Sprintf("prism-backup-%s.db", time.Now().UTC().Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			logger.WarnContext(r.Context(), "backup: error streaming response", slog.Any("error", err))
		}
	}
}

// RestoreHandler returns an http.HandlerFunc that accepts a SQLite database
// file upload and replaces the live database file with it.
//
// The replacement takes effect on the next process start: SQLite holds the
// file open, so the handler writes the upload alongside the live database and
// the operator restarts the service to load it. The response says as much.
func RestoreHandler(dbPath string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Override the global 1 MiB body limit — restore files can be large.
		const maxRestoreSize = 500 << 20 // 500 MiB
		r.Body = http.MaxBytesReader(w, r.Body, maxRestoreSize)

		tmp, err := os.CreateTemp("", "prism-restore-*.db")
		if err != nil {
			logger.ErrorContext(r.Context(), "restore: failed to create temp file", slog.Any("error", err))
			http.Error(w, "failed to write restore file", http.StatusInternalServerError)
			return
		}
		tmpPath := tmp.Name()
		defer func() { _ = os.Remove(tmpPath) }()

		if _, err := io.Copy(tmp, r.Body); err != nil {
			tmp.Close()
			logger.ErrorContext(r.Context(), "restore: failed to write body", slog.Any("error", err))
			http.Error(w, "failed to write restore file", http.StatusInternalServerError)
			return
		}
		tmp.Close()

		// Validate: the file must begin with the SQLite database magic header.
		if err := validateSQLiteDB(tmpPath); err != nil {
			logger.WarnContext(r.Context(), "restore: uploaded file is not a SQLite database", slog.Any("error", err))
			http.Error(w, "uploaded file is not a SQLite database", http.StatusBadRequest)
			return
		}

		// Move the validated upload over the live database file. SQLite keeps
		// its handle on the old inode until restart, so the swap is safe.
		if err := os.Rename(tmpPath, dbPath); err != nil {
			logger.ErrorContext(r.Context(), "restore: failed to replace database file", slog.Any("error", err))
			http.Error(w, "failed to apply restore", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Database restored. Restart Prism to load the restored database."}`))
	}
}

// validateSQLiteDB checks that the file begins with the SQLite 3 magic header.
func validateSQLiteDB(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, len(sqliteMagic))
	n, err := io.ReadFull(f, header)
	if err != nil || n < len(sqliteMagic) {
		return fmt.Errorf("file is too small to be a SQLite database")
	}
	if string(header) != sqliteMagic {
		return fmt.Errorf("file does not appear to be a SQLite database")
	}
	return nil
}
