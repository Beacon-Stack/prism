import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  markAddedInDiscover,
  useDiscoverBrowse,
  useDiscoverProviders,
  useDiscoverSearch,
  useDiscoverShelves,
  useDiscoverSuggest,
  useGenreList,
  type DiscoverResult,
} from "@/api/discover";
import PageHeader from "@/components/PageHeader";
import { APIError } from "@beacon-shared/api";
import {
  DiscoverControls,
  DiscoverEmpty,
  DiscoverGrid,
  DiscoverLoading,
  DiscoverPager,
  DiscoverUnavailable,
} from "@beacon-shared/Discover";
import { discoverRegion, useDiscoverState } from "@beacon-shared/discoverState";
import { useAddMovie } from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import Modal from "@beacon-shared/Modal";
import DiscoverCard from "./DiscoverCard";
import DiscoverShelves from "./DiscoverShelves";
import SearchSuggestions from "./SearchSuggestions";

// Add Movie Modal

function QuickAddModal({
  movie,
  onClose,
}: {
  movie: DiscoverResult;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMovie = useAddMovie();
  const qc = useQueryClient();

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");

  // Auto-select first library/profile
  const lib = libraryId || libraries?.[0]?.id || "";
  const prof = profileId || profiles?.[0]?.id || "";

  function handleAdd() {
    if (!lib || !prof) return;
    addMovie.mutate(
      {
        tmdb_id: movie.tmdb_id,
        library_id: lib,
        quality_profile_id: prof,
        monitored: true,
      },
      {
        onSuccess: (added) => {
          markAddedInDiscover(qc, movie.tmdb_id, added.id);
          toast.success(`Added ${movie.title}`);
          onClose();
        },
        onError: (err) => toast.error((err as Error).message),
      }
    );
  }

  const inputStyle: React.CSSProperties = {
    width: "100%",
    padding: "8px 10px",
    borderRadius: 6,
    border: "1px solid var(--color-border-default)",
    background: "var(--color-bg-elevated)",
    color: "var(--color-text-primary)",
    fontSize: 13,
  };

  return (
    <Modal onClose={onClose} width={400}>
      <div style={{ padding: 24 }}>
        <h3
          style={{
            margin: "0 0 4px",
            fontSize: 16,
            fontWeight: 600,
            color: "var(--color-text-primary)",
          }}
        >
          Add {movie.title}
        </h3>
        <p
          style={{
            margin: "0 0 20px",
            fontSize: 12,
            color: "var(--color-text-muted)",
          }}
        >
          {movie.year > 0 && movie.year}
        </p>

        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Library
            </label>
            <select
              value={lib}
              onChange={(e) => setLibraryId(e.target.value)}
              style={inputStyle}
              data-testid="add-library-select"
            >
              {libraries?.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                marginBottom: 4,
              }}
            >
              Quality Profile
            </label>
            <select
              value={prof}
              onChange={(e) => setProfileId(e.target.value)}
              style={inputStyle}
              data-testid="add-profile-select"
            >
              {profiles?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "flex-end",
            gap: 8,
            marginTop: 20,
          }}
        >
          <button
            onClick={onClose}
            style={{
              padding: "7px 14px",
              borderRadius: 6,
              border: "1px solid var(--color-border-default)",
              background: "transparent",
              color: "var(--color-text-secondary)",
              fontSize: 13,
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={!lib || !prof || addMovie.isPending}
            data-testid="add-movie-btn"
            style={{
              padding: "7px 16px",
              borderRadius: 6,
              border: "none",
              background:
                !lib || !prof || addMovie.isPending
                  ? "var(--color-bg-subtle)"
                  : "var(--color-accent)",
              color:
                !lib || !prof || addMovie.isPending
                  ? "var(--color-text-muted)"
                  : "var(--color-accent-fg)",
              fontSize: 13,
              fontWeight: 500,
              cursor:
                !lib || !prof || addMovie.isPending
                  ? "not-allowed"
                  : "pointer",
            }}
          >
            {addMovie.isPending ? "Adding..." : "Add Movie"}
          </button>
        </div>
      </div>
    </Modal>
  );
}

// Page

const region = discoverRegion();

export default function DiscoverPage() {
  const s = useDiscoverState({ home: true });
  const [addingMovie, setAddingMovie] = useState<DiscoverResult | null>(null);
  const shelves = useDiscoverShelves(region, s.home);
  const browse = useDiscoverBrowse(
    {
      category: s.category,
      page: s.page,
      region,
      genres: s.filters.genres,
      provider: s.filters.provider,
      minRating: s.filters.minRating,
      sort: s.filters.sort,
      decade: s.filters.decade,
      runtime: s.filters.runtime,
      language: s.filters.language,
      pivot: s.pivot,
    },
    !s.searching && !s.home
  );
  const search = useDiscoverSearch(s.query, s.page, s.searching);
  const suggest = useDiscoverSuggest(s.query, s.searching);
  const { data: genres } = useGenreList();
  const { data: providers } = useDiscoverProviders(region);

  const active = s.searching ? search : s.home ? shelves : browse;
  const data = s.home ? undefined : (s.searching ? search : browse).data;
  const unavailable = active.error instanceof APIError && active.error.status === 503 ? active.error : null;

  // "Hide what I have" is applied to the page TMDB returned, so a page can
  // come up short, or empty while later pages still have movies.
  const hiding = s.filters.hideOwned && !s.searching;
  const shown = data ? (hiding ? data.results.filter((m) => !m.in_library) : data.results) : [];
  const linkButton: React.CSSProperties = { background: "none", border: "none", color: "var(--color-accent)", fontSize: 13, cursor: "pointer" };

  return (
    <div style={{ padding: 24, maxWidth: 1200 }}>
      <PageHeader title="Discover" description="Find movies to add to your library." />

      <DiscoverControls
        noun="movies"
        query={s.query}
        onQuery={s.setQuery}
        busy={s.searching && search.isFetching}
        category={s.category}
        onCategory={s.setCategory}
        filters={s.filters}
        filterCount={s.filterCount}
        onFilter={s.setFilter}
        onClearFilters={s.clearFilters}
        genres={genres}
        providers={providers}
        region={region}
        more
        home={s.home}
        onHome={s.goHome}
        pivot={s.pivot}
        onClearPivot={s.clearPivot}
      />

      {s.searching && suggest.data && <SearchSuggestions suggestions={suggest.data} />}

      {unavailable ? (
        <DiscoverUnavailable app="Prism" message={unavailable.detail} />
      ) : s.home ? (
        <DiscoverShelves
          shelves={shelves.data?.shelves}
          loading={shelves.isLoading}
          failed={shelves.isError}
          shuffling={shelves.isPlaceholderData}
          onShuffle={shelves.shuffle}
          onAdd={setAddingMovie}
          onBrowse={() => s.setCategory("trending")}
        />
      ) : active.isLoading ? (
        <DiscoverLoading />
      ) : active.isError ? (
        <DiscoverEmpty title="Couldn't load this list" hint="TMDB didn't answer. Try again in a moment." />
      ) : data && data.results.length === 0 ? (
        s.searching ? (
          <DiscoverEmpty title="No movies found" hint="Try a different title." />
        ) : (
          <DiscoverEmpty
            title="Nothing matches these filters"
            hint={s.filterCount ? "Try fewer filters or a different list." : undefined}
            action={
              s.filterCount ? (
                <button type="button" onClick={s.clearFilters} style={linkButton}>
                  Clear filters
                </button>
              ) : undefined
            }
          />
        )
      ) : data ? (
        <div style={{ opacity: active.isPlaceholderData ? 0.6 : 1, transition: "opacity 120ms" }}>
          {shown.length === 0 ? (
            <DiscoverEmpty
              title="You have everything on this page"
              hint={data.page < data.total_pages ? "There may be more on the next page." : undefined}
              action={
                <button type="button" onClick={() => s.setFilter("hideOwned", false)} style={linkButton}>
                  Show what I have
                </button>
              }
            />
          ) : (
            <DiscoverGrid>
              {shown.map((movie) => (
                <DiscoverCard key={movie.tmdb_id} movie={movie} onAdd={setAddingMovie} />
              ))}
            </DiscoverGrid>
          )}
          <DiscoverPager page={data.page} totalPages={data.total_pages} onPage={s.setPage} />
        </div>
      ) : null}

      {addingMovie && <QuickAddModal movie={addingMovie} onClose={() => setAddingMovie(null)} />}
    </div>
  );
}
