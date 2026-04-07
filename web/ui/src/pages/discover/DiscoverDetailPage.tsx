import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Star, Clock, Calendar } from "lucide-react";
import { toast } from "sonner";
import { useDiscoverMovie, type DiscoverMovieDetail } from "@/api/discover";
import { useAddMovie } from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import { Poster } from "@/components/Poster";
import Modal from "@/components/Modal";

function formatRuntime(minutes: number): string {
  if (!minutes) return "";
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

// ── Add Movie Modal (same as DiscoverPage) ──────────────────────────────────

function QuickAddModal({
  movie,
  onClose,
}: {
  movie: { tmdb_id: number; title: string; year: number };
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const addMovie = useAddMovie();

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");

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
        <h3 style={{ margin: "0 0 4px", fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
          Add {movie.title}
        </h3>
        <p style={{ margin: "0 0 20px", fontSize: 12, color: "var(--color-text-muted)" }}>
          {movie.year > 0 && movie.year}
        </p>

        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div>
            <label style={{ display: "block", fontSize: 12, fontWeight: 500, color: "var(--color-text-secondary)", marginBottom: 4 }}>
              Library
            </label>
            <select value={lib} onChange={(e) => setLibraryId(e.target.value)} style={inputStyle}>
              {libraries?.map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label style={{ display: "block", fontSize: 12, fontWeight: 500, color: "var(--color-text-secondary)", marginBottom: 4 }}>
              Quality Profile
            </label>
            <select value={prof} onChange={(e) => setProfileId(e.target.value)} style={inputStyle}>
              {profiles?.map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </div>
        </div>

        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, marginTop: 20 }}>
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
            style={{
              padding: "7px 16px",
              borderRadius: 6,
              border: "none",
              background: !lib || !prof || addMovie.isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
              color: !lib || !prof || addMovie.isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              fontSize: 13,
              fontWeight: 500,
              cursor: !lib || !prof || addMovie.isPending ? "not-allowed" : "pointer",
            }}
          >
            {addMovie.isPending ? "Adding..." : "Add Movie"}
          </button>
        </div>
      </div>
    </Modal>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function DiscoverDetailPage() {
  const { tmdbId } = useParams<{ tmdbId: string }>();
  const navigate = useNavigate();
  const { data: movie, isLoading, error } = useDiscoverMovie(Number(tmdbId) || 0);
  const [showAdd, setShowAdd] = useState(false);

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: "flex", flexDirection: "column", gap: 20 }}>
        <div className="skeleton" style={{ height: 16, width: 80, borderRadius: 4 }} />
        <div className="skeleton" style={{ height: 24, width: 300, borderRadius: 4 }} />
        <div style={{ display: "flex", gap: 24 }}>
          <div className="skeleton" style={{ width: 200, height: 300, borderRadius: 8, flexShrink: 0 }} />
          <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="skeleton" style={{ height: 20, borderRadius: 4 }} />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error || !movie) {
    return (
      <div style={{ padding: 24 }}>
        <Link to="/discover" style={{ fontSize: 13, color: "var(--color-accent)", textDecoration: "none" }}>
          ← Discover
        </Link>
        <p style={{ marginTop: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
          Movie not found or failed to load.
        </p>
      </div>
    );
  }

  return (
    <div style={{ padding: 24, maxWidth: 1000, position: "relative" }}>
      {/* Backdrop */}
      {movie.backdrop_path && (
        <div
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: 280,
            backgroundImage: `url(https://image.tmdb.org/t/p/w1280${movie.backdrop_path})`,
            backgroundSize: "cover",
            backgroundPosition: "center top",
            opacity: 0.12,
            maskImage: "linear-gradient(to bottom, black 40%, transparent 100%)",
            WebkitMaskImage: "linear-gradient(to bottom, black 40%, transparent 100%)",
            pointerEvents: "none",
            borderRadius: 12,
          }}
        />
      )}

      {/* Back link */}
      <Link
        to="/discover"
        style={{ fontSize: 13, color: "var(--color-text-muted)", textDecoration: "none", display: "inline-block", marginBottom: 20 }}
        onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-primary)"; }}
        onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-muted)"; }}
      >
        ← Discover
      </Link>

      {/* Header row */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24, gap: 16 }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700, color: "var(--color-text-primary)", letterSpacing: "-0.02em" }}>
            {movie.title}
          </h1>
          {movie.year > 0 && (
            <p style={{ margin: "2px 0 0", fontSize: 14, color: "var(--color-text-muted)" }}>{movie.year}</p>
          )}
        </div>
        <div style={{ flexShrink: 0 }}>
          {movie.in_library ? (
            <button
              onClick={() => movie.library_movie_id && navigate(`/movies/${movie.library_movie_id}`)}
              style={{
                padding: "7px 16px",
                borderRadius: 6,
                border: "1px solid var(--color-success)",
                background: "color-mix(in srgb, var(--color-success) 12%, transparent)",
                color: "var(--color-success)",
                fontSize: 13,
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              In Library →
            </button>
          ) : movie.excluded ? (
            <span
              style={{
                padding: "7px 16px",
                borderRadius: 6,
                background: "var(--color-bg-subtle)",
                color: "var(--color-text-muted)",
                fontSize: 13,
                fontWeight: 500,
              }}
            >
              Excluded
            </span>
          ) : (
            <button
              onClick={() => setShowAdd(true)}
              style={{
                padding: "7px 16px",
                borderRadius: 6,
                border: "none",
                background: "var(--color-accent)",
                color: "var(--color-accent-fg)",
                fontSize: 13,
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              + Add to Library
            </button>
          )}
        </div>
      </div>

      {/* Main layout — poster + content */}
      <div style={{ display: "flex", gap: 24, alignItems: "flex-start" }}>
        {/* Poster */}
        <div style={{ flexShrink: 0, width: 200 }}>
          <Poster
            src={movie.poster_path ? `https://image.tmdb.org/t/p/w342${movie.poster_path}` : undefined}
            title={movie.title}
            year={movie.year}
            loading="eager"
            style={{ boxShadow: "var(--shadow-modal)" }}
          />
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* Quick facts */}
          <div style={{ display: "flex", gap: 16, flexWrap: "wrap", marginBottom: 20 }}>
            {movie.rating > 0 && (
              <div style={{ display: "flex", alignItems: "center", gap: 5, fontSize: 13, color: "#fbbf24" }}>
                <Star size={14} fill="#fbbf24" stroke="none" />
                <span style={{ fontWeight: 600 }}>{movie.rating.toFixed(1)}</span>
              </div>
            )}
            {movie.runtime_minutes > 0 && (
              <div style={{ display: "flex", alignItems: "center", gap: 5, fontSize: 13, color: "var(--color-text-muted)" }}>
                <Clock size={13} />
                {formatRuntime(movie.runtime_minutes)}
              </div>
            )}
            {movie.release_date && (
              <div style={{ display: "flex", alignItems: "center", gap: 5, fontSize: 13, color: "var(--color-text-muted)" }}>
                <Calendar size={13} />
                {movie.release_date}
              </div>
            )}
            {movie.status && (
              <span style={{
                fontSize: 11,
                padding: "2px 8px",
                borderRadius: 4,
                background: movie.status === "released"
                  ? "color-mix(in srgb, var(--color-success) 15%, transparent)"
                  : "var(--color-bg-subtle)",
                color: movie.status === "released" ? "var(--color-success)" : "var(--color-text-muted)",
                fontWeight: 600,
                textTransform: "capitalize",
              }}>
                {movie.status}
              </span>
            )}
          </div>

          {/* Genres */}
          {movie.genres?.length > 0 && (
            <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginBottom: 16 }}>
              {movie.genres.map((g) => (
                <span
                  key={g}
                  style={{
                    fontSize: 11,
                    padding: "3px 10px",
                    borderRadius: 4,
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    color: "var(--color-text-secondary)",
                    fontWeight: 500,
                  }}
                >
                  {g}
                </span>
              ))}
            </div>
          )}

          {/* Overview */}
          {movie.overview && (
            <p style={{ margin: "0 0 24px", fontSize: 14, lineHeight: 1.7, color: "var(--color-text-secondary)" }}>
              {movie.overview}
            </p>
          )}

          {/* Crew */}
          {movie.crew?.length > 0 && (
            <div style={{ marginBottom: 24 }}>
              <h3 style={{ margin: "0 0 10px", fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
                Crew
              </h3>
              <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
                {movie.crew.map((c) => (
                  <div key={`${c.id}-${c.job}`} style={{ minWidth: 100 }}>
                    <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>{c.name}</div>
                    <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{c.job}</div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Cast */}
          {movie.cast?.length > 0 && (
            <div style={{ marginBottom: 24 }}>
              <h3 style={{ margin: "0 0 10px", fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
                Cast
              </h3>
              <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
                {movie.cast.map((c) => (
                  <Link
                    key={c.id}
                    to={`/people/${c.id}`}
                    style={{
                      textDecoration: "none",
                      color: "inherit",
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                      padding: "6px 10px",
                      borderRadius: 6,
                      background: "var(--color-bg-elevated)",
                      border: "1px solid var(--color-border-subtle)",
                    }}
                  >
                    {c.profile_path ? (
                      <img
                        src={`https://image.tmdb.org/t/p/w45${c.profile_path}`}
                        alt={c.name}
                        style={{ width: 28, height: 28, borderRadius: "50%", objectFit: "cover" }}
                      />
                    ) : (
                      <div style={{ width: 28, height: 28, borderRadius: "50%", background: "var(--color-bg-subtle)" }} />
                    )}
                    <div>
                      <div style={{ fontSize: 12, fontWeight: 500, color: "var(--color-text-primary)" }}>{c.name}</div>
                      <div style={{ fontSize: 10, color: "var(--color-text-muted)" }}>{c.character}</div>
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Recommendations */}
      {movie.recommendations?.length > 0 && (
        <div style={{ marginTop: 32 }}>
          <h3 style={{ margin: "0 0 14px", fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
            Recommended
          </h3>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(110px, 1fr))",
              gap: 14,
            }}
          >
            {movie.recommendations.map((rec) => (
              <Link
                key={rec.tmdb_id}
                to={`/discover/${rec.tmdb_id}`}
                style={{ textDecoration: "none", color: "inherit" }}
              >
                <Poster
                  src={rec.poster_path ? `https://image.tmdb.org/t/p/w185${rec.poster_path}` : undefined}
                  title={rec.title}
                  year={rec.year}
                />
                <div style={{ marginTop: 6 }}>
                  <span style={{ display: "block", fontSize: 11, fontWeight: 500, color: "var(--color-text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {rec.title}
                  </span>
                  {rec.year > 0 && (
                    <span style={{ fontSize: 10, color: "var(--color-text-muted)" }}>{rec.year}</span>
                  )}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Add modal */}
      {showAdd && movie && (
        <QuickAddModal
          movie={movie}
          onClose={() => setShowAdd(false)}
        />
      )}
    </div>
  );
}
