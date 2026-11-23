import { describe, it, expect, vi } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { releaseFixture } from "@/test/fixtures";
import { ConfirmProvider } from "@beacon-shared/ConfirmDialog";
import { ManualSearchModal } from "./ManualSearchModal";

const props = { movieId: "movie-1", movieTitle: "Fight Club", onClose: vi.fn() };

const Window = () => (
  <ConfirmProvider>
    <ManualSearchModal {...props} />
  </ConfirmProvider>
);


type Line = {
  releases: unknown[];
  indexers: { indexer_id: string; indexer_name: string; results: number; error?: string; pending?: boolean; searching?: boolean }[];
  done: boolean;
};

// serve answers the stream with these lines; gate (if given) holds back
// everything after the first line until it resolves.
function serve(lines: Line[], gate?: Promise<void>) {
  server.use(
    http.get("/api/v1/movies/movie-1/releases/stream", () => {
      const enc = new TextEncoder();
      const body = new ReadableStream({
        async start(c) {
          c.enqueue(enc.encode(JSON.stringify(lines[0]) + "\n"));
          if (gate) await gate;
          for (const l of lines.slice(1)) c.enqueue(enc.encode(JSON.stringify(l) + "\n"));
          c.close();
        },
      });
      return new HttpResponse(body, { headers: { "Content-Type": "application/x-ndjson" } });
    }),
  );
}

const fastRelease = { ...releaseFixture, guid: "fast", title: "Fight.Club.1999.1080p.BluRay.x264" };

describe("ManualSearchModal streaming", () => {
  it("shows results from fast indexers while slow ones are still searching", async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    serve(
      [
        {
          releases: [fastRelease],
          indexers: [
            { indexer_id: "fast", indexer_name: "Fast", results: 1 },
            { indexer_id: "slow", indexer_name: "SlowOne", results: 0, searching: true },
          ],
          done: false,
        },
        {
          releases: [fastRelease],
          indexers: [
            { indexer_id: "fast", indexer_name: "Fast", results: 1 },
            { indexer_id: "slow", indexer_name: "SlowOne", results: 0 },
          ],
          done: true,
        },
      ],
      gate,
    );
    renderWithProviders(<Window />);
    expect(await screen.findByText("Fight.Club.1999.1080p.BluRay.x264")).toBeInTheDocument();
    expect(screen.getByText("Still waiting on SlowOne")).toBeInTheDocument();
    release();
    await waitFor(() => expect(screen.queryByText("Still waiting on SlowOne")).not.toBeInTheDocument());
  });

  it("says when a Cloudflare-walled indexer is still solving", async () => {
    serve([
      {
        releases: [fastRelease],
        indexers: [
          { indexer_id: "fast", indexer_name: "Fast", results: 1 },
          { indexer_id: "cf", indexer_name: "1337x", results: 0, pending: true },
        ],
        done: true,
      },
    ]);
    renderWithProviders(<Window />);
    expect(await screen.findByText(/Still searching 1337x/)).toBeInTheDocument();
  });

  it("says no releases only when the search has finished, with why", async () => {
    serve([
      { releases: [], indexers: [{ indexer_id: "a", indexer_name: "Broken", results: 0, error: "HTTP 403" }], done: true },
    ]);
    renderWithProviders(<Window />);
    expect(await screen.findByText("No releases found.")).toBeInTheDocument();
    expect(screen.getByText("Broken: HTTP 403")).toBeInTheDocument();
  });
});

describe("ManualSearchModal theater recordings", () => {
  const cam = {
    ...releaseFixture,
    guid: "cam",
    title: "Fight.Club.1999.HDCAM.x264",
    quality: { ...releaseFixture.quality, source: "cam", name: "CAM-1080p" },
  };
  const indexers = [{ indexer_id: "a", indexer_name: "A", results: 2 }];

  it("hides CAM releases until asked, then shows them", async () => {
    serve([{ releases: [fastRelease, cam], indexers, done: true }]);
    renderWithProviders(<Window />);
    expect(await screen.findByText("Fight.Club.1999.1080p.BluRay.x264")).toBeInTheDocument();
    expect(screen.queryByText("Fight.Club.1999.HDCAM.x264")).not.toBeInTheDocument();
    expect(screen.getByText(/1 theater recording \(CAM\/TS\) hidden/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Show" }));
    expect(screen.getByText("Fight.Club.1999.HDCAM.x264")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Hide" }));
    expect(screen.queryByText("Fight.Club.1999.HDCAM.x264")).not.toBeInTheDocument();
  });

  it("says when only theater recordings were found", async () => {
    serve([{ releases: [cam], indexers, done: true }]);
    renderWithProviders(<Window />);
    expect(await screen.findByText("Only theater recordings (CAM/TS) found.")).toBeInTheDocument();
  });
});
