import { useState } from "react";
import { keepPreviousData, useQuery, type QueryClient } from "@tanstack/react-query";
import type { DiscoverPivot } from "@beacon-shared/discoverState";
import { apiFetch } from "./client";

export interface DiscoverResult {
  tmdb_id: number;
  title: string;
  year: number;
  overview: string;
  poster_path: string;
  rating: number;
  in_library: boolean;
  excluded: boolean;
  library_movie_id?: string;
}

export interface DiscoverListResponse {
  results: DiscoverResult[];
  page: number;
  total_pages: number;
}

export interface TMDBGenre {
  id: number;
  name: string;
}

export function useGenreList() {
  return useQuery({
    queryKey: ["discover", "genres"],
    queryFn: () => apiFetch<TMDBGenre[]>("/discover/genres"),
    staleTime: Infinity,
  });
}

interface DiscoverCast {
  id: number;
  name: string;
  character: string;
  profile_path: string;
}

interface DiscoverCrew {
  id: number;
  name: string;
  job: string;
  profile_path: string;
}

interface DiscoverRecommendation {
  tmdb_id: number;
  title: string;
  year: number;
  poster_path: string;
}

export interface DiscoverMovieDetail {
  tmdb_id: number;
  imdb_id?: string;
  title: string;
  original_title: string;
  year: number;
  overview: string;
  release_date: string;
  runtime_minutes: number;
  genres: string[];
  poster_path: string;
  backdrop_path: string;
  status: string;
  rating: number;
  in_library: boolean;
  excluded: boolean;
  library_movie_id?: string;
  cast: DiscoverCast[];
  crew: DiscoverCrew[];
  recommendations: DiscoverRecommendation[];
  genre_refs: TMDBGenre[];
  keywords: DiscoverRef[];
  companies: DiscoverRef[];
  collection?: DiscoverRef;
}

/** A person, collection, keyword or studio a list of movies can be built from. */
export interface DiscoverRef {
  id: number;
  name: string;
  image_path?: string;
  /** What a person is known for ("Directing"). */
  detail?: string;
}

/** The Discover address that lists the movies with a keyword, from a studio or in a collection. */
export function pivotLink(kind: DiscoverPivot["kind"], ref: DiscoverRef): string {
  const qs = new URLSearchParams({ [kind]: String(ref.id), pivot: ref.name });
  // A collection is its own list; the others narrow Popular.
  if (kind !== "collection") qs.set("category", "popular");
  return `/discover?${qs}`;
}

export function useDiscoverMovie(tmdbId: number) {
  return useQuery({
    queryKey: ["discover", "movie", tmdbId],
    queryFn: () => apiFetch<DiscoverMovieDetail>(`/discover/${tmdbId}`),
    enabled: tmdbId > 0,
    staleTime: 5 * 60_000,
  });
}

// Browse with filters, search, streaming services

export interface DiscoverBrowseParams {
  category: string;
  page: number;
  region: string;
  genres: number[];
  provider: number;
  minRating: number;
  sort: string;
  decade: number;
  runtime: string;
  language: string;
  pivot: DiscoverPivot | null;
}

export interface StreamingService {
  id: number;
  name: string;
  logo_path: string;
}

export function useDiscoverBrowse(p: DiscoverBrowseParams, enabled = true) {
  const set: Record<string, string | number> = {
    genre: p.genres.join(","),
    provider: p.provider,
    min_rating: p.minRating,
    sort: p.sort,
    decade: p.decade,
    runtime: p.runtime,
    language: p.language,
    ...(p.pivot ? { [p.pivot.kind]: p.pivot.id } : {}),
  };
  const qs = new URLSearchParams({ category: p.category, page: String(p.page), region: p.region });
  for (const [k, v] of Object.entries(set)) {
    if (v) qs.set(k, String(v));
  }
  return useQuery({
    queryKey: ["discover", "browse", qs.toString()],
    queryFn: () => apiFetch<DiscoverListResponse>(`/discover/browse?${qs}`),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
  });
}

// Home view: rows picked from the library

export interface DiscoverShelf {
  id: string;
  kind: "because" | "genre" | "collection" | "list";
  title: string;
  /** App path that shows the whole list. */
  see_all?: string;
  results: DiscoverResult[];
}

interface DiscoverShelvesResponse {
  shelves: DiscoverShelf[];
}

function newSeed(): number {
  return 1 + Math.floor(Math.random() * 2_000_000_000);
}

// Which library movies the rows start from. Kept for the life of the tab
// so coming back from a movie's page shows the same rows; a reload or
// Shuffle draws new ones.
let shelfSeed = newSeed();

export function useDiscoverShelves(region: string, enabled = true) {
  const [seed, setSeed] = useState(shelfSeed);
  const query = useQuery({
    queryKey: ["discover", "shelves", region, seed],
    queryFn: () => apiFetch<DiscoverShelvesResponse>(`/discover/shelves?region=${region}&seed=${seed}`),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 15 * 60_000,
  });
  function shuffle() {
    shelfSeed = newSeed();
    setSeed(shelfSeed);
  }
  return { ...query, shuffle };
}

/**
 * After a movie is added from Discover: lists are fetched again so its
 * card says In Library. The home rows are marked in place instead, since
 * fetching them again would rearrange them under the viewer.
 */
export function markAddedInDiscover(qc: QueryClient, tmdbId: number, movieId: string) {
  qc.setQueriesData<DiscoverShelvesResponse>({ queryKey: ["discover", "shelves"] }, (old) =>
    old && {
      shelves: old.shelves.map((shelf) => ({
        ...shelf,
        results: shelf.results.map((m) => (m.tmdb_id === tmdbId ? { ...m, in_library: true, library_movie_id: movieId } : m)),
      })),
    }
  );
  qc.invalidateQueries({ queryKey: ["discover", "browse"] });
  qc.invalidateQueries({ queryKey: ["discover", "search"] });
  qc.invalidateQueries({ queryKey: ["discover", "movie", tmdbId] });
}

// Search beyond titles

export interface DiscoverSuggestions {
  people: DiscoverRef[];
  collections: DiscoverRef[];
  keywords: DiscoverRef[];
  companies: DiscoverRef[];
}

export function useDiscoverSuggest(query: string, enabled = true) {
  const q = query.trim();
  return useQuery({
    queryKey: ["discover", "suggest", q],
    queryFn: () => apiFetch<DiscoverSuggestions>(`/discover/suggest?q=${encodeURIComponent(q)}`),
    enabled: enabled && q.length > 0,
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
  });
}

export function useDiscoverSearch(query: string, page: number, enabled = true) {
  const q = query.trim();
  return useQuery({
    queryKey: ["discover", "search", q, page],
    queryFn: () => apiFetch<DiscoverListResponse>(`/discover/search?q=${encodeURIComponent(q)}&page=${page}`),
    enabled: enabled && q.length > 0,
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
  });
}

export function useDiscoverProviders(region: string) {
  return useQuery({
    queryKey: ["discover", "providers", region],
    queryFn: () => apiFetch<StreamingService[]>(`/discover/providers?region=${region}`),
    staleTime: 24 * 60 * 60_000,
  });
}
