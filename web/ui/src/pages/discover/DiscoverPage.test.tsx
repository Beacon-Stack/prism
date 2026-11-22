import { beforeEach, describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import DiscoverPage from "./DiscoverPage";

function renderPage() {
  return renderWithProviders(createElement(DiscoverPage));
}

const fixtures = {
  results: [
    {
      tmdb_id: 550,
      title: "Fight Club",
      year: 1999,
      overview: "An insomniac office worker...",
      poster_path: "/poster1.jpg",
      rating: 8.4,
      in_library: false,
      excluded: false,
    },
    {
      tmdb_id: 155,
      title: "The Dark Knight",
      year: 2008,
      overview: "Batman raises the stakes...",
      poster_path: "/poster2.jpg",
      rating: 8.5,
      in_library: true,
      excluded: false,
      library_movie_id: "movie-1",
    },
    {
      tmdb_id: 999,
      title: "Excluded Movie",
      year: 2020,
      overview: "...",
      poster_path: "/poster3.jpg",
      rating: 5.0,
      in_library: false,
      excluded: true,
    },
  ],
  page: 1,
  total_pages: 5,
};

// The page keeps its state in the URL; start each test from a clean one.
beforeEach(() => window.history.replaceState({}, "", "/discover"));

const browse = "/api/v1/discover/browse";

function serveBrowse(data: object = fixtures) {
  const seen: URLSearchParams[] = [];
  server.use(
    http.get(browse, ({ request }) => {
      seen.push(new URL(request.url).searchParams);
      return HttpResponse.json(data);
    })
  );
  return seen;
}

describe("DiscoverPage", () => {
  it("renders heading, search, the five lists and the filters", async () => {
    renderPage();
    expect(screen.getByText("Discover")).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: "Search movies" })).toBeInTheDocument();
    for (const name of ["Trending", "Popular", "Top rated", "New on streaming", "Coming soon"]) {
      expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }
    expect(screen.getByRole("combobox", { name: "Genre" })).toBeInTheDocument();
    expect(await screen.findByRole("option", { name: "Netflix" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Minimum rating" })).toBeInTheDocument();
  });

  it("shows loading skeletons while fetching", () => {
    server.use(http.get(browse, () => new Promise(() => {})));
    const { container } = renderPage();
    expect(container.querySelectorAll(".skeleton").length).toBeGreaterThan(0);
  });

  it("renders movie grid from API data, trending first", async () => {
    const seen = serveBrowse();
    renderPage();
    expect(await screen.findByText("Fight Club")).toBeInTheDocument();
    expect(screen.getByText("The Dark Knight")).toBeInTheDocument();
    expect(screen.getByText("Excluded Movie")).toBeInTheDocument();
    expect(seen[0].get("category")).toBe("trending");
    expect(seen[0].get("region")).toMatch(/^[A-Z]{2}$/);
  });

  it("shows In Library badge for movies in library", async () => {
    serveBrowse();
    renderPage();
    expect(await screen.findByTestId("in-library-155")).toBeInTheDocument();
  });

  it("shows + Add button for movies not in library", async () => {
    serveBrowse();
    renderPage();
    expect(await screen.findByTestId("add-550")).toBeInTheDocument();
  });

  it("shows Excluded badge for excluded movies", async () => {
    serveBrowse();
    renderPage();
    expect(await screen.findByText("Excluded")).toBeInTheDocument();
  });

  it("shows TMDB rating", async () => {
    serveBrowse();
    renderPage();
    expect(await screen.findByText("8.4")).toBeInTheDocument();
  });

  it("pages through results", async () => {
    const user = userEvent.setup();
    const seen = serveBrowse();
    renderPage();
    expect(await screen.findByText("Page 1 of 5")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(seen.at(-1)?.get("page")).toBe("2"));
  });

  it("New on streaming asks for the new list", async () => {
    const user = userEvent.setup();
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");
    await user.click(screen.getByRole("button", { name: "New on streaming" }));
    await waitFor(() => expect(seen.at(-1)?.get("category")).toBe("new"));
    expect(screen.getByRole("button", { name: "New on streaming" })).toHaveAttribute("aria-pressed", "true");
    expect(window.location.search).toContain("category=new");
  });

  it("combines filters, and a filter on Trending moves to Popular", async () => {
    const user = userEvent.setup();
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");
    await screen.findByRole("option", { name: "Netflix" });

    await user.selectOptions(screen.getByRole("combobox", { name: "Genre" }), "Comedy");
    await waitFor(() => expect(seen.at(-1)?.get("genre")).toBe("35"));
    expect(seen.at(-1)?.get("category")).toBe("popular");

    await user.selectOptions(screen.getByRole("combobox", { name: "Streaming service" }), "Netflix");
    await user.selectOptions(screen.getByRole("combobox", { name: "Minimum rating" }), "7");
    await waitFor(() => {
      const last = seen.at(-1)!;
      expect([last.get("genre"), last.get("provider"), last.get("min_rating")]).toEqual(["35", "8", "7"]);
    });

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await waitFor(() => expect(seen.at(-1)?.get("genre")).toBeNull());
    expect(seen.at(-1)?.get("category")).toBe("popular");
  });

  it("choosing Trending clears the filters", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/discover?category=popular&genre=35");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");
    expect(seen[0].get("genre")).toBe("35");
    await user.click(screen.getByRole("button", { name: "Trending" }));
    await waitFor(() => expect(seen.at(-1)?.get("category")).toBe("trending"));
    expect(seen.at(-1)?.get("genre")).toBeNull();
  });

  it("searches by title", async () => {
    const user = userEvent.setup();
    serveBrowse();
    let q = "";
    server.use(
      http.get("/api/v1/discover/search", ({ request }) => {
        q = new URL(request.url).searchParams.get("q") ?? "";
        return HttpResponse.json({ ...fixtures, results: [{ ...fixtures.results[0], tmdb_id: 77, title: "Dune" }], total_pages: 1 });
      })
    );
    renderPage();
    await screen.findByText("Fight Club");
    await user.type(screen.getByRole("searchbox", { name: "Search movies" }), "dune");
    expect(await screen.findByText("Dune")).toBeInTheDocument();
    expect(q).toBe("dune");
    // The lists and filters step aside while searching.
    expect(screen.queryByRole("button", { name: "Popular" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByText("Fight Club")).toBeInTheDocument();
  });

  it("says when nothing matches the filters", async () => {
    window.history.replaceState({}, "", "/discover?category=new&provider=8");
    serveBrowse({ results: [], page: 1, total_pages: 0 });
    renderPage();
    expect(await screen.findByText("Nothing matches these filters")).toBeInTheDocument();
  });

  it("shows error state on API failure", async () => {
    server.use(http.get(browse, () => HttpResponse.json({ title: "Internal Server Error" }, { status: 500 })));
    renderPage();
    expect(await screen.findByText("Couldn't load this list")).toBeInTheDocument();
  });

  it("explains when TMDB isn't set up", async () => {
    server.use(http.get(browse, () => HttpResponse.json({ title: "Service Unavailable", detail: "TMDB is not configured" }, { status: 503 })));
    renderPage();
    expect(await screen.findByText("TMDB isn't set up")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Settings → Providers" })).toHaveAttribute("href", "/settings/providers");
  });

  it("opens add modal when clicking + Add", async () => {
    const user = userEvent.setup();
    serveBrowse();
    renderPage();
    await user.click(await screen.findByTestId("add-550"));
    expect(await screen.findByText("Add Fight Club")).toBeInTheDocument();
    expect(screen.getByTestId("add-movie-btn")).toBeInTheDocument();
  });
});
