import { useNavigate, Link } from "react-router-dom";
import { Star } from "lucide-react";
import type { DiscoverResult } from "@/api/discover";
import { Poster } from "@/components/Poster";

/** One movie on Discover: poster, rating, title, and Add or In Library. */
export default function DiscoverCard({
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
