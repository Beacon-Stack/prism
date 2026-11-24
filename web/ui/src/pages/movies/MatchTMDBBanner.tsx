import { useState, useEffect } from "react";
import { useMatchMovie, useLookupMovies, useMovieSuggestions } from "@/api/movies";
import type { TMDBResult } from "@/types";
import { actionBtn } from "./styles";

export default function MatchTMDBBanner({ movieId }: { movieId: string }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [manualResults, setManualResults] = useState<TMDBResult[] | null>(null);

  const suggestions = useMovieSuggestions(movieId, open);
  const lookup = useLookupMovies();
  const match = useMatchMovie();

  // When the panel opens, pre-fill the query with the parsed title.
  useEffect(() => {
    if (open && suggestions.data) {
      setQuery(suggestions.data.parsed_title);
    }
  }, [open, suggestions.data]);

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    const r = await lookup.mutateAsync({ query: query.trim() });
    setManualResults(r.slice(0, 8));
  }

  function handleClose() {
    setOpen(false);
    setManualResults(null);
    setQuery("");
  }

  // Show manual results if the user has searched; otherwise show auto-suggestions.
  const displayResults: TMDBResult[] = manualResults ?? suggestions.data?.results ?? [];

  function handleSelect(r: TMDBResult) {
    match.mutate({ id: movieId, tmdb_id: r.tmdb_id });
  }

  return (
    <div
      style={{
        background: "color-mix(in srgb, var(--color-warning) 8%, var(--color-bg-surface))",
        border: "1px solid color-mix(in srgb, var(--color-warning) 35%, var(--color-border-subtle))",
        borderRadius: 8,
        padding: "12px 16px",
        marginBottom: 20,
        display: "flex",
        flexDirection: "column",
        gap: 10,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
        <span style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>
          This movie has no TMDB match. Artwork and metadata won't be available until it's matched.
        </span>
        {!open && (
          <button
            onClick={() => setOpen(true)}
            style={{
              flexShrink: 0,
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 5,
              padding: "5px 14px",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
              whiteSpace: "nowrap",
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            Match to TMDB
          </button>
        )}
      </div>

      {open && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {suggestions.data && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>
              Parsed from filename: <strong>{suggestions.data.parsed_title}</strong>
              {suggestions.data.parsed_year > 0 && <> ({suggestions.data.parsed_year})</>}
            </p>
          )}

          <form onSubmit={handleSearch} style={{ display: "flex", gap: 6 }}>
            <input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.currentTarget.value)}
              placeholder="Search TMDB by title…"
              style={{
                flex: 1,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 5,
                padding: "6px 10px",
                fontSize: 13,
                color: "var(--color-text-primary)",
                outline: "none",
              }}
              onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-accent)"; }}
              onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = "var(--color-border-default)"; }}
            />
            <button
              type="submit"
              disabled={lookup.isPending}
              style={actionBtn("var(--color-accent-fg)", "var(--color-accent)")}
            >
              {lookup.isPending ? "…" : "Search"}
            </button>
            <button
              type="button"
              onClick={handleClose}
              style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
            >
              Cancel
            </button>
          </form>

          {suggestions.isLoading && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>Looking up suggestions…</p>
          )}

          {displayResults.length > 0 && (
            <div
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                overflow: "hidden",
              }}
            >
              {displayResults.map((r) => (
                <button
                  key={r.tmdb_id}
                  onClick={() => handleSelect(r)}
                  disabled={match.isPending}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    width: "100%",
                    textAlign: "left",
                    padding: "8px 12px",
                    background: "none",
                    border: "none",
                    borderBottom: "1px solid var(--color-border-subtle)",
                    cursor: match.isPending ? "not-allowed" : "pointer",
                  }}
                  onMouseEnter={(e) => {
                    if (!match.isPending)
                      (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.background = "none";
                  }}
                >
                  {r.poster_path && (
                    <img
                      src={`https://image.tmdb.org/t/p/w92${r.poster_path}`}
                      alt=""
                      style={{ width: 32, height: 48, objectFit: "cover", borderRadius: 3, flexShrink: 0 }}
                    />
                  )}
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
                      {r.title}
                    </div>
                    {r.year > 0 && (
                      <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 1 }}>
                        {r.year}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}

          {match.isPending && (
            <p style={{ margin: 0, fontSize: 12, color: "var(--color-text-muted)" }}>Matching…</p>
          )}
        </div>
      )}
    </div>
  );
}
