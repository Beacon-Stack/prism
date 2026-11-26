import { describe, it, expect } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import StatsPage from "./StatsPage";

// Shared fixtures

const collectionStats = {
  total_movies: 50,
  monitored: 45,
  with_file: 38,
  missing: 10,
  needs_upgrade: 3,
  edition_mismatches: 0,
  recently_added: 2,
};

// Two buckets: one 1080p Bluray and one 2160p WebDL.
// This gives the QualityCard data to render with total > 0 so the toggle appears.
const qualityBuckets = [
  { resolution: "1080p", source: "Bluray", codec: "x264", hdr: "none", count: 30 },
  { resolution: "2160p", source: "WebDL", codec: "x265", hdr: "HDR10", count: 10 },
];

const qualityTiers = [
  { resolution: "1080p", source: "Bluray", count: 30 },
  { resolution: "2160p", source: "WebDL", count: 10 },
];

const storageStats = {
  total_bytes: 2_000_000_000_000,
  file_count: 40,
  trend: [],
};

const grabStats = {
  total_grabs: 60,
  successful: 55,
  failed: 5,
  success_rate: 0.917,
  top_indexers: [],
};

const decadeStats = [
  { decade: "1990s", count: 5 },
  { decade: "2000s", count: 20 },
];

const growthStats = [
  { month: "2025-01", added: 10, cumulative: 10 },
];

const genreStats = [
  { genre: "Action", count: 20 },
  { genre: "Drama", count: 15 },
];

// Register all stats endpoints for a full page render.
function useFullStatsHandlers() {
  server.use(
    http.get("/api/v1/stats/collection", () => HttpResponse.json(collectionStats)),
    http.get("/api/v1/stats/quality", () => HttpResponse.json(qualityBuckets)),
    http.get("/api/v1/stats/quality/tiers", () => HttpResponse.json(qualityTiers)),
    http.get("/api/v1/stats/storage", () => HttpResponse.json(storageStats)),
    http.get("/api/v1/stats/grabs", () => HttpResponse.json(grabStats)),
    http.get("/api/v1/stats/decades", () => HttpResponse.json(decadeStats)),
    http.get("/api/v1/stats/growth", () => HttpResponse.json(growthStats)),
    http.get("/api/v1/stats/genres", () => HttpResponse.json(genreStats))
  );
}

function renderPage() {
  return renderWithProviders(createElement(StatsPage));
}

describe("StatsPage", () => {
  it("renders the Statistics heading", () => {
    useFullStatsHandlers();
    renderPage();
    expect(screen.getByText("Statistics")).toBeInTheDocument();
  });

  it("shows skeleton cards while data is loading", () => {
    useFullStatsHandlers();
    const { container } = renderPage();
    // At least one skeleton should be present before data arrives
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("renders collection stats after loading", async () => {
    useFullStatsHandlers();
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("50")).toBeInTheDocument();
    });
    expect(screen.getByText("Total Movies")).toBeInTheDocument();
    expect(screen.getByText("38")).toBeInTheDocument();
    expect(screen.getByText("Have File")).toBeInTheDocument();
  });

  // Quality tier toggle

  it("shows By Dimension and By Tier toggle buttons", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Dimension")).toBeInTheDocument();
    });
    expect(screen.getByText("By Tier")).toBeInTheDocument();
  });

  it("By Dimension view is active by default", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Dimension")).toBeInTheDocument();
    });

    // Dimension sub-charts are rendered in the default view
    expect(screen.getByText("Resolution")).toBeInTheDocument();
    expect(screen.getByText("Source")).toBeInTheDocument();
    expect(screen.getByText("Codec")).toBeInTheDocument();
    expect(screen.getByText("HDR")).toBeInTheDocument();

    // The tier-specific prompt should not be visible yet
    expect(
      screen.queryByText("Click a tier to filter the movie library.")
    ).not.toBeInTheDocument();
  });

  it("switching to By Tier shows tier chart heading and filter prompt", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Tier")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Tier"));

    await waitFor(() => {
      expect(
        screen.getByText("Click a tier to filter the movie library.")
      ).toBeInTheDocument();
    });

    // The combined resolution+source chart heading
    expect(screen.getByText("Resolution + Source")).toBeInTheDocument();
  });

  it("switching back to By Dimension restores dimension charts", async () => {
    useFullStatsHandlers();
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("By Tier")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Tier"));

    await waitFor(() => {
      expect(screen.getByText("Resolution + Source")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("By Dimension"));

    await waitFor(() => {
      expect(screen.getByText("Resolution")).toBeInTheDocument();
    });
    expect(
      screen.queryByText("Click a tier to filter the movie library.")
    ).not.toBeInTheDocument();
  });

  it("shows empty Quality Distribution when there are no files", async () => {
    server.use(
      http.get("/api/v1/stats/collection", () => HttpResponse.json(collectionStats)),
      http.get("/api/v1/stats/quality", () => HttpResponse.json([])),
      http.get("/api/v1/stats/quality/tiers", () => HttpResponse.json([])),
      http.get("/api/v1/stats/storage", () => HttpResponse.json(storageStats)),
      http.get("/api/v1/stats/grabs", () => HttpResponse.json(grabStats)),
      http.get("/api/v1/stats/decades", () => HttpResponse.json(decadeStats)),
      http.get("/api/v1/stats/growth", () => HttpResponse.json(growthStats)),
      http.get("/api/v1/stats/genres", () => HttpResponse.json(genreStats))
    );
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("No movie files yet.")).toBeInTheDocument();
    });

    // Toggle should not be rendered when there are no files
    expect(screen.queryByText("By Dimension")).not.toBeInTheDocument();
    expect(screen.queryByText("By Tier")).not.toBeInTheDocument();
  });
  // Where the space goes

  const GB = 1024 ** 3;
  const slice = (label: string, gb: number, movies: number, minutes: number) => ({
    label,
    bytes: gb * GB,
    movies,
    timed_bytes: gb * GB,
    runtime_minutes: minutes,
  });
  const breakdown = {
    total: slice("", 120, 3, 480),
    by_resolution: [slice("2160p", 100, 2, 300), slice("1080p", 20, 2, 180)],
    by_genre: [slice("Drama", 110, 2, 400), slice("Comedy", 10, 1, 0)],
    by_decade: [slice("1990s", 80, 2, 300), slice("2020s", 40, 1, 180)],
    largest: [
      { movie_id: "m-heat", title: "Heat", year: 1995, bytes: 70 * GB, resolution: "2160p", runtime_minutes: 170 },
      { movie_id: "m-dune", title: "Dune", year: 2021, bytes: 40 * GB, resolution: "2160p", runtime_minutes: 155 },
    ],
  };

  function serveBreakdown(data: object = breakdown) {
    useFullStatsHandlers();
    server.use(http.get("/api/v1/stats/storage/breakdown", () => HttpResponse.json(data)));
  }

  it("shows where the space goes, by resolution first", async () => {
    serveBreakdown();
    renderPage();

    const slices = await screen.findByTestId("storage-slices");
    // 100 GB over 5 hours of video.
    expect(slices).toHaveTextContent("2160p100 GB · 2 movies · 20 GB/h");
    expect(slices).toHaveTextContent("1080p20 GB · 2 movies");
    // 120 GB over 8 hours, and over 3 movies.
    expect(screen.getByText("15 GB/h")).toBeInTheDocument();
    expect(screen.getByText("Average per movie").previousSibling).toHaveTextContent("40 GB");

    const largest = screen.getByTestId("largest-movies");
    expect(largest).toHaveTextContent("Heat 1995 · 2160p70 GB");
    expect(screen.getByRole("link", { name: "Heat" })).toHaveAttribute("href", "/movies/m-heat");
  });

  it("leaves out a resolution that isn't known instead of printing 'unknown'", async () => {
    serveBreakdown({
      ...breakdown,
      by_resolution: [slice("unknown", 80, 1, 100), slice("sd", 1, 1, 90)],
      largest: [
        { movie_id: "m-thing", title: "The Thing", year: 1982, bytes: 80 * GB, resolution: "unknown", runtime_minutes: 109 },
        { movie_id: "m-clerks", title: "Clerks", year: 1994, bytes: GB, resolution: "sd", runtime_minutes: 92 },
      ],
    });
    renderPage();

    const largest = await screen.findByTestId("largest-movies");
    expect(largest).toHaveTextContent("The Thing 198280 GB");
    expect(largest).toHaveTextContent("Clerks 1994 · SD1 GB");
    expect(largest).not.toHaveTextContent("unknown");
    const slices = screen.getByTestId("storage-slices");
    expect(slices).toHaveTextContent("Not known80 GB");
    expect(slices).toHaveTextContent("SD1 GB");
  });

  it("regroups the space by genre and decade", async () => {
    serveBreakdown();
    renderPage();
    await screen.findByTestId("storage-slices");

    fireEvent.click(screen.getByRole("button", { name: "Genre" }));
    const slices = screen.getByTestId("storage-slices");
    expect(slices).toHaveTextContent("Drama110 GB · 2 movies");
    // No runtime known for the one comedy: no per-hour figure for it.
    expect(slices).toHaveTextContent(/Comedy10 GB · 1 movie$/);
    expect(screen.getByText(/counts toward each of its genres/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Decade" }));
    expect(screen.getByTestId("storage-slices")).toHaveTextContent("1990s80 GB · 2 movies");
    expect(screen.queryByText(/counts toward each of its genres/)).not.toBeInTheDocument();
  });

  it("says when the disk fills at the current rate", async () => {
    serveBreakdown({ ...breakdown, forecast: { bytes_per_day: GB, free_bytes: 420 * GB, days_until_full: 420, full_on: "2028-01-20T00:00:00Z" } });
    renderPage();
    expect(await screen.findByText("about 14 months")).toBeInTheDocument();
    expect(screen.getByText("Until the disk is full: 420 GB free, growing 7 GB a week")).toBeInTheDocument();
  });

  it("says so when the library isn't growing, and nothing when there is no forecast", async () => {
    serveBreakdown({ ...breakdown, forecast: { bytes_per_day: 0, free_bytes: 420 * GB } });
    const first = renderPage();
    expect(await screen.findByText("Free, and the library isn't growing")).toBeInTheDocument();
    expect(screen.getByText("420 GB")).toBeInTheDocument();
    first.unmount();

    serveBreakdown();
    renderPage();
    await screen.findByTestId("storage-slices");
    expect(screen.queryByText(/Until the disk is full/)).not.toBeInTheDocument();
    expect(screen.queryByText(/isn't growing/)).not.toBeInTheDocument();
  });

  it("has its own empty state for a library with no files", async () => {
    useFullStatsHandlers();
    renderPage();
    expect(await screen.findByText("Nothing on disk yet.")).toBeInTheDocument();
  });

  // Watching

  const watching = {
    owned: 40,
    watched: 10,
    plays: 23,
    minutes_watched: 3000,
    by_month: [
      { month: "2026-10", plays: 4 },
      { month: "2026-11", plays: 7 },
    ],
    most_watched: [
      { movie_id: "m-heat", title: "Heat", year: 1995, plays: 3, last_watched_at: "2026-11-02T20:00:00Z" },
      { movie_id: "m-dune", title: "Dune", year: 2021, plays: 1, last_watched_at: "2026-09-12T20:00:00Z" },
    ],
    longest_unwatched: [{ movie_id: "m-alien", title: "Alien", year: 1979, added_at: "2020-01-15T12:00:00Z" }],
  };

  function serveWatching(data: object = watching) {
    useFullStatsHandlers();
    server.use(http.get("/api/v1/stats/watching", () => HttpResponse.json(data)));
  }

  it("shows what has been watched and what is still waiting", async () => {
    serveWatching();
    renderPage();

    expect(await screen.findByText("25%")).toBeInTheDocument();
    expect(screen.getByText("Watched: 10 of 40 movies")).toBeInTheDocument();
    expect(screen.getByText("23")).toBeInTheDocument();
    expect(screen.getByText("2d 2h")).toBeInTheDocument();

    const most = screen.getByTestId("most-watched");
    expect(most).toHaveTextContent(/Heat 19953 plays · last Nov 2026/);
    expect(most).toHaveTextContent(/Dune 20211 play · last Sep 2026/);

    const waiting = screen.getByTestId("longest-unwatched");
    expect(waiting).toHaveTextContent(/Alien 1979added Jan 2020/);
    expect(waiting.querySelector("a")).toHaveAttribute("href", "/movies/m-alien");
  });

  it("says so when everything has been watched", async () => {
    serveWatching({ ...watching, longest_unwatched: [] });
    renderPage();
    expect(await screen.findByText("Nothing. Every movie here has been watched.")).toBeInTheDocument();
  });

  it("points to Media Servers when there is no watch history", async () => {
    useFullStatsHandlers();
    renderPage();
    expect(await screen.findByText(/No watch history yet/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Settings → Media Servers" })).toHaveAttribute("href", "/settings/media-servers");
  });
});
