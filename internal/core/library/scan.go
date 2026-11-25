package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beacon-stack/prism/internal/core/edition"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
)

// ScanResult says what a library scan changed.
type ScanResult struct {
	Present  int // tracked files found where they were recorded
	Relinked int // tracked files found at a new path
	Attached int // files on disk newly linked to a movie in the library
	Missing  int // movies whose files are gone
}

// minMovieBytes is the smallest file a scan will link to a movie on its
// own. Anything smaller is a sample, a trailer or an extra.
const minMovieBytes = 100 * 1024 * 1024

// Scan brings a library's records in line with what is on disk:
//
//   - a tracked file still at its path is confirmed;
//   - a tracked file that moved (same name and size elsewhere in the
//     library) has its path corrected;
//   - a video file nothing tracks is linked to the movie it belongs to,
//     when that movie is already in the library and has no file;
//   - a movie whose files are all gone is marked missing.
//
// Files that belong to no movie in the library are left for the import
// screen (see ScanDisk). Scan is idempotent.
func (s *Service) Scan(ctx context.Context, libraryID string) (ScanResult, error) {
	var res ScanResult
	lib, err := s.Get(ctx, libraryID)
	if err != nil {
		return res, err
	}
	tracked, err := s.q.ListMovieFilesByLibrary(ctx, libraryID)
	if err != nil {
		return res, fmt.Errorf("listing movie files for library %q: %w", libraryID, err)
	}
	movies, err := s.q.ListMoviesByLibrary(ctx, dbgen.ListMoviesByLibraryParams{LibraryID: libraryID, Limit: 1_000_000})
	if err != nil {
		return res, fmt.Errorf("listing movies for library %q: %w", libraryID, err)
	}

	trackedPaths := make(map[string]bool, len(tracked))
	for _, f := range tracked {
		trackedPaths[f.Path] = true
	}
	// Every video file in the library that nothing tracks. An unreadable
	// root is not an error here: with nothing found, tracked files are
	// still checked one by one below.
	untracked, _ := scanDisk(lib.RootPath, trackedPaths)
	taken := make(map[string]bool)

	now := time.Now().UTC().Format(time.RFC3339)
	hasFile := make(map[string]bool)  // movie ID → a file of its is on disk
	lostFile := make(map[string]bool) // movie ID → a file of its is gone

	for _, f := range tracked {
		_, statErr := os.Stat(f.Path)
		switch {
		case statErr == nil:
			res.Present++
			hasFile[f.MovieID] = true
			_ = s.q.UpdateMovieFileIndexed(ctx, dbgen.UpdateMovieFileIndexedParams{IndexedAt: now, ID: f.ID})
			// Backfill edition from filename if not already set.
			if f.Edition == nil {
				if ed := edition.Parse(filepath.Base(f.Path)); ed != nil {
					name := ed.Name
					_ = s.q.UpdateMovieFileEdition(ctx, dbgen.UpdateMovieFileEditionParams{Edition: &name, ID: f.ID})
				}
			}
		case os.IsNotExist(statErr):
			moved := findMoved(f, untracked, taken)
			if moved == "" {
				lostFile[f.MovieID] = true
				continue
			}
			if err := s.q.UpdateMovieFilePath(ctx, dbgen.UpdateMovieFilePathParams{Path: moved, ID: f.ID}); err != nil {
				return res, fmt.Errorf("correcting path of %q: %w", f.Path, err)
			}
			_, _ = s.q.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{Path: &moved, UpdatedAt: now, ID: f.MovieID})
			_ = s.q.DeleteLibraryFileCandidate(ctx, dbgen.DeleteLibraryFileCandidateParams{LibraryID: libraryID, FilePath: moved})
			taken[moved] = true
			hasFile[f.MovieID] = true
			res.Relinked++
		}
		// Any other stat error (a share that's offline, say) proves
		// nothing either way, so the record is left alone.
	}

	if s.attach != nil {
		for _, m := range movies {
			if hasFile[m.ID] {
				continue
			}
			file := bestFileFor(m, untracked, taken)
			if file == nil {
				continue
			}
			if err := s.attach(ctx, m.ID, file.Path, file.SizeBytes, ParseQualityFromPath(file.Path)); err != nil {
				return res, fmt.Errorf("linking %q to %q: %w", file.Path, m.Title, err)
			}
			_ = s.q.DeleteLibraryFileCandidate(ctx, dbgen.DeleteLibraryFileCandidateParams{LibraryID: libraryID, FilePath: file.Path})
			taken[file.Path] = true
			hasFile[m.ID] = true
			res.Attached++
		}
	}

	for _, m := range movies {
		switch {
		case hasFile[m.ID] && m.Status != "downloaded":
			_, _ = s.q.UpdateMovieStatus(ctx, dbgen.UpdateMovieStatusParams{Status: "downloaded", UpdatedAt: now, ID: m.ID})
		case !hasFile[m.ID] && lostFile[m.ID]:
			res.Missing++
			if m.Status != "missing" {
				_, _ = s.q.UpdateMovieStatus(ctx, dbgen.UpdateMovieStatusParams{Status: "missing", UpdatedAt: now, ID: m.ID})
			}
		}
	}

	// Trigger background TMDB matching for any unmatched candidates.
	if s.meta != nil {
		go s.matchCandidates(context.Background(), libraryID)
	}
	return res, nil
}

// findMoved returns the path of the one untracked file with the same name
// and size as a tracked file that is no longer where it was recorded, or
// "" when there is none or more than one.
func findMoved(f dbgen.MovieFile, untracked []DiskFile, taken map[string]bool) string {
	name := filepath.Base(f.Path)
	found := ""
	for _, d := range untracked {
		if taken[d.Path] || d.SizeBytes != f.SizeBytes || filepath.Base(d.Path) != name {
			continue
		}
		if found != "" {
			return ""
		}
		found = d.Path
	}
	return found
}

// bestFileFor returns the largest untracked file whose name gives this
// movie's title and year, skipping samples and extras. Title and year
// must both match exactly: a wrong link is worse than a missed one.
func bestFileFor(m dbgen.Movie, untracked []DiskFile, taken map[string]bool) *DiskFile {
	if m.Year == 0 {
		return nil
	}
	title := normalizeCandidateTitle(m.Title)
	original := normalizeCandidateTitle(m.OriginalTitle)
	var best *DiskFile
	for i, d := range untracked {
		if taken[d.Path] || int64(d.ParsedYear) != m.Year || d.SizeBytes < minMovieBytes || isExtra(d.Path) {
			continue
		}
		parsed := normalizeCandidateTitle(d.ParsedTitle)
		if parsed == "" || (parsed != title && parsed != original) {
			continue
		}
		if best == nil || d.SizeBytes > best.SizeBytes {
			best = &untracked[i]
		}
	}
	return best
}

// isExtra reports whether a path looks like a sample, trailer or bonus
// feature rather than the movie itself.
func isExtra(path string) bool {
	lower := strings.ToLower(path)
	for _, dir := range []string{"/sample/", "/samples/", "/extras/", "/featurettes/", "/trailers/", "/behind the scenes/"} {
		if strings.Contains(lower, dir) {
			return true
		}
	}
	base := filepath.Base(lower)
	return strings.Contains(base, "sample") || strings.Contains(base, "trailer")
}

// moveFilePaths rewrites recorded file paths after a library's root
// folder changes, so files that moved with the library stay linked.
func (s *Service) moveFilePaths(ctx context.Context, libraryID, oldRoot, newRoot string) error {
	oldPrefix := strings.TrimRight(oldRoot, "/") + "/"
	newPrefix := strings.TrimRight(newRoot, "/") + "/"
	files, err := s.q.ListMovieFilesByLibrary(ctx, libraryID)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, f := range files {
		if !strings.HasPrefix(f.Path, oldPrefix) {
			continue
		}
		moved := newPrefix + strings.TrimPrefix(f.Path, oldPrefix)
		if err := s.q.UpdateMovieFilePath(ctx, dbgen.UpdateMovieFilePathParams{Path: moved, ID: f.ID}); err != nil {
			return err
		}
		_, _ = s.q.UpdateMoviePath(ctx, dbgen.UpdateMoviePathParams{Path: &moved, UpdatedAt: now, ID: f.MovieID})
	}
	return nil
}
