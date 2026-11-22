import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { Star } from "lucide-react";
import { toast } from "sonner";
import {
  useDiscoverBrowse,
  useDiscoverProviders,
  useDiscoverSearch,
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
import { Poster } from "@/components/Poster";
import Modal from "@beacon-shared/Modal";

// ── Add Movie Modal ─────────────────────────────────────────────────────────

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
        onSuccess: () => {
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

// ── Movie Card ──────────────────────────────────────────────────────────────

function DiscoverCard({
  movie,
  onAdd,
}: {
  movie: DiscoverResult;
  onAdd: (m: DiscoverResult) => void;
}) {
  const navigate = useNavigate();

  return (
    <div style={{ width: "100%" }}>
      <Link to={`/discover/${movie.tmdb_id}`} style={{ textDecoration: "none", color: "inherit", display: "block", cursor: "pointer" }}>
        <div style={{ position: "relative" }}>
          <Poster
            src={
              movie.poster_path
                ? `https://image.tmdb.org/t/p/w342${movie.poster_path}`
                : undefined
            }
            title={movie.title}
            year={movie.year}
          />

          {/* Rating badge */}
          {movie.rating > 0 && (
            <div
              style={{
                position: "absolute",
                top: 6,
                right: 6,
                display: "flex",
                alignItems: "center",
                gap: 3,
                padding: "2px 6px",
                borderRadius: 4,
                background: "rgba(0,0,0,0.7)",
                backdropFilter: "blur(4px)",
                fontSize: 11,
                fontWeight: 600,
                color: "#fbbf24",
              }}
            >
              <Star size={10} fill="#fbbf24" stroke="none" />
              {movie.rating.toFixed(1)}
            </div>
          )}
        </div>

        <div style={{ marginTop: 8 }}>
          <span
            style={{
              display: "block",
              fontSize: 12,
              fontWeight: 500,
              color: "var(--color-text-primary)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {movie.title}
          </span>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            {movie.year > 0 && movie.year}
          </span>
        </div>
      </Link>

      {/* Action badge */}
      <div style={{ marginTop: 6 }}>
        {movie.in_library ? (
          <button
            onClick={() =>
              movie.library_movie_id &&
              navigate(`/movies/${movie.library_movie_id}`)
            }
            data-testid={`in-library-${movie.tmdb_id}`}
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              border: "none",
              background:
                "color-mix(in srgb, var(--color-success) 15%, transparent)",
              color: "var(--color-success)",
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            In Library
          </button>
        ) : movie.excluded ? (
          <span
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              background: "var(--color-bg-subtle)",
              color: "var(--color-text-muted)",
              fontWeight: 500,
            }}
          >
            Excluded
          </span>
        ) : (
          <button
            onClick={() => onAdd(movie)}
            data-testid={`add-${movie.tmdb_id}`}
            style={{
              fontSize: 10,
              padding: "3px 8px",
              borderRadius: 4,
              border: "none",
              background:
                "color-mix(in srgb, var(--color-accent) 15%, transparent)",
              color: "var(--color-accent)",
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            + Add
          </button>
        )}
      </div>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

const region = discoverRegion();

export default function DiscoverPage() {
  const s = useDiscoverState();
  const [addingMovie, setAddingMovie] = useState<DiscoverResult | null>(null);
  const browse = useDiscoverBrowse({ category: s.category, page: s.page, region, ...s.filters }, !s.searching);
  const search = useDiscoverSearch(s.query, s.page, s.searching);
  const { data: genres } = useGenreList();
  const { data: providers } = useDiscoverProviders(region);

  const active = s.searching ? search : browse;
  const data = active.data;
  const unavailable = active.error instanceof APIError && active.error.status === 503 ? active.error : null;

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
      />

      {unavailable ? (
        <DiscoverUnavailable app="Prism" message={unavailable.detail} />
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
                <button type="button" onClick={s.clearFilters} style={{ background: "none", border: "none", color: "var(--color-accent)", fontSize: 13, cursor: "pointer" }}>
                  Clear filters
                </button>
              ) : undefined
            }
          />
        )
      ) : data ? (
        <div style={{ opacity: active.isPlaceholderData ? 0.6 : 1, transition: "opacity 120ms" }}>
          <DiscoverGrid>
            {data.results.map((movie) => (
              <DiscoverCard key={movie.tmdb_id} movie={movie} onAdd={setAddingMovie} />
            ))}
          </DiscoverGrid>
          <DiscoverPager page={data.page} totalPages={data.total_pages} onPage={s.setPage} />
        </div>
      ) : null}

      {addingMovie && <QuickAddModal movie={addingMovie} onClose={() => setAddingMovie(null)} />}
    </div>
  );
}
