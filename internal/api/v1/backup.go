package v1

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BackupHandler returns an http.HandlerFunc that streams a pg_dump SQL dump of
// the database as a file download.
//
// Registered directly on the chi router (not via huma) because huma wraps all
// responses in JSON, which is unsuitable for file downloads.
func BackupHandler(dsn string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmp, err := os.CreateTemp("", "prism-backup-*.sql")
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to create temp file", slog.Any("error", err))
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}
		tmpPath := tmp.Name()
		tmp.Close()
		defer func() { _ = os.Remove(tmpPath) }()

		// gosec G204: pg_dump's only variable input is the DSN, which comes
		// from pilot's own config — not from user-supplied request data.
		cmd := exec.CommandContext(r.Context(), "pg_dump", "--file="+tmpPath, "--format=plain", dsn) //nolint:gosec
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.ErrorContext(r.Context(), "backup: pg_dump failed",
				slog.Any("error", err),
				slog.String("output", string(output)),
			)
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}

		f, err := os.Open(tmpPath)
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to open dump file", slog.Any("error", err))
			http.Error(w, "failed to read backup", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		filename := fmt.Sprintf("prism-backup-%s.sql", time.Now().UTC().Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			logger.WarnContext(r.Context(), "backup: error streaming response", slog.Any("error", err))
		}
	}
}

// RestoreHandler returns an http.HandlerFunc that accepts a SQL dump file upload
// (application/sql or application/octet-stream) and applies it to the database
// using psql.
func RestoreHandler(dsn string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Override the global 1 MiB body limit — restore files can be large.
		const maxRestoreSize = 500 << 20 // 500 MiB
		r.Body = http.MaxBytesReader(w, r.Body, maxRestoreSize)

		tmp, err := os.CreateTemp("", "prism-restore-*.sql")
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

		// Validate: the file should start with a recognizable SQL dump header.
		if err := validateSQLDump(tmpPath); err != nil {
			logger.WarnContext(r.Context(), "restore: uploaded file is not a valid SQL dump", slog.Any("error", err))
			http.Error(w, "uploaded file is not a valid SQL dump", http.StatusBadRequest)
			return
		}

		cmd := exec.CommandContext(r.Context(), "psql", dsn, "-f", tmpPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.ErrorContext(r.Context(), "restore: psql failed",
				slog.Any("error", err),
				slog.String("output", string(output)),
			)
			http.Error(w, "failed to apply restore", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Database restored from SQL dump."}`))
	}
}

// validateSQLDump checks that the file starts with a plausible SQL dump header.
func validateSQLDump(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 256)
	n, err := f.Read(header)
	if err != nil && n == 0 {
		return fmt.Errorf("file is empty or unreadable: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("file is empty")
	}
	s := strings.ToLower(string(header[:n]))
	if strings.Contains(s, "pg_dump") || strings.HasPrefix(s, "--") || strings.HasPrefix(s, "set ") || strings.HasPrefix(s, "create ") {
		return nil
	}
	return fmt.Errorf("file does not appear to be a pg_dump SQL dump")
}
