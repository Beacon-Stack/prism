import { useState } from "react";
import { useAddMovie, useLookupMovies, useEditions } from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import Modal from "@beacon-shared/Modal";
import type { TMDBResult } from "@/types";
import { inputStyle as dialogInputStyle, labelStyle as dialogLabelStyle } from "@/lib/styles";

function focusBorder(e: React.FocusEvent<HTMLElement>) {
  (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
}
function blurBorder(e: React.FocusEvent<HTMLElement>) {
  (e.currentTarget as HTMLElement).style.borderColor =
    "var(--color-border-default)";
}

function SearchResultRow({
  result,
  onSelect,
}: {
  result: TMDBResult;
  onSelect: (r: TMDBResult) => void;
}) {
  const posterSrc = result.poster_path
    ? `https://image.tmdb.org/t/p/w92${result.poster_path}`
    : null;

  return (
    <button
      onClick={() => onSelect(result)}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 14px",
        background: "none",
        border: "none",
        borderBottom: "1px solid var(--color-border-subtle)",
        cursor: "pointer",
        textAlign: "left",
        width: "100%",
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background =
          "var(--color-bg-surface)";
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLButtonElement).style.background = "none";
      }}
    >
      {posterSrc ? (
        <img
          src={posterSrc}
          alt={result.title}
          style={{
            width: 36,
            height: 54,
            objectFit: "cover",
            borderRadius: 3,
            flexShrink: 0,
          }}
        />
      ) : (
        <div
          style={{
            width: 36,
            height: 54,
            borderRadius: 3,
            background: "var(--color-bg-subtle)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
          }}
        >
          {result.title}
          {result.year > 0 && (
            <span
              style={{
                marginLeft: 6,
                fontSize: 12,
                color: "var(--color-text-muted)",
                fontWeight: 400,
              }}
            >
              {result.year}
            </span>
          )}
        </div>
        {result.overview && (
          <div
            style={{
              fontSize: 12,
              color: "var(--color-text-muted)",
              marginTop: 2,
              overflow: "hidden",
              display: "-webkit-box",
              WebkitLineClamp: 2,
              WebkitBoxOrient: "vertical",
            }}
          >
            {result.overview}
          </div>
        )}
      </div>
      <span
        style={{
          fontSize: 11,
          color: "var(--color-accent)",
          fontWeight: 600,
          flexShrink: 0,
        }}
      >
        Select →
      </span>
    </button>
  );
}

function SearchStep({
  onSelect,
  onClose,
}: {
  onSelect: (result: TMDBResult) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const lookup = useLookupMovies();

  function handleSearch() {
    if (!query.trim()) return;
    lookup.mutate({ query: query.trim() });
  }

  return (
    <>
      <div style={{ display: "flex", gap: 8 }}>
        <input
          style={{ ...dialogInputStyle, flex: 1 }}
          value={query}
          onChange={(e) => setQuery(e.currentTarget.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSearch()}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="Search TMDB — title or year…"
          autoFocus
        />
        <button
          onClick={handleSearch}
          disabled={lookup.isPending || !query.trim()}
          style={{
            background:
              lookup.isPending || !query.trim()
                ? "var(--color-bg-subtle)"
                : "var(--color-accent)",
            color:
              lookup.isPending || !query.trim()
                ? "var(--color-text-muted)"
                : "var(--color-accent-fg)",
            border: "none",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
            cursor:
              lookup.isPending || !query.trim() ? "not-allowed" : "pointer",
            whiteSpace: "nowrap",
          }}
        >
          {lookup.isPending ? "Searching…" : "Search"}
        </button>
      </div>

      {lookup.isError && (
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>
          {lookup.error.message}
        </p>
      )}

      {lookup.data && lookup.data.length === 0 && (
        <p
          style={{
            margin: 0,
            fontSize: 13,
            color: "var(--color-text-muted)",
            textAlign: "center",
            padding: "24px 0",
          }}
        >
          No results found. Try a different search.
        </p>
      )}

      {lookup.data && lookup.data.length > 0 && (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 1,
            maxHeight: 360,
            overflowY: "auto",
            borderRadius: 6,
            border: "1px solid var(--color-border-subtle)",
            background: "var(--color-bg-elevated)",
          }}
        >
          {lookup.data.map((result) => (
            <SearchResultRow
              key={result.tmdb_id}
              result={result}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}

      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <button
          onClick={onClose}
          style={{
            background: "none",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            color: "var(--color-text-secondary)",
            cursor: "pointer",
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-bg-elevated)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = "none";
          }}
        >
          Cancel
        </button>
      </div>
    </>
  );
}

function ConfigureStep({
  result,
  onBack,
  onSuccess,
  onClose,
}: {
  result: TMDBResult;
  onBack: () => void;
  onSuccess: () => void;
  onClose: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const { data: editions } = useEditions();
  const addMovie = useAddMovie();

  const [libraryId, setLibraryId] = useState("");
  const [profileId, setProfileId] = useState("");

  // Sync defaults once data loads — useState initializer only runs on mount,
  // before the queries have resolved.
  const firstLibrary = libraries?.[0]?.id ?? "";
  const firstProfile = profiles?.[0]?.id ?? "";
  if (!libraryId && firstLibrary) setLibraryId(firstLibrary);
  if (!profileId && firstProfile) setProfileId(firstProfile);
  const [preferredEdition, setPreferredEdition] = useState("");
  const [monitored, setMonitored] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const posterSrc = result.poster_path
    ? `https://image.tmdb.org/t/p/w185${result.poster_path}`
    : null;

  function handleAdd() {
    if (!libraryId) {
      setError("Select a library.");
      return;
    }
    if (!profileId) {
      setError("Select a quality profile.");
      return;
    }
    addMovie.mutate(
      {
        tmdb_id: result.tmdb_id,
        library_id: libraryId,
        quality_profile_id: profileId,
        monitored,
        ...(preferredEdition ? { preferred_edition: preferredEdition } : {}),
      },
      {
        onSuccess,
        onError: (e) => {
          const msg = (e as Error).message;
          if (msg.includes("409") || msg.toLowerCase().includes("already")) {
            setError("This movie is already in the library.");
          } else {
            setError(msg);
          }
        },
      }
    );
  }

  return (
    <>
      {/* Movie preview */}
      <div
        style={{
          display: "flex",
          gap: 16,
          padding: 16,
          background: "var(--color-bg-elevated)",
          borderRadius: 8,
          border: "1px solid var(--color-border-subtle)",
        }}
      >
        {posterSrc ? (
          <img
            src={posterSrc}
            alt={result.title}
            style={{
              width: 64,
              height: 96,
              objectFit: "cover",
              borderRadius: 4,
              flexShrink: 0,
            }}
          />
        ) : (
          <div
            style={{
              width: 64,
              height: 96,
              background: "var(--color-bg-subtle)",
              borderRadius: 4,
              flexShrink: 0,
            }}
          />
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <p
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 600,
              color: "var(--color-text-primary)",
            }}
          >
            {result.title}
          </p>
          {result.year > 0 && (
            <p
              style={{
                margin: "2px 0 0",
                fontSize: 13,
                color: "var(--color-text-muted)",
              }}
            >
              {result.year}
            </p>
          )}
          {result.overview && (
            <p
              style={{
                margin: "8px 0 0",
                fontSize: 12,
                color: "var(--color-text-secondary)",
                lineHeight: 1.5,
                overflow: "hidden",
                display: "-webkit-box",
                WebkitLineClamp: 3,
                WebkitBoxOrient: "vertical",
              }}
            >
              {result.overview}
            </p>
          )}
        </div>
      </div>

      {/* Config fields */}
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <div>
          <label style={dialogLabelStyle}>Library *</label>
          <select
            style={{ ...dialogInputStyle, cursor: "pointer" }}
            value={libraryId}
            onChange={(e) => {
              setLibraryId(e.currentTarget.value);
              setError(null);
            }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          >
            {!libraries?.length && (
              <option value="">No libraries configured</option>
            )}
            {libraries?.map((lib) => (
              <option key={lib.id} value={lib.id}>
                {lib.name} — {lib.root_path}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label style={dialogLabelStyle}>Quality Profile *</label>
          <select
            style={{ ...dialogInputStyle, cursor: "pointer" }}
            value={profileId}
            onChange={(e) => {
              setProfileId(e.currentTarget.value);
              setError(null);
            }}
            onFocus={focusBorder}
            onBlur={blurBorder}
          >
            {!profiles?.length && (
              <option value="">No profiles configured</option>
            )}
            {profiles?.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label style={dialogLabelStyle}>Preferred Edition</label>
          <select
            style={{ ...dialogInputStyle, cursor: "pointer" }}
            value={preferredEdition}
            onChange={(e) => setPreferredEdition(e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
          >
            <option value="">Any (no preference)</option>
            {editions?.map((ed) => (
              <option key={ed} value={ed}>{ed}</option>
            ))}
          </select>
        </div>

        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            cursor: "pointer",
            userSelect: "none",
          }}
        >
          <input
            type="checkbox"
            checked={monitored}
            onChange={(e) => setMonitored(e.currentTarget.checked)}
            style={{
              width: 16,
              height: 16,
              cursor: "pointer",
              accentColor: "var(--color-accent)",
            }}
          />
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            Monitor for releases
          </span>
        </label>
      </div>

      {error && (
        <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>
          {error}
        </p>
      )}

      <div style={{ display: "flex", justifyContent: "space-between" }}>
        <button
          onClick={onBack}
          style={{
            background: "none",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            padding: "8px 16px",
            fontSize: 13,
            color: "var(--color-text-secondary)",
            cursor: "pointer",
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background =
              "var(--color-bg-elevated)";
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = "none";
          }}
        >
          ← Back
        </button>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-bg-elevated)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = "none";
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleAdd}
            disabled={addMovie.isPending}
            style={{
              background: addMovie.isPending
                ? "var(--color-bg-subtle)"
                : "var(--color-accent)",
              color: addMovie.isPending
                ? "var(--color-text-muted)"
                : "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: addMovie.isPending ? "not-allowed" : "pointer",
            }}
            onMouseEnter={(e) => {
              if (!addMovie.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (!addMovie.isPending)
                (e.currentTarget as HTMLButtonElement).style.background =
                  "var(--color-accent)";
            }}
          >
            {addMovie.isPending ? "Adding…" : "Add Movie"}
          </button>
        </div>
      </div>
    </>
  );
}

export default function AddMovieDialog({ onClose }: { onClose: () => void }) {
  const [selected, setSelected] = useState<TMDBResult | null>(null);

  return (
    <Modal onClose={onClose} width={580} innerStyle={{ padding: 24, gap: 20, overflowY: "auto" }}>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <h2
            style={{
              margin: 0,
              fontSize: 16,
              fontWeight: 600,
              color: "var(--color-text-primary)",
            }}
          >
            {selected ? "Configure Movie" : "Add Movie"}
          </h2>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              fontSize: 18,
              lineHeight: 1,
              padding: "4px 6px",
              borderRadius: 4,
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color =
                "var(--color-text-primary)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color =
                "var(--color-text-muted)";
            }}
          >
            ✕
          </button>
        </div>

        {selected ? (
          <ConfigureStep
            result={selected}
            onBack={() => setSelected(null)}
            onSuccess={onClose}
            onClose={onClose}
          />
        ) : (
          <SearchStep onSelect={setSelected} onClose={onClose} />
        )}
    </Modal>
  );
}
