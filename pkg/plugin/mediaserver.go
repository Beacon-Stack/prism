package plugin

import "context"

// MediaServer is implemented by media server plugins (Plex, Emby, Jellyfin).
// After a movie file is imported, the dispatcher calls RefreshLibrary so the
// media server picks up the new file without waiting for a scheduled scan.
type MediaServer interface {
	// Name returns the human-readable name of the media server plugin.
	Name() string

	// RefreshLibrary tells the media server to re-scan the library section
	// that contains moviePath. Implementations may fall back to a full
	// library refresh if path-scoped scanning is unavailable.
	RefreshLibrary(ctx context.Context, moviePath string) error

	// Test verifies that the media server is reachable and the credentials
	// are valid.
	Test(ctx context.Context) error
}

// MediaItem identifies one movie held by a media server.
type MediaItem struct {
	TMDBID int // 0 when the server has no TMDB ID for it
	Title  string
	Year   int
}

// MovieLister is implemented by media servers that can say which movies
// they hold, so the app can show what is ready to watch.
type MovieLister interface {
	ListMovieItems(ctx context.Context) ([]MediaItem, error)
}
