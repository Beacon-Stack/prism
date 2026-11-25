package library_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/core/movie"
	"github.com/beacon-stack/prism/internal/core/quality"
	dbgen "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/testutil"
)

const movieSize = 150 * 1024 * 1024 // above the scan's minimum for a movie

// scanFixture is a library on a temp folder with real services behind it.
type scanFixture struct {
	t    *testing.T
	q    *dbgen.Queries
	libs *library.Service
	lib  library.Library
	root string
}

func newScanFixture(t *testing.T) *scanFixture {
	t.Helper()
	q := testutil.NewTestDB(t)
	libs := library.NewService(q, nil, nil)
	movies := movie.NewService(q, nil, nil, slog.New(slog.DiscardHandler))
	libs.SetFileAttacher(movies.AttachFile)

	root := t.TempDir()
	req := sampleCreateRequest(createQualityProfile(t, quality.NewService(q, nil)))
	req.RootPath = root
	lib, err := libs.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create library: %v", err)
	}
	return &scanFixture{t: t, q: q, libs: libs, lib: lib, root: root}
}

// file creates a file of the given size (sparse, so it costs no disk) at
// a path relative to the library root and returns its full path.
func (f *scanFixture) file(rel string, size int64) string {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Truncate(path, size); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func (f *scanFixture) movie(id, title string, year int64) {
	f.t.Helper()
	_, err := f.q.CreateMovie(context.Background(), dbgen.CreateMovieParams{
		ID: id, TmdbID: int64(len(id)) + year, Title: title, OriginalTitle: title, Year: year,
		GenresJson: "[]", Status: "released", Monitored: true,
		LibraryID: f.lib.ID, QualityProfileID: f.lib.DefaultQualityProfileID,
		AddedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
		MinimumAvailability: "released", ReleaseDate: "2025-01-01",
	})
	if err != nil {
		f.t.Fatalf("CreateMovie: %v", err)
	}
}

func (f *scanFixture) tracked(movieID, path string, size int64) {
	f.t.Helper()
	_, err := f.q.CreateMovieFile(context.Background(), dbgen.CreateMovieFileParams{
		ID: "file-" + movieID, MovieID: movieID, Path: path, SizeBytes: size, QualityJson: "{}",
		ImportedAt: "2026-01-01T00:00:00Z", IndexedAt: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		f.t.Fatalf("CreateMovieFile: %v", err)
	}
}

func (f *scanFixture) scan() library.ScanResult {
	f.t.Helper()
	res, err := f.libs.Scan(context.Background(), f.lib.ID)
	if err != nil {
		f.t.Fatalf("Scan: %v", err)
	}
	return res
}

func (f *scanFixture) status(movieID string) string {
	f.t.Helper()
	m, err := f.q.GetMovie(context.Background(), movieID)
	if err != nil {
		f.t.Fatal(err)
	}
	return m.Status
}

func (f *scanFixture) paths(movieID string) []string {
	f.t.Helper()
	files, err := f.q.ListMovieFiles(context.Background(), movieID)
	if err != nil {
		f.t.Fatal(err)
	}
	out := make([]string, len(files))
	for i, mf := range files {
		out[i] = mf.Path
	}
	return out
}

// A movie in the list whose file was put in the library folder by hand,
// under the release's own name, is picked up.
func TestScan_LinksAFileToAMovieAlreadyInTheLibrary(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "A House of Dynamite", 2025)
	want := f.file("A House Of Dynamite (2025) [1080p] [WEBRip] [5.1] [YTS.MX]/A.House.Of.Dynamite.2025.1080p.WEBRip.x264.AAC5.1-[YTS.MX].mp4", movieSize)

	res := f.scan()

	if res.Attached != 1 {
		t.Errorf("Attached = %d, want 1", res.Attached)
	}
	if got := f.paths("m1"); len(got) != 1 || got[0] != want {
		t.Errorf("files = %v, want [%s]", got, want)
	}
	if got := f.status("m1"); got != "downloaded" {
		t.Errorf("status = %q, want downloaded", got)
	}

	// A second scan changes nothing.
	if res := f.scan(); res.Attached != 0 || res.Present != 1 {
		t.Errorf("second scan = %+v, want 1 present and nothing attached", res)
	}
	if got := f.paths("m1"); len(got) != 1 {
		t.Errorf("files after second scan = %v, want one", got)
	}
}

func TestScan_PicksTheMovieNotItsSampleAndPrefersTheLargerCopy(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "The Thing", 1982)
	f.file("The.Thing.1982.720p/The.Thing.1982.BluRay.720p.x264.DTS-WiKi.Sample.mkv", movieSize)
	f.file("The.Thing.1982.720p/The.Thing.1982.BluRay.720p.x264.DTS-WiKi.mkv", movieSize)
	want := f.file("The Thing (1982)/The Thing (1982).mkv", 2*movieSize)

	f.scan()

	if got := f.paths("m1"); len(got) != 1 || got[0] != want {
		t.Errorf("files = %v, want [%s]", got, want)
	}
}

// Linking the wrong file is worse than linking none.
func TestScan_LeavesFilesItCannotPlaceConfidently(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "Dune", 2021)
	f.movie("m2", "The Shawshank Redemption", 1994)
	f.file("Dune (1984)/Dune (1984).mkv", movieSize)                                   // other year
	f.file("Dune Part Two (2024)/Dune Part Two (2024).mkv", movieSize)                 // other title
	f.file("Shawshank/the.shawshank.redemption.1994.720p.sample-sinner.mkv", 4<<20)    // only a sample
	f.file("Arrival 2016 1080p BluRay x264 DTS-JYK/Arrival.2016.1080p.mkv", movieSize) // not in the list

	res := f.scan()

	if res.Attached != 0 {
		t.Errorf("Attached = %d, want 0", res.Attached)
	}
	for _, id := range []string{"m1", "m2"} {
		if got := f.paths(id); len(got) != 0 {
			t.Errorf("%s files = %v, want none", id, got)
		}
		if got := f.status(id); got != "released" {
			t.Errorf("%s status = %q, want released", id, got)
		}
	}
}

// The library folder was mounted somewhere else: the record still names
// the old path, and the same file is at the new one.
func TestScan_CorrectsThePathOfAFileThatMoved(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "Ballerina", 2025)
	f.tracked("m1", "/movies/Ballerina (2025)/Ballerina (2025).mkv", movieSize)
	want := f.file("Ballerina (2025)/Ballerina (2025).mkv", movieSize)

	res := f.scan()

	if res.Relinked != 1 || res.Attached != 0 || res.Missing != 0 {
		t.Errorf("result = %+v, want 1 relinked only", res)
	}
	if got := f.paths("m1"); len(got) != 1 || got[0] != want {
		t.Errorf("files = %v, want [%s]", got, want)
	}
	if got := f.status("m1"); got != "downloaded" {
		t.Errorf("status = %q, want downloaded", got)
	}
}

func TestScan_MarksAMovieMissingWhenItsFileIsGone(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "Dune", 2021)
	f.tracked("m1", "/movies/Dune (2021)/Dune (2021).mkv", movieSize)

	res := f.scan()

	if res.Missing != 1 {
		t.Errorf("Missing = %d, want 1", res.Missing)
	}
	if got := f.status("m1"); got != "missing" {
		t.Errorf("status = %q, want missing", got)
	}
}

func TestUpdate_ChangingTheRootFolderMovesFilePaths(t *testing.T) {
	f := newScanFixture(t)
	f.movie("m1", "Anna", 2019)
	f.tracked("m1", filepath.Join(f.root, "Anna (2019)/Anna (2019).mkv"), movieSize)

	req := sampleCreateRequest(f.lib.DefaultQualityProfileID)
	req.RootPath = "/data/media/movies"
	if _, err := f.libs.Update(context.Background(), f.lib.ID, req); err != nil {
		t.Fatalf("Update: %v", err)
	}

	want := "/data/media/movies/Anna (2019)/Anna (2019).mkv"
	if got := f.paths("m1"); len(got) != 1 || got[0] != want {
		t.Errorf("files = %v, want [%s]", got, want)
	}
}
