import { describe, it, expect } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { createElement } from "react";
import { server } from "@/test/handlers";
import { movieFixture } from "@/test/fixtures";
import MovieDetail from "./MovieDetail";

function renderAt(pathname: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity } },
  });
  return render(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: [pathname] },
        createElement(
          Routes,
          null,
          createElement(Route, {
            path: "/movies/:id",
            element: createElement(MovieDetail),
          })
        )
      )
    )
  );
}

describe("MovieDetail — You might also like", () => {
  it("links in-library recommendations to /movies/:movie_id and non-library ones to /discover/:tmdb_id", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1", () => HttpResponse.json(movieFixture)),
      http.get("/api/v1/movies/movie-1/files", () => HttpResponse.json([])),
      http.get("/api/v1/movies/movie-1/credits", () =>
        HttpResponse.json({
          cast: [],
          crew: [],
          recommendations: [
            {
              tmdb_id: 680,
              title: "Pulp Fiction",
              year: 1994,
              poster_path: "/pulp.jpg",
              in_library: false,
            },
            {
              tmdb_id: 155,
              title: "The Dark Knight",
              year: 2008,
              poster_path: "/tdk.jpg",
              in_library: true,
              movie_id: "movie-42",
            },
          ],
        })
      ),
      http.get("/api/v1/mediainfo/status", () =>
        HttpResponse.json({ available: false })
      ),
      http.get("/api/v1/editions", () => HttpResponse.json([]))
    );

    renderAt("/movies/movie-1");

    await waitFor(() =>
      expect(screen.getByText("You might also like")).toBeInTheDocument()
    );

    const section = screen.getByTestId("similar-movies");
    const links = within(section).getAllByRole("link");

    const pulp = links.find((a) => a.textContent?.includes("Pulp Fiction"));
    const tdk = links.find((a) => a.textContent?.includes("The Dark Knight"));

    expect(pulp).toBeDefined();
    expect(tdk).toBeDefined();
    expect(pulp).toHaveAttribute("href", "/discover/680");
    expect(tdk).toHaveAttribute("href", "/movies/movie-42");
  });
});
