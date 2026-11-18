package firstrun

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/beacon-stack/prism/internal/config"
	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/core/quality"
	dbpkg "github.com/beacon-stack/prism/internal/db"
	db "github.com/beacon-stack/prism/internal/db/generated"
	"github.com/beacon-stack/prism/internal/events"
	"github.com/beacon-stack/prism/pkg/plugin"
)

type env struct {
	profiles  *quality.Service
	libraries *library.Service
	dir       string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	d, err := dbpkg.Open(config.DatabaseConfig{Path: filepath.Join(dir, "pilot.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := dbpkg.Migrate(d.SQL); err != nil {
		t.Fatal(err)
	}
	q := db.New(d.SQL)
	bus := events.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return &env{profiles: quality.NewService(q, bus), libraries: library.NewService(q, bus, nil), dir: dir}
}

func (e *env) run(t *testing.T, paths ...string) {
	t.Helper()
	if err := Run(context.Background(), e.profiles, e.libraries, Options{LibraryPaths: paths},
		slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
}

func TestRun_FreshInstallGetsProfilesAndLibrary(t *testing.T) {
	e := newEnv(t)
	data := filepath.Join(e.dir, "data")
	if err := os.MkdirAll(filepath.Join(data, "media"), 0o755); err != nil {
		t.Fatal(err)
	}
	tvRoot := filepath.Join(data, "media", "movies")
	e.run(t, tvRoot, filepath.Join(e.dir, "movies"))

	profiles, _ := e.profiles.List(context.Background())
	if len(profiles) != len(DefaultProfiles()) {
		t.Fatalf("profiles = %d, want %d", len(profiles), len(DefaultProfiles()))
	}
	libs, _ := e.libraries.List(context.Background())
	if len(libs) != 1 || libs[0].RootPath != tvRoot {
		t.Fatalf("libraries = %+v, want one at %s", libs, tvRoot)
	}
	if fi, err := os.Stat(tvRoot); err != nil || !fi.IsDir() {
		t.Fatal("library folder was not created")
	}
	var defaultID string
	for _, p := range profiles {
		if p.Name == DefaultProfileName {
			defaultID = p.ID
		}
	}
	if libs[0].DefaultQualityProfileID != defaultID {
		t.Fatalf("library profile = %s, want %s (%s)", libs[0].DefaultQualityProfileID, defaultID, DefaultProfileName)
	}
}

func TestRun_IsIdempotent(t *testing.T) {
	e := newEnv(t)
	tvRoot := filepath.Join(e.dir, "tv")
	e.run(t, tvRoot)
	e.run(t, tvRoot)
	profiles, _ := e.profiles.List(context.Background())
	libs, _ := e.libraries.List(context.Background())
	if len(profiles) != len(DefaultProfiles()) || len(libs) != 1 {
		t.Fatalf("second run duplicated defaults: %d profiles, %d libraries", len(profiles), len(libs))
	}
}

func TestRun_LeavesExistingSetupAlone(t *testing.T) {
	e := newEnv(t)
	mine, err := e.profiles.Create(context.Background(), quality.CreateRequest{
		Name:   "Mine",
		Cutoff: plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceWEBDL},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Existing profile, no library: no new profiles, library uses mine.
	tvRoot := filepath.Join(e.dir, "tv")
	e.run(t, tvRoot)
	profiles, _ := e.profiles.List(context.Background())
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want only the user's", len(profiles))
	}
	libs, _ := e.libraries.List(context.Background())
	if len(libs) != 1 || libs[0].DefaultQualityProfileID != mine.ID {
		t.Fatalf("library = %+v, want one using the existing profile", libs)
	}

	// Existing library: nothing more is created, even at a new path.
	other := filepath.Join(e.dir, "other")
	e.run(t, other)
	libs, _ = e.libraries.List(context.Background())
	if len(libs) != 1 {
		t.Fatalf("libraries = %d, want 1", len(libs))
	}
}

func TestRun_NoMediaFolderSkipsLibrary(t *testing.T) {
	e := newEnv(t)
	e.run(t, filepath.Join(e.dir, "missing", "media", "movies"))
	libs, _ := e.libraries.List(context.Background())
	if len(libs) != 0 {
		t.Fatalf("created a library with no media folder: %+v", libs)
	}
	profiles, _ := e.profiles.List(context.Background())
	if len(profiles) == 0 {
		t.Fatal("profiles should still be created")
	}
}

// Every codec of every listed tier must be accepted, and the cutoff must
// be reachable: a release at the cutoff tier satisfies the profile.
func TestDefaultProfiles_AcceptEveryCodec(t *testing.T) {
	for _, req := range DefaultProfiles() {
		p := quality.Profile{Cutoff: req.Cutoff, Qualities: req.Qualities, UpgradeAllowed: req.UpgradeAllowed, UpgradeUntil: req.UpgradeUntil}
		for _, allowed := range req.Qualities {
			for _, c := range []plugin.Codec{plugin.CodecX264, plugin.CodecX265, plugin.CodecAV1, plugin.CodecUnknown} {
				rel := plugin.Quality{Resolution: allowed.Resolution, Source: allowed.Source, Codec: c}
				if !p.WantRelease(rel, nil) {
					t.Errorf("%s: rejects %s %s %s with no file on disk", req.Name, rel.Resolution, rel.Source, c)
				}
			}
		}
		cam := plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceCAM, Codec: plugin.CodecX264}
		if p.WantRelease(cam, nil) {
			t.Errorf("%s: accepts a CAM release", req.Name)
		}
	}
}
