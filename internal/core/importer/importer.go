// Package importer moves completed downloads into the library directory tree,
// creates movie_file records, and updates movie status.
package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/beacon-stack/prism/internal/core/downloadhandling"
	"github.com/beacon-stack/prism/internal/core/mediainfo"
	"github.com/beacon-stack/prism/internal/core/mediamanagement"
	"github.com/beacon-stack/prism/internal/core/pathutil"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/internal/core/renamer"
	"github.com/beacon-stack/prism/internal/db"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// videoExtensions is the set of file extensions considered video files.
var videoExtensions = map[string]bool{
	".mkv":  true,
	".mp4":  true,
	".avi":  true,
	".ts":   true,
	".m2ts": true,
	".mov":  true,
	".wmv":  true,
}

// Service subscribes to TypeDownloadDone events and imports completed files
// into the library directory tree.
type Service struct {
	q        dbgen.Querier
	sqlDB    *sql.DB // for transactions; nil in tests
	bus      *events.Bus
	logger   *slog.Logger
	mm       *mediamanagement.Service
	dh       *downloadhandling.Service
	mediaSvc *mediainfo.Service // nil = no scanning
}

// NewService creates a new Service.
func NewService(q dbgen.Querier, bus *events.Bus, logger *slog.Logger, mm *mediamanagement.Service, dh *downloadhandling.Service, mediaSvc *mediainfo.Service, sqlDB ...*sql.DB) *Service {
	s := &Service{q: q, bus: bus, logger: logger, mm: mm, dh: dh, mediaSvc: mediaSvc}
	if len(sqlDB) > 0 {
		s.sqlDB = sqlDB[0]
	}
	return s
}

// Subscribe registers the importer handler on the event bus.
// Call this once during application startup.
func (s *Service) Subscribe() {
	s.bus.Subscribe(func(ctx context.Context, e events.Event) {
		if e.Type != events.TypeDownloadDone {
			return
		}
		// Respect the "Enable Completed Download Handling" toggle.
		if s.dh != nil {
			dhs, err := s.dh.Get(ctx)
			if err != nil {
				s.logger.Warn("import: failed to load download handling settings, skipping import", "error", err)
				return
			}
			if !dhs.EnableCompleted {
				s.logger.Debug("import: completed download handling disabled, skipping")
				return
			}
		}
		grabID, _ := e.Data["grab_id"].(string)
		contentPath, _ := e.Data["content_path"].(string)
		if grabID == "" {
			s.logger.Warn("import: TypeDownloadDone event missing grab_id")
			return
		}
		if err := s.importFile(ctx, grabID, contentPath); err != nil {
			s.logger.Error("import failed",
				"grab_id", grabID,
				"content_path", contentPath,
				"error", err,
			)
			if s.bus != nil {
				// Best-effort: look up the movie title for the
				// activity log row. Without it, the Activity page
				// renders "Import failed for " — uninformative
				// when the user is trying to triage failures.
				data := map[string]any{
					"grab_id": grabID,
					"reason":  err.Error(),
				}
				if grab, gErr := s.q.GetGrabByID(ctx, grabID); gErr == nil {
					data["release_title"] = grab.ReleaseTitle
					if mov, mErr := s.q.GetMovie(ctx, grab.MovieID); mErr == nil {
						data["movie_title"] = mov.Title
					}
				}
				s.bus.Publish(ctx, events.Event{
					Type:    events.TypeImportFailed,
					MovieID: e.MovieID,
					Data:    data,
				})
			}
		}
	})
}

// ImportFromHaulRecord runs the import pipeline against a Haul history
// record without requiring a pre-existing Prism grab_history row.
// Used by the "import from Haul" flow on the movie detail page and
// the Activity-page "downloaded but not in library" rail — surfaces
// files Haul has but Prism's library doesn't know about (because
// the original grab record was lost, the torrent was added directly
// in Haul, or the file pre-dates the importer).
//
// movieID is the target movie in Prism's DB; contentPath is the
// absolute path Haul reports as save_path/name. Quality + edition
// are zero-valued (importer derives what it can from filename when
// the grab is empty).
func (s *Service) ImportFromHaulRecord(ctx context.Context, movieID, contentPath string) error {
	s.logger.Info("import started (from Haul record)", "movie_id", movieID, "content_path", contentPath)

	// Synthesize a minimal grab. No grab_history row is created —
	// this import is orthogonal to the grab pipeline. The
	// ID field is left empty so the TypeImportComplete event's
	// grab_id field is "" (callers can detect synthetic imports
	// that way).
	grab := dbgen.GrabHistory{MovieID: movieID}
	return s.runImport(ctx, grab, contentPath, "")
}

// importFile performs the full import pipeline for a single completed download.
func (s *Service) importFile(ctx context.Context, grabID, contentPath string) error {
	s.logger.Info("import started", "grab_id", grabID, "content_path", contentPath)

	// ── Load context from DB ───────────────────────────────────────────────
	grab, err := s.q.GetGrabByID(ctx, grabID)
	if err != nil {
		return fmt.Errorf("loading grab %q: %w", grabID, err)
	}
	return s.runImport(ctx, grab, contentPath, grabID)
}

// runImport is the post-grab-loaded portion of importFile. Extracted
// so both the regular grab-driven path AND the import-from-Haul
// path can share it. grabID is used only for the
// TypeImportComplete event Data; pass "" when there's no associated
// grab (then the consumer sees grab_id="" in the event payload).
func (s *Service) runImport(ctx context.Context, grab dbgen.GrabHistory, contentPath string, grabID string) error {
	mov, err := s.q.GetMovie(ctx, grab.MovieID)
	if err != nil {
		return fmt.Errorf("loading movie %q: %w", grab.MovieID, err)
	}

	lib, err := s.q.GetLibrary(ctx, mov.LibraryID)
	if err != nil {
		return fmt.Errorf("loading library %q: %w", mov.LibraryID, err)
	}

	// ── Resolve source file ────────────────────────────────────────────────
	srcPath, err := resolveSourceFile(contentPath)
	if err != nil {
		return fmt.Errorf("resolving source file from %q: %w", contentPath, err)
	}

	// ── Load media management settings ─────────────────────────────────────
	mm, err := s.mm.Get(ctx)
	if err != nil {
		return fmt.Errorf("loading media management settings: %w", err)
	}

	// ── Reconstruct quality + compute destination ──────────────────────────
	q := qualityFromGrab(grab)

	fileFormat := mm.StandardMovieFormat
	if lib.NamingFormat != nil && *lib.NamingFormat != "" {
		fileFormat = *lib.NamingFormat
	}
	folderFormat := mm.MovieFolderFormat
	if lib.FolderFormat != nil && *lib.FolderFormat != "" {
		folderFormat = *lib.FolderFormat
	}

	var fileEdition string
	if grab.ReleaseEdition != nil {
		fileEdition = *grab.ReleaseEdition
	}
	rm := renamer.Movie{
		Title:         mov.Title,
		OriginalTitle: mov.OriginalTitle,
		Year:          int(mov.Year),
		Edition:       fileEdition,
	}
	ext := filepath.Ext(srcPath)
	colon := renamer.ColonReplacement(mm.ColonReplacement)

	var destPath string
	if !mm.RenameMovies {
		destPath = filepath.Join(lib.RootPath,
			renamer.FolderName(folderFormat, rm),
			filepath.Base(srcPath))
	} else {
		destPath = renamer.DestPath(lib.RootPath, fileFormat, folderFormat, rm, q, colon, ext)
	}

	// ── Create destination directory ───────────────────────────────────────
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// ── Transfer the file (hardlink preferred, copy+delete as fallback) ────
	if err := transferFile(srcPath, destPath); err != nil {
		return fmt.Errorf("transferring file %q → %q: %w", srcPath, destPath, err)
	}

	// ── Copy extra files (subtitles, NFOs, etc.) ───────────────────────────
	if mm.ImportExtraFiles && len(mm.ExtraFileExtensions) > 0 {
		srcDir := filepath.Dir(srcPath)
		destDir := filepath.Dir(destPath)
		copyExtraFiles(s.logger, srcDir, destDir, mm.ExtraFileExtensions)
	}

	// ── Persist movie_file record + update movie status (atomic) ──────────
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("stat after transfer: %w", err)
	}

	qualityJSON, err := json.Marshal(q)
	if err != nil {
		return fmt.Errorf("marshaling quality: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	fileID := uuid.New().String()

	dbOps := func(dbq dbgen.Querier) error {
		if _, err := dbq.CreateMovieFile(ctx, dbgen.CreateMovieFileParams{
			ID:          fileID,
			MovieID:     grab.MovieID,
			Path:        destPath,
			SizeBytes:   info.Size(),
			QualityJson: string(qualityJSON),
			ImportedAt:  now,
			IndexedAt:   now,
		}); err != nil {
			return fmt.Errorf("creating movie_file record: %w", err)
		}

		if _, err := dbq.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{
			Path:      &destPath,
			UpdatedAt: now,
			ID:        grab.MovieID,
		}); err != nil {
			return fmt.Errorf("updating movie path: %w", err)
		}

		if _, err := dbq.UpdateMovieStatus(ctx, dbgen.UpdateMovieStatusParams{
			Status:    "downloaded",
			UpdatedAt: now,
			ID:        grab.MovieID,
		}); err != nil {
			return fmt.Errorf("updating movie status: %w", err)
		}
		return nil
	}

	if s.sqlDB != nil {
		if err := db.RunInTx(ctx, s.sqlDB, dbOps); err != nil {
			return err
		}
	} else {
		if err := dbOps(s.q); err != nil {
			return err
		}
	}

	// ── Trigger mediainfo scan (fire-and-forget) ───────────────────────────
	if s.mediaSvc != nil && s.mediaSvc.Available() {
		go func() {
			scanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if scanErr := s.mediaSvc.ScanFile(scanCtx, fileID, destPath); scanErr != nil {
				s.logger.Debug("mediainfo scan failed", "file_id", fileID, "error", scanErr)
			}
		}()
	}

	// ── Publish success event ──────────────────────────────────────────────
	// movie_title + quality are read by the activity classifier — see
	// classify() in internal/core/activity/service.go. The previous
	// emit omitted both, leaving activity rows rendered as "Imported "
	// with an empty title.
	qualityLabel := string(q.Resolution)
	if q.Source != "" {
		qualityLabel = fmt.Sprintf("%s %s", q.Resolution, q.Source)
	}
	s.bus.Publish(ctx, events.Event{
		Type:    events.TypeImportComplete,
		MovieID: grab.MovieID,
		Data: map[string]any{
			"grab_id":     grabID,
			"dest_path":   destPath,
			"movie_title": mov.Title,
			"quality":     qualityLabel,
		},
	})

	s.logger.Info("import complete",
		"movie_id", grab.MovieID,
		"dest_path", destPath,
	)
	return nil
}

// validateContentPath delegates to the shared pathutil package.
var validateContentPath = pathutil.ValidateContentPath

// resolveSourceFile returns the path to the video file to import.
// If contentPath is a regular file, it is returned directly.
// If it is a directory, the largest video file inside it is returned.
func resolveSourceFile(contentPath string) (string, error) {
	if err := validateContentPath(contentPath); err != nil {
		return "", err
	}

	info, err := os.Stat(contentPath)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		if !videoExtensions[filepath.Ext(contentPath)] {
			return "", fmt.Errorf("not a recognised video extension: %q", filepath.Ext(contentPath))
		}
		return contentPath, nil
	}

	// Directory: walk and find the largest video file.
	type candidate struct {
		path string
		size int64
	}
	var candidates []candidate

	err = filepath.WalkDir(contentPath, func(path string, d os.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		if !videoExtensions[filepath.Ext(path)] {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		candidates = append(candidates, candidate{path: path, size: fi.Size()})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking content directory: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no video file found in %q", contentPath)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].size > candidates[j].size
	})
	return candidates[0].path, nil
}

// transferFile copies src to dst.
// It tries os.Link first (same filesystem, no data copy); on failure it falls
// back to a full io.Copy. The source file is intentionally NOT deleted — the
// download client's seed lifecycle (seed ratio/time limits, "Remove Completed
// Downloads") is responsible for cleanup. Deleting the source here would break
// active torrent seeds in cross-volume (e.g. Docker) setups where hardlink
// fails and copy is used instead.
func transferFile(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	// Fallback: copy only (no source deletion).
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}

	return nil
}

// copyExtraFiles walks srcDir and hardlinks (or copies) any file whose
// extension matches one of exts into destDir. Errors are logged but do not
// abort the import.
func copyExtraFiles(logger *slog.Logger, srcDir, destDir string, exts []string) {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e != "" && e[0] != '.' {
			e = "." + e
		}
		extSet[strings.ToLower(e)] = true
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		logger.Warn("extra files: cannot read source dir", "dir", srcDir, "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !extSet[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(destDir, name)
		if err := transferFile(src, dst); err != nil {
			logger.Warn("extra files: transfer failed", "src", src, "dst", dst, "error", err)
		}
	}
}

// qualityFromGrab reconstructs a plugin.Quality from the denormalized fields
// stored in grab_history.
func qualityFromGrab(g dbgen.GrabHistory) plugin.Quality {
	res := plugin.Resolution(g.ReleaseResolution)
	src := plugin.Source(g.ReleaseSource)
	codec := plugin.Codec(g.ReleaseCodec)
	hdr := plugin.HDRFormat(g.ReleaseHdr)
	return plugin.Quality{
		Resolution: res,
		Source:     src,
		Codec:      codec,
		HDR:        hdr,
		Name:       quality.BuildName(res, src, codec, hdr),
	}
}
