// ActivityPage.test.tsx — exercises the four rails (Currently
// downloading, Recently imported, Needs attention, Background tasks).
// The page no longer has filter pills; the legacy flat-timeline tests
// were deleted with the component.

import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import ActivityPage from "./ActivityPage";

function renderPage() {
  return renderWithProviders(createElement(ActivityPage));
}

const movie = {
  id: "movie-1",
  tmdb_id: 100,
  title: "Inception",
  year: 2010,
  status: "monitored",
  monitored: true,
  library_id: "lib-1",
  size_on_disk: 0,
  added_at: "2024-01-01T00:00:00Z",
};

describe("ActivityPage", () => {
  it("renders the page heading", () => {
    renderPage();
    expect(screen.getByRole("heading", { name: /Activity/i })).toBeInTheDocument();
  });

  // The Downloading rail is the first user-facing data rail. If it
  // doesn't show items the queue API returns, the page is back to
  // being useless.
  it("shows currently-downloading items in the Downloading rail", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [movie], total: 1, page: 1, per_page: 500 })
      ),
      http.get("/api/v1/queue", () =>
        HttpResponse.json([
          {
            id: "q1",
            movie_id: "movie-1",
            release_title: "Inception.2010.UHD.BluRay.2160p",
            protocol: "torrent",
            size: 50_000_000_000,
            downloaded_bytes: 25_000_000_000,
            status: "downloading",
            grabbed_at: new Date().toISOString(),
          },
        ])
      )
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText("Inception.2010.UHD.BluRay.2160p")
      ).toBeInTheDocument()
    );
  });

  // The Recently Imported rail reads activity_log import_complete events
  // (NOT grab_history) so completion-time bucketing works correctly.
  // grab_history.grabbed_at is request time, which can be days before
  // a large download finishes — using it for the 48h cutoff hides
  // imports that just landed.
  it("shows recently-completed import_complete events in the Recently Imported rail", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [movie], total: 1, page: 1, per_page: 500 })
      ),
      http.get("/api/v1/activity", ({ request }) => {
        const url = new URL(request.url);
        if (url.searchParams.get("category") !== "import_succeeded") {
          return HttpResponse.json({ activities: [], total: 0 });
        }
        return HttpResponse.json({
          activities: [
            {
              id: "a-import",
              type: "import_complete",
              category: "import_succeeded",
              movie_id: "movie-1",
              title: "Imported Inception — 1080p BluRay",
              detail: { quality: "1080p BluRay" },
              created_at: new Date(Date.now() - 3600_000).toISOString(),
            },
          ],
          total: 1,
        });
      })
    );

    renderPage();
    // The rail renders the resolved movie title (from useMovies idx),
    // not the raw event title — so we look for "Inception".
    await waitFor(() =>
      expect(screen.getByText("Inception")).toBeInTheDocument()
    );
  });

  // Pin the time-window contract: an import_complete event older than
  // 48h must NOT render. Without this the rail would slowly drift into
  // an "all imports ever" view.
  it("filters out import_complete events older than 48h", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [movie], total: 1, page: 1, per_page: 500 })
      ),
      http.get("/api/v1/activity", ({ request }) => {
        const url = new URL(request.url);
        if (url.searchParams.get("category") !== "import_succeeded") {
          return HttpResponse.json({ activities: [], total: 0 });
        }
        return HttpResponse.json({
          activities: [
            {
              id: "a-old",
              type: "import_complete",
              category: "import_succeeded",
              movie_id: "movie-1",
              title: "Imported Inception",
              created_at: new Date(Date.now() - 72 * 3600_000).toISOString(), // 3 days ago
            },
          ],
          total: 1,
        });
      })
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/No imports in the last 48 hours/i)).toBeInTheDocument()
    );
  });

  // The Needs Attention rail is the load-bearing piece for triage.
  // If the API returns failures, they MUST render — otherwise the rail
  // claims "nothing needs attention" while failures are piling up.
  it("renders failures from the needs-attention API in the Needs Attention rail", async () => {
    server.use(
      http.get("/api/v1/activity/needs-attention", () =>
        HttpResponse.json({
          items: [
            {
              kind: "grab_failed",
              grab_id: "g1",
              movie_id: "movie-1",
              release_title: "Bad.Release.1080p",
              detail: "Download failed",
              created_at: new Date().toISOString(),
            },
            {
              kind: "import_failed",
              movie_id: "movie-1",
              release_title: "Other.Release.1080p",
              detail: "permission denied",
              created_at: new Date(Date.now() - 1000).toISOString(),
            },
          ],
          counts: { grab_failed: 1, import_failed: 1 },
        })
      )
    );

    renderPage();
    await waitFor(() => {
      expect(screen.getByText("Bad.Release.1080p")).toBeInTheDocument();
      expect(screen.getByText("Other.Release.1080p")).toBeInTheDocument();
    });
    // Counts summary should be visible too.
    expect(screen.getByText(/1 failed grab/i)).toBeInTheDocument();
    expect(screen.getByText(/1 failed import/i)).toBeInTheDocument();
  });

  // Empty state for needs-attention shouldn't crash and must say
  // something reassuring rather than rendering nothing.
  it("renders the clean-window empty state when no failures", async () => {
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText(/Nothing needs attention/i)
      ).toBeInTheDocument()
    );
  });

  // The Background Tasks rail is a known placeholder. Default-collapsed
  // so it doesn't take vertical space — assert the header is rendered.
  it("renders the background-tasks rail header", () => {
    renderPage();
    expect(
      screen.getByText(/Active background tasks/i)
    ).toBeInTheDocument();
  });
});
