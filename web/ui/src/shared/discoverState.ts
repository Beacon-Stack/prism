import { useSearchParams } from "react-router-dom";

/** The Discover lists, the same in Pilot and Prism. */
export const DISCOVER_CATEGORIES = [
  { value: "trending", label: "Trending" },
  { value: "popular", label: "Popular" },
  { value: "top-rated", label: "Top rated" },
  { value: "new", label: "New on streaming" },
  { value: "coming-soon", label: "Coming soon" },
] as const;

export type DiscoverCategory = (typeof DISCOVER_CATEGORIES)[number]["value"];

/** Orders that replace a list's own. */
export const DISCOVER_SORTS = [
  { value: "rating", label: "Highest rated" },
  { value: "newest", label: "Newest" },
  { value: "revenue", label: "Box office" },
] as const;

export type DiscoverSort = "" | (typeof DISCOVER_SORTS)[number]["value"];

export const DISCOVER_RUNTIMES = [
  { value: "short", label: "Under 90 min" },
  { value: "medium", label: "90–120 min" },
  { value: "long", label: "Over 2 hours" },
] as const;

export type DiscoverRuntime = "" | (typeof DISCOVER_RUNTIMES)[number]["value"];

/** Original languages offered as a filter (ISO 639-1). */
export const DISCOVER_LANGUAGES = [
  { code: "en", name: "English" },
  { code: "es", name: "Spanish" },
  { code: "fr", name: "French" },
  { code: "de", name: "German" },
  { code: "it", name: "Italian" },
  { code: "pt", name: "Portuguese" },
  { code: "ja", name: "Japanese" },
  { code: "ko", name: "Korean" },
  { code: "zh", name: "Chinese" },
  { code: "cn", name: "Cantonese" },
  { code: "hi", name: "Hindi" },
  { code: "sv", name: "Swedish" },
  { code: "da", name: "Danish" },
  { code: "no", name: "Norwegian" },
  { code: "ru", name: "Russian" },
  { code: "tr", name: "Turkish" },
] as const;

export interface DiscoverFilters {
  /** The first of `genres`, for an app that filters by one genre. */
  genre: number;
  provider: number;
  minRating: number;
  /** Titles must be in every one of these. */
  genres: number[];
  sort: DiscoverSort;
  /** First year of a decade: 1990 is 1990–1999. */
  decade: number;
  runtime: DiscoverRuntime;
  /** Original language (ISO 639-1). */
  language: string;
  /** Leave out what's already in the library. */
  hideOwned: boolean;
}

export type DiscoverFilterValue = number | string | boolean | number[];

/**
 * One thing the list is narrowed to, reached by a link rather than a
 * filter control: a keyword, a studio, or a collection of titles.
 */
export interface DiscoverPivot {
  kind: "keyword" | "company" | "collection";
  id: number;
  name: string;
}

const PIVOT_KINDS = ["keyword", "company", "collection"] as const;

/** Searches start at this many characters. */
export const MIN_SEARCH = 2;

/**
 * The viewer's country from the browser locale ("en-GB" → "GB"), for
 * streaming services and release dates. US when the locale has none.
 */
export function discoverRegion(): string {
  try {
    const langs = typeof navigator !== "undefined" ? [...(navigator.languages ?? []), navigator.language] : [];
    for (const l of langs) {
      if (!l) continue;
      const region = new Intl.Locale(l).maximize().region;
      if (region && /^[A-Z]{2}$/.test(region)) return region;
    }
  } catch {
    // Old browser or odd locale string: fall through.
  }
  return "US";
}

function num(v: string | null): number {
  const n = Number(v);
  return Number.isFinite(n) && n > 0 ? n : 0;
}

function oneOf<T extends string>(options: readonly { value: T }[], v: string | null): T | "" {
  return options.some((o) => o.value === v) ? (v as T) : "";
}

const URL_KEYS: Record<keyof DiscoverFilters, string> = {
  genre: "genre",
  genres: "genre",
  provider: "provider",
  minRating: "min_rating",
  sort: "sort",
  decade: "decade",
  runtime: "runtime",
  language: "language",
  hideOwned: "hide_owned",
};

function encode(value: DiscoverFilterValue): string | null {
  if (Array.isArray(value)) return value.length ? value.join(",") : null;
  if (typeof value === "boolean") return value ? "1" : null;
  return value ? String(value) : null;
}

/**
 * Whether a list can apply a filter. Trending is a fixed list; New on
 * streaming and Coming soon set their own dates, so a decade means
 * nothing there; and nothing unreleased has a rating or box office to
 * sort by. Hiding what's in the library works on any list.
 */
function fits(category: DiscoverCategory, key: keyof DiscoverFilters, value: DiscoverFilterValue): boolean {
  if (key === "hideOwned") return true;
  if (category === "trending") return false;
  if (key === "decade") return category === "popular" || category === "top-rated";
  if (key === "sort" && category === "coming-soon") return value === "newest";
  return true;
}

const NO_PIVOT = { keyword: null, company: null, collection: null, pivot: null };
/** Everything a fixed list can't apply. */
const NO_LIST_FILTERS = { genre: null, provider: null, min_rating: null, sort: null, decade: null, runtime: null, language: null, ...NO_PIVOT };

interface DiscoverStateOptions {
  /**
   * The app has a home view, shown when the URL names no list. Without
   * it, a URL with no list shows Trending.
   */
  home?: boolean;
}

/**
 * Discover's search, list, filters and page, kept in the URL so the back
 * button from a title's page returns to the same list.
 */
export function useDiscoverState(options: DiscoverStateOptions = {}) {
  const [params, setParams] = useSearchParams();

  const raw = params.get("category");
  const named = DISCOVER_CATEGORIES.some((c) => c.value === raw);
  const category: DiscoverCategory = named ? (raw as DiscoverCategory) : "trending";
  const query = params.get("q") ?? "";
  const searching = query.trim().length >= MIN_SEARCH;
  const page = Math.max(1, Math.floor(num(params.get("page"))) || 1);

  const genres = [...new Set((params.get("genre") ?? "").split(",").map(num).filter(Boolean))];
  const language = params.get("language") ?? "";
  const filters: DiscoverFilters = {
    genre: genres[0] ?? 0,
    genres,
    provider: num(params.get("provider")),
    minRating: num(params.get("min_rating")),
    sort: oneOf(DISCOVER_SORTS, params.get("sort")),
    decade: num(params.get("decade")),
    runtime: oneOf(DISCOVER_RUNTIMES, params.get("runtime")),
    language: /^[a-z]{2}$/.test(language) ? language : "",
    hideOwned: params.get("hide_owned") === "1",
  };
  const listFilterCount = [genres.length, filters.provider, filters.minRating, filters.decade, filters.runtime, filters.language].filter(Boolean).length;
  const filterCount = listFilterCount + (filters.hideOwned ? 1 : 0);

  const pivotKind = PIVOT_KINDS.find((k) => num(params.get(k)) > 0);
  const pivot: DiscoverPivot | null = pivotKind ? { kind: pivotKind, id: num(params.get(pivotKind)), name: params.get("pivot") ?? "" } : null;

  const home = !!options.home && !named && !searching && listFilterCount === 0 && !filters.sort && !pivot;

  function update(patch: Record<string, string | number | null>, keepPage = false) {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(patch)) {
      if (v === null || v === "" || v === 0) next.delete(k);
      else next.set(k, String(v));
    }
    if (!keepPage) next.delete("page");
    setParams(next, { replace: true });
  }

  return {
    query,
    category,
    page,
    filters,
    filterCount,
    pivot,
    searching,
    /** No list is chosen: show the app's home view. Only with the `home` option. */
    home,
    setQuery: (q: string) => update({ q: q.trim() ? q : null }),
    // Choosing a list drops whatever it can't apply.
    setCategory: (c: DiscoverCategory) => {
      const patch: Record<string, string | null> = { category: c === "trending" && !options.home ? null : c };
      if (c === "trending") Object.assign(patch, NO_LIST_FILTERS);
      if (!fits(c, "decade", filters.decade)) patch.decade = null;
      if (filters.sort && !fits(c, "sort", filters.sort)) patch.sort = null;
      update(patch);
    },
    // ...and choosing a filter the current list can't apply moves to Popular.
    setFilter: (key: keyof DiscoverFilters, value: DiscoverFilterValue) => {
      const encoded = encode(value);
      update({
        [URL_KEYS[key]]: encoded,
        ...(encoded !== null && !fits(category, key, value) ? { category: "popular" } : {}),
      });
    },
    clearFilters: () => update({ genre: null, provider: null, min_rating: null, decade: null, runtime: null, language: null, hide_owned: null }),
    clearPivot: () => update(NO_PIVOT),
    /** Back to the home view: no list, search, filters or sort. */
    goHome: () => update({ category: null, q: null, ...NO_LIST_FILTERS }),
    setPage: (p: number) => update({ page: p > 1 ? p : null }, true),
  };
}
