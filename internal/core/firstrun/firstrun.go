// Package firstrun gives a fresh install working defaults, the way Radarr
// ships with quality profiles and asks only for a root folder.
//
// On startup, and only while the database has no quality profiles and no
// libraries, it creates:
//
//   - default quality profiles (Any, SD, HD-720p, HD-1080p,
//     HD - 720p/1080p, Ultra-HD), and
//   - a "Movies" library at the first usable path from [Options.LibraryPaths]
//     (by default /data/media/movies in the single-/data-mount layout,
//     falling back to /movies for the older three-mount layout).
//
// Anything the user has already set up is left alone: if a profile or a
// library exists, that part is skipped.
//
// Prism matches a release against a profile by resolution, source AND
// codec, so each default quality is listed once per codec.
package firstrun

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/beacon-stack/prism/internal/core/library"
	"github.com/beacon-stack/prism/internal/core/quality"
	"github.com/beacon-stack/prism/pkg/plugin"
)

// DefaultProfileName is the profile the default library uses.
const DefaultProfileName = "HD-1080p"

// DefaultLibraryPaths is where the default movie library goes, in order
// of preference. A path is used when its parent folder exists (the folder
// itself is created if needed): /data/media/movies for the single /data
// mount, /movies for the older layout.
var DefaultLibraryPaths = []string{"/data/media/movies", "/movies"}

// Options configures Run.
type Options struct {
	// LibraryPaths overrides DefaultLibraryPaths (tests, custom layouts).
	LibraryPaths []string
}

type profileStore interface {
	List(ctx context.Context) ([]quality.Profile, error)
	Create(ctx context.Context, req quality.CreateRequest) (quality.Profile, error)
}

type libraryStore interface {
	List(ctx context.Context) ([]library.Library, error)
	Create(ctx context.Context, req library.CreateRequest) (library.Library, error)
}

// Run applies the first-run defaults. It is safe to call on every start.
func Run(ctx context.Context, profiles profileStore, libraries libraryStore, opts Options, logger *slog.Logger) error {
	existingProfiles, err := profiles.List(ctx)
	if err != nil {
		return fmt.Errorf("listing quality profiles: %w", err)
	}
	existingLibraries, err := libraries.List(ctx)
	if err != nil {
		return fmt.Errorf("listing libraries: %w", err)
	}

	var defaultProfileID string
	if len(existingProfiles) == 0 {
		for _, req := range DefaultProfiles() {
			p, err := profiles.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("creating quality profile %q: %w", req.Name, err)
			}
			if p.Name == DefaultProfileName {
				defaultProfileID = p.ID
			}
		}
		logger.Info("first run: created default quality profiles", "count", len(DefaultProfiles()))
	} else {
		defaultProfileID = existingProfiles[0].ID
		for _, p := range existingProfiles {
			if p.Name == DefaultProfileName {
				defaultProfileID = p.ID
			}
		}
	}

	if len(existingLibraries) > 0 {
		return nil
	}
	paths := opts.LibraryPaths
	if paths == nil {
		paths = DefaultLibraryPaths
	}
	root := pickLibraryPath(paths)
	if root == "" {
		logger.Info("first run: no media folder found, add a library in Settings → Libraries",
			"looked_for", paths)
		return nil
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		logger.Warn("first run: could not create the default library folder; add a library in Settings → Libraries",
			"path", root, "uid", os.Getuid(), "gid", os.Getgid(), "error", err)
		return nil
	}
	if _, err := libraries.Create(ctx, library.CreateRequest{
		Name:                    "Movies",
		RootPath:                root,
		DefaultQualityProfileID: defaultProfileID,
	}); err != nil {
		return fmt.Errorf("creating default library: %w", err)
	}
	logger.Info("first run: created default movie library", "path", root)
	return nil
}

// pickLibraryPath returns the first path whose parent directory exists.
func pickLibraryPath(paths []string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(filepath.Dir(filepath.Clean(p))); err == nil && fi.IsDir() {
			return p
		}
	}
	return ""
}

// codecs are the variants each default quality is listed under, best
// first, because Prism's matching includes the codec.
var codecs = []plugin.Codec{plugin.CodecAV1, plugin.CodecX265, plugin.CodecX264, plugin.CodecUnknown}

type tier struct {
	res plugin.Resolution
	src plugin.Source
}

// expand lists every tier once per codec, keeping the tier order.
func expand(tiers ...tier) []plugin.Quality {
	var out []plugin.Quality
	for _, t := range tiers {
		for _, c := range codecs {
			out = append(out, plugin.Quality{Resolution: t.res, Source: t.src, Codec: c, HDR: plugin.HDRNone})
		}
	}
	return out
}

// lowest returns t with the lowest-scoring codec, so a cutoff is met by
// any codec of that tier.
func lowest(t tier) plugin.Quality {
	return plugin.Quality{Resolution: t.res, Source: t.src, Codec: plugin.CodecUnknown, HDR: plugin.HDRNone}
}

// DefaultProfiles returns the profiles a fresh install starts with,
// modelled on Radarr's. Qualities are ordered best first.
func DefaultProfiles() []quality.CreateRequest {
	var (
		dvd        = tier{plugin.ResolutionSD, plugin.SourceDVD}
		sdWEBDL    = tier{plugin.ResolutionSD, plugin.SourceWEBDL}
		sdWEBRip   = tier{plugin.ResolutionSD, plugin.SourceWEBRip}
		webdl720   = tier{plugin.Resolution720p, plugin.SourceWEBDL}
		webrip720  = tier{plugin.Resolution720p, plugin.SourceWEBRip}
		bluray720  = tier{plugin.Resolution720p, plugin.SourceBluRay}
		webdl1080  = tier{plugin.Resolution1080p, plugin.SourceWEBDL}
		webrip1080 = tier{plugin.Resolution1080p, plugin.SourceWEBRip}
		bluray1080 = tier{plugin.Resolution1080p, plugin.SourceBluRay}
		remux1080  = tier{plugin.Resolution1080p, plugin.SourceRemux}
		webdl2160  = tier{plugin.Resolution2160p, plugin.SourceWEBDL}
		webrip2160 = tier{plugin.Resolution2160p, plugin.SourceWEBRip}
		bluray2160 = tier{plugin.Resolution2160p, plugin.SourceBluRay}
		remux2160  = tier{plugin.Resolution2160p, plugin.SourceRemux}
	)
	sd := []tier{sdWEBDL, sdWEBRip, dvd}
	hd720 := []tier{bluray720, webdl720, webrip720}
	hd1080 := []tier{bluray1080, webdl1080, webrip1080}
	uhd := []tier{bluray2160, webdl2160, webrip2160}

	cat := func(groups ...[]tier) []tier {
		var out []tier
		for _, g := range groups {
			out = append(out, g...)
		}
		return out
	}
	until := func(t tier) *plugin.Quality { q := lowest(t); return &q }

	return []quality.CreateRequest{
		{Name: "Any", Cutoff: lowest(dvd), Qualities: expand(cat(uhd, hd1080, hd720, sd)...)},
		{Name: "SD", Cutoff: lowest(dvd), Qualities: expand(sd...)},
		{Name: "HD-720p", Cutoff: lowest(bluray720), Qualities: expand(hd720...)},
		{Name: DefaultProfileName, Cutoff: lowest(bluray1080), Qualities: expand(hd1080...), UpgradeAllowed: true, UpgradeUntil: until(bluray1080)},
		{Name: "HD - 720p/1080p", Cutoff: lowest(bluray720), Qualities: expand(cat(hd1080, hd720)...), UpgradeAllowed: true, UpgradeUntil: until(bluray1080)},
		{Name: "Remux-1080p", Cutoff: lowest(remux1080), Qualities: expand(remux1080, bluray1080), UpgradeAllowed: true, UpgradeUntil: until(remux1080)},
		{Name: "Ultra-HD", Cutoff: lowest(webdl2160), Qualities: expand(cat([]tier{remux2160}, uhd)...), UpgradeAllowed: true, UpgradeUntil: until(remux2160)},
	}
}
