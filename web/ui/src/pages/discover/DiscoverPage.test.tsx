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

// The page keeps its state in the URL; start each test on the Trending
// list. The home view, at /discover, has its own tests below.
beforeEach(() => window.history.replaceState({}, "", "/discover?category=trending"));

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
    expect(screen.getByRole("button", { name: "For you" })).toBeInTheDocument();
    expect(await screen.findByRole("option", { name: "Netflix" })).toBeInTheDocument();
    for (const name of ["Minimum rating", "Decade", "Runtime", "Language", "Sort"]) {
      expect(screen.getByRole("combobox", { name })).toBeInTheDocument();
    }
    expect(screen.getByRole("button", { name: "Any genre" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Hide what I have" })).toBeInTheDocument();
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

    await user.click(screen.getByRole("button", { name: "Any genre" }));
    await user.click(await screen.findByRole("button", { name: "Comedy" }));
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
  it("picks several genres at once", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/discover?category=popular");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");

    await user.click(screen.getByRole("button", { name: "Any genre" }));
    await user.click(await screen.findByRole("button", { name: "Comedy" }));
    await user.click(screen.getByRole("button", { name: "Action" }));
    await waitFor(() => expect(seen.at(-1)?.get("genre")).toBe("35,28"));
    expect(screen.getByRole("button", { name: "Action + Comedy" })).toHaveAttribute("aria-expanded", "true");

    await user.click(screen.getByRole("button", { name: "Comedy" }));
    await waitFor(() => expect(seen.at(-1)?.get("genre")).toBe("28"));
  });

  it("sends decade, runtime, language and sort", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/discover?category=top-rated");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");

    await user.selectOptions(screen.getByRole("combobox", { name: "Decade" }), "1990s");
    await user.selectOptions(screen.getByRole("combobox", { name: "Runtime" }), "Over 2 hours");
    await user.selectOptions(screen.getByRole("combobox", { name: "Language" }), "Korean");
    await user.selectOptions(screen.getByRole("combobox", { name: "Sort" }), "Newest");
    await waitFor(() => {
      const last = seen.at(-1)!;
      expect([last.get("category"), last.get("decade"), last.get("runtime"), last.get("language"), last.get("sort")]).toEqual([
        "top-rated",
        "1990",
        "long",
        "ko",
        "newest",
      ]);
    });
  });

  it("a decade on New on streaming moves to Popular, and Coming soon drops what it can't apply", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/discover?category=new");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");

    await user.selectOptions(screen.getByRole("combobox", { name: "Decade" }), "1980s");
    await waitFor(() => expect(seen.at(-1)?.get("decade")).toBe("1980"));
    expect(seen.at(-1)?.get("category")).toBe("popular");

    await user.selectOptions(screen.getByRole("combobox", { name: "Sort" }), "Highest rated");
    await user.selectOptions(screen.getByRole("combobox", { name: "Runtime" }), "Under 90 min");
    await waitFor(() => expect(seen.at(-1)?.get("runtime")).toBe("short"));

    await user.click(screen.getByRole("button", { name: "Coming soon" }));
    await waitFor(() => expect(seen.at(-1)?.get("category")).toBe("coming-soon"));
    const last = seen.at(-1)!;
    expect([last.get("decade"), last.get("sort"), last.get("runtime")]).toEqual([null, null, "short"]);
  });

  it("hides what's already in the library, on any list", async () => {
    const user = userEvent.setup();
    const seen = serveBrowse();
    renderPage();
    expect(await screen.findByText("The Dark Knight")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Hide what I have" }));
    await waitFor(() => expect(screen.queryByText("The Dark Knight")).not.toBeInTheDocument());
    expect(screen.getByText("Fight Club")).toBeInTheDocument();
    // Hiding happens here, not at TMDB: the list stays Trending.
    expect(seen.at(-1)?.get("category")).toBe("trending");
    expect(window.location.search).toContain("hide_owned=1");
  });

  it("says so when the whole page is already in the library", async () => {
    window.history.replaceState({}, "", "/discover?category=popular&hide_owned=1");
    serveBrowse({ results: [fixtures.results[1]], page: 1, total_pages: 3 });
    renderPage();
    expect(await screen.findByText("You have everything on this page")).toBeInTheDocument();
    expect(screen.getByText("Page 1 of 3")).toBeInTheDocument();
  });

  it("narrows to a keyword from a link, and removes it", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/discover?category=popular&keyword=10051&pivot=heist");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");
    expect(seen[0].get("keyword")).toBe("10051");
    expect(screen.getByText("Keyword: heist")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove keyword heist" }));
    await waitFor(() => expect(seen.at(-1)?.get("keyword")).toBeNull());
    expect(seen.at(-1)?.get("category")).toBe("popular");
    expect(screen.queryByText("Keyword: heist")).not.toBeInTheDocument();
  });

  it("shows a collection as its own list, without lists or filters", async () => {
    window.history.replaceState({}, "", "/discover?collection=263&pivot=The+Dark+Knight+Collection");
    const seen = serveBrowse();
    renderPage();
    await screen.findByText("Fight Club");
    expect(seen[0].get("collection")).toBe("263");
    expect(screen.getByText("Collection: The Dark Knight Collection")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Popular" })).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Decade" })).not.toBeInTheDocument();
  });

  it("a search also offers people, collections, keywords and studios", async () => {
    const user = userEvent.setup();
    serveBrowse();
    server.use(
      http.get("/api/v1/discover/suggest", () =>
        HttpResponse.json({
          people: [{ id: 525, name: "Christopher Nolan", detail: "Directing" }],
          collections: [{ id: 263, name: "The Dark Knight Collection" }],
          keywords: [{ id: 10051, name: "heist" }],
          companies: [{ id: 41077, name: "A24" }],
        })
      )
    );
    renderPage();
    await screen.findByText("Fight Club");
    await user.type(screen.getByRole("searchbox", { name: "Search movies" }), "nolan");

    expect(await screen.findByRole("link", { name: /Christopher Nolan/ })).toHaveAttribute("href", "/people/525");
    expect(screen.getByRole("link", { name: "The Dark Knight Collection" })).toHaveAttribute(
      "href",
      "/discover?collection=263&pivot=The+Dark+Knight+Collection"
    );
    expect(screen.getByRole("link", { name: "heist" })).toHaveAttribute("href", "/discover?keyword=10051&pivot=heist&category=popular");
    expect(screen.getByRole("link", { name: "A24" })).toHaveAttribute("href", "/discover?company=41077&pivot=A24&category=popular");
  });
});

describe("DiscoverPage home", () => {
  const shelves = {
    shelves: [
      { id: "because-550", kind: "because", title: "Because you have Fight Club", results: [{ ...fixtures.results[0], tmdb_id: 807, title: "Se7en" }] },
      {
        id: "genre-18",
        kind: "genre",
        title: "Top-rated Drama you don't have",
        see_all: "/discover?category=top-rated&genre=18&hide_owned=1",
        results: [{ ...fixtures.results[0], tmdb_id: 13, title: "Forrest Gump" }],
      },
    ],
  };

  function serveShelves(data: object = shelves) {
    const seen: URLSearchParams[] = [];
    server.use(
      http.get("/api/v1/discover/shelves", ({ request }) => {
        seen.push(new URL(request.url).searchParams);
        return HttpResponse.json(data);
      })
    );
    return seen;
  }

  beforeEach(() => window.history.replaceState({}, "", "/discover"));

  it("opens on rows picked from the library", async () => {
    const seen = serveShelves();
    const browsed = serveBrowse();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Because you have Fight Club" })).toBeInTheDocument();
    expect(screen.getByText("Se7en")).toBeInTheDocument();
    expect(screen.getByText("Forrest Gump")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "See all" })).toHaveAttribute("href", "/discover?category=top-rated&genre=18&hide_owned=1");
    expect(screen.getByRole("button", { name: "For you" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Trending" })).toHaveAttribute("aria-pressed", "false");
    expect(Number(seen[0].get("seed"))).toBeGreaterThan(0);
    expect(browsed).toHaveLength(0);
  });

  it("Shuffle asks for different rows", async () => {
    const user = userEvent.setup();
    const seen = serveShelves();
    renderPage();
    await screen.findByText("Se7en");
    await user.click(screen.getByRole("button", { name: "Shuffle" }));
    await waitFor(() => expect(seen).toHaveLength(2));
    expect(seen[1].get("seed")).not.toBe(seen[0].get("seed"));
  });

  it("goes from the rows to a list and back", async () => {
    const user = userEvent.setup();
    serveShelves();
    const browsed = serveBrowse();
    renderPage();
    await screen.findByText("Se7en");

    await user.click(screen.getByRole("button", { name: "Trending" }));
    expect(await screen.findByText("The Dark Knight")).toBeInTheDocument();
    expect(browsed.at(-1)?.get("category")).toBe("trending");
    expect(screen.queryByText("Se7en")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "For you" }));
    expect(await screen.findByText("Se7en")).toBeInTheDocument();
    expect(window.location.search).toBe("");
  });

  it("a filter chosen on the home view opens Popular with it", async () => {
    const user = userEvent.setup();
    serveShelves();
    const browsed = serveBrowse();
    renderPage();
    await screen.findByText("Se7en");
    await user.selectOptions(screen.getByRole("combobox", { name: "Decade" }), "1990s");
    await waitFor(() => expect(browsed.at(-1)?.get("decade")).toBe("1990"));
    expect(browsed.at(-1)?.get("category")).toBe("popular");
  });

  it("an added movie's card says In Library without the rows changing", async () => {
    const user = userEvent.setup();
    const seen = serveShelves();
    server.use(
      http.get("/api/v1/libraries", () => HttpResponse.json([{ id: "lib-1", name: "Movies" }])),
      http.get("/api/v1/quality-profiles", () => HttpResponse.json([{ id: "qp-1", name: "HD" }])),
      http.post("/api/v1/movies", () => HttpResponse.json({ id: "movie-9", tmdb_id: 807, title: "Se7en" }))
    );
    renderPage();
    await user.click(await screen.findByTestId("add-807"));
    await screen.findByRole("option", { name: "Movies" });
    await screen.findByRole("option", { name: "HD" });
    await user.click(screen.getByTestId("add-movie-btn"));

    expect(await screen.findByTestId("in-library-807")).toBeInTheDocument();
    expect(screen.getByText("Forrest Gump")).toBeInTheDocument();
    expect(seen).toHaveLength(1);
  });

  it("offers Trending when there is nothing to suggest", async () => {
    serveShelves({ shelves: [] });
    renderPage();
    expect(await screen.findByText("Nothing to suggest right now")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Browse Trending" })).toBeInTheDocument();
  });

  it("explains when TMDB isn't set up", async () => {
    server.use(http.get("/api/v1/discover/shelves", () => HttpResponse.json({ title: "Service Unavailable", detail: "TMDB is not configured" }, { status: 503 })));
    renderPage();
    expect(await screen.findByText("TMDB isn't set up")).toBeInTheDocument();
  });
});
