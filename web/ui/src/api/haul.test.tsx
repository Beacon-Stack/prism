import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useMovieHaulHistory, useReimportFromHaul } from "./haul";
import type { HaulRecord } from "./haul";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

const haulRecordFixture: HaulRecord = {
  info_hash: "abc123",
  name: "Fight.Club.1999.1080p.BluRay.x264-GROUP",
  save_path: "/downloads/",
  category: "prism",
  added_at: "2026-04-01T00:00:00Z",
  completed_at: "2026-04-01T02:00:00Z",
  removed_at: "",
  requester: "prism",
  series_id: "",
  episode_id: "",
  tmdb_id: 550,
  season: 0,
  episode: 0,
};

describe("useMovieHaulHistory", () => {
  it("returns records for a movie", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/haul-history", () =>
        HttpResponse.json({ records: [haulRecordFixture] })
      )
    );

    const { result } = renderHook(() => useMovieHaulHistory("movie-1"), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].info_hash).toBe("abc123");
  });

  it("returns empty array when no Haul records exist", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/haul-history", () =>
        HttpResponse.json({ records: [] })
      )
    );

    const { result } = renderHook(() => useMovieHaulHistory("movie-1"), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(0);
  });

  it("does not fetch when movieId is empty", () => {
    const { result } = renderHook(() => useMovieHaulHistory(""), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useReimportFromHaul", () => {
  it("posts info_hash to import endpoint", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.post("/api/v1/import/from-haul", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json({ status: "imported" });
      })
    );

    const { result } = renderHook(() => useReimportFromHaul("movie-1"), {
      wrapper: createWrapper(),
    });
    result.current.mutate("abc123");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toEqual({ info_hash: "abc123" });
  });
});
