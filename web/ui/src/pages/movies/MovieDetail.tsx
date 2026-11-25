import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Poster } from "@/components/Poster";
import { useMovieCredits } from "@/api/movies";
import { toast } from "sonner";
import {
  useMovie,
  useDeleteMovie,
  useUpdateMovie,
  useRefreshMovie,
  useMovieFiles,
  useDeleteMovieFile,
  useAutoSearch,
  useEditions,
} from "@/api/movies";
import { useMovieHaulHistory, useReimportFromHaul } from "@/api/haul";
import { ManualSearchModal } from "@/components/ManualSearchModal";
import FilesTab from "./MovieFilesTab";
import HistoryTab from "./MovieHistoryTab";
import MatchTMDBBanner from "./MatchTMDBBanner";
import { actionBtn } from "./styles";

function formatRuntime(minutes: number): string {
  if (!minutes) return "—";
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}


// Rename confirmation modal

function HaulBanner({ movieId }: { movieId: string }) {
  const { data: records, isLoading } = useMovieHaulHistory(movieId);
  const reimport = useReimportFromHaul(movieId);

  // Only show banner for completed, still-present records
  const available = (records ?? []).filter(
    (r) => r.completed_at && !r.removed_at
  );

  if (isLoading || available.length === 0) return null;

  const best = available[0];
  const completedDate = best.completed_at
    ? new Date(best.completed_at).toLocaleDateString(undefined, { month: "short", day: "numeric" })
    : "";

  return (
    <div
      data-testid="haul-banner"
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 12,
        padding: "10px 16px",
        background: "color-mix(in srgb, var(--color-accent) 8%, var(--color-bg-surface))",
        border: "1px solid color-mix(in srgb, var(--color-accent) 25%, var(--color-border-subtle))",
        borderRadius: 8,
        marginBottom: 20,
      }}
    >
      <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
        Haul has{available.length > 1 ? ` ${available.length} downloads` : " 1 download"} for this movie
        {completedDate ? ` (${completedDate})` : ""}
        {" — "}
        <span style={{ color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)", fontSize: 11 }}>
          {best.name}
        </span>
      </span>
      <button
        onClick={() => reimport.mutate(best.info_hash)}
        disabled={reimport.isPending}
        style={{
          flexShrink: 0,
          background: "var(--color-accent)",
          color: "var(--color-accent-fg)",
          border: "none",
          borderRadius: 5,
          padding: "5px 14px",
          fontSize: 12,
          fontWeight: 500,
          cursor: reimport.isPending ? "not-allowed" : "pointer",
          whiteSpace: "nowrap",
          opacity: reimport.isPending ? 0.7 : 1,
        }}
      >
        {reimport.isPending ? "Importing…" : "Re-import"}
      </button>
    </div>
  );
}

// Delete confirm

function DeleteConfirmBar({ movieId, onCancel }: { movieId: string; onCancel: () => void }) {
  const del = useDeleteMovie();
  const navigate = useNavigate();

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 16px",
        background: "color-mix(in srgb, var(--color-danger) 8%, var(--color-bg-surface))",
        border: "1px solid color-mix(in srgb, var(--color-danger) 25%, var(--color-border-subtle))",
        borderRadius: 8,
        marginBottom: 20,
      }}
    >
      <span style={{ fontSize: 13, color: "var(--color-text-primary)", flex: 1 }}>
        Remove this movie from Prism? (Files on disk are not deleted.)
      </span>
      <button
        onClick={() => del.mutate(movieId, { onSuccess: () => navigate("/") })}
        disabled={del.isPending}
        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 15%, transparent)")}
      >
        {del.isPending ? "Deleting…" : "Yes, Remove"}
      </button>
      <button onClick={onCancel} style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}>
        Cancel
      </button>
    </div>
  );
}

// Page

type Tab = "overview" | "files" | "history";

export default function MovieDetail() {
  const { id } = useParams<{ id: string }>();
  const { data: movie, isLoading, error } = useMovie(id ?? "");
  const { data: credits } = useMovieCredits(id ?? "");
  const { data: files } = useMovieFiles(id ?? "");
  const deleteFile = useDeleteMovieFile(id ?? "");
  const updateMovie = useUpdateMovie();
  const refreshMovie = useRefreshMovie();
  const autoSearch = useAutoSearch();
  const { data: editions } = useEditions();

  const [tab, setTab] = useState<Tab>("overview");
  const [confirming, setConfirming] = useState(false);
  const [refreshed, setRefreshed] = useState(false);
  const [showManualSearch, setShowManualSearch] = useState(false);

  function handleMonitoredToggle() {
    if (!movie) return;
    updateMovie.mutate({
      id: movie.id,
      title: movie.title,
      monitored: !movie.monitored,
      library_id: movie.library_id,
      quality_profile_id: movie.quality_profile_id,
      minimum_availability: movie.minimum_availability,
      preferred_edition: movie.preferred_edition,
    });
  }

  function handleMinimumAvailabilityChange(value: string) {
    if (!movie) return;
    updateMovie.mutate({
      id: movie.id,
      title: movie.title,
      monitored: movie.monitored,
      library_id: movie.library_id,
      quality_profile_id: movie.quality_profile_id,
      minimum_availability: value,
      preferred_edition: movie.preferred_edition,
    });
  }

  function handlePreferredEditionChange(value: string) {
    if (!movie) return;
    updateMovie.mutate({
      id: movie.id,
      title: movie.title,
      monitored: movie.monitored,
      library_id: movie.library_id,
      quality_profile_id: movie.quality_profile_id,
      minimum_availability: movie.minimum_availability,
      preferred_edition: value || undefined,
    });
  }

  function handleRefresh() {
    if (!movie) return;
    refreshMovie.mutate(movie.id, {
      onSuccess: () => {
        setRefreshed(true);
        setTimeout(() => setRefreshed(false), 2000);
      },
    });
  }

  function handleAutoSearch() {
    if (!movie) return;
    autoSearch.mutate(movie.id, {
      onSuccess: (data) => {
        switch (data.result) {
          case "grabbed":
            toast.success(`Grabbed: ${data.grab?.release_title ?? "release"}`);
            break;
          case "no_match":
            toast.info(data.reason ?? "No suitable release found");
            break;
          case "skipped":
            toast.info(data.reason ?? "Already downloading");
            break;
        }
      },
      onError: (err) => toast.error((err as Error).message),
    });
  }

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: "flex", flexDirection: "column", gap: 20 }}>
        <div className="skeleton" style={{ height: 24, width: 200, borderRadius: 4 }} />
        <div style={{ display: "flex", gap: 24 }}>
          <div className="skeleton" style={{ width: 200, height: 300, borderRadius: 8 }} />
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
        <Link to="/" style={{ fontSize: 13, color: "var(--color-accent)", textDecoration: "none" }}>
          ← Movies
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
      {movie.fanart_url && (
        <div
          data-testid="backdrop"
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: 280,
            backgroundImage: `url(${movie.fanart_url})`,
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
        to="/"
        style={{ fontSize: 13, color: "var(--color-text-muted)", textDecoration: "none", display: "inline-block", marginBottom: 20 }}
        onMouseEnter={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-primary)"; }}
        onMouseLeave={(e) => { (e.currentTarget as HTMLAnchorElement).style.color = "var(--color-text-muted)"; }}
      >
        ← Movies
      </Link>

      {/* Unmatched banner */}
      {movie.tmdb_id === 0 && <MatchTMDBBanner movieId={movie.id} />}

      {/* Haul re-import banner — only when no file is on disk */}
      {(!files || files.length === 0) && <HaulBanner movieId={movie.id} />}

      {/* Delete confirmation bar */}
      {confirming && (
        <DeleteConfirmBar movieId={movie.id} onCancel={() => setConfirming(false)} />
      )}

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
        <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
          <button
            onClick={handleMonitoredToggle}
            disabled={updateMovie.isPending}
            style={{
              ...actionBtn(
                movie.monitored ? "var(--color-success)" : "var(--color-text-muted)",
                movie.monitored
                  ? "color-mix(in srgb, var(--color-success) 12%, transparent)"
                  : "var(--color-bg-elevated)"
              ),
              borderColor: movie.monitored ? "var(--color-success)" : "var(--color-border-default)",
            }}
          >
            {movie.monitored ? "Monitored" : "Unmonitored"}
          </button>
          {refreshed ? (
            <span style={{ fontSize: 12, color: "var(--color-success)", display: "flex", alignItems: "center", padding: "0 4px" }}>
              Queued ✓
            </span>
          ) : (
            <button
              onClick={handleRefresh}
              disabled={refreshMovie.isPending}
              style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
            >
              {refreshMovie.isPending ? "Refreshing…" : "Refresh Metadata"}
            </button>
          )}
          <button
            onClick={() => setShowManualSearch(true)}
            style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
          >
            Interactive Search
          </button>
          <button
            onClick={handleAutoSearch}
            disabled={autoSearch.isPending}
            style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
          >
            {autoSearch.isPending ? "Searching…" : "Auto Search"}
          </button>
          <button
            onClick={() => setConfirming((v) => !v)}
            style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 10%, transparent)")}
          >
            Delete
          </button>
        </div>
      </div>

      {/* Main layout — poster centered against content */}
      <div style={{ display: "flex", gap: 24, alignItems: "center" }}>
        {/* Poster */}
        <div style={{ flexShrink: 0, width: 180 }}>
          <Poster
            src={movie.poster_url}
            title={movie.title}
            year={movie.year}
            loading="eager"
            style={{ boxShadow: "var(--shadow-modal)" }}
          />
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* Tabs */}
          <div style={{ display: "flex", gap: 0, borderBottom: "1px solid var(--color-border-subtle)", marginBottom: 20 }}>
            {(["overview", "files", "history"] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                style={{
                  background: "none",
                  border: "none",
                  borderBottom: `2px solid ${tab === t ? "var(--color-accent)" : "transparent"}`,
                  padding: "10px 16px",
                  fontSize: 13,
                  fontWeight: tab === t ? 600 : 400,
                  color: tab === t ? "var(--color-accent)" : "var(--color-text-muted)",
                  cursor: "pointer",
                  textTransform: "capitalize",
                  marginBottom: -1,
                  transition: "color 0.1s, border-color 0.1s",
                }}
              >
                {t}
              </button>
            ))}
          </div>

          {/* Tab content */}
          {tab === "overview" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              {/* Quick facts */}
              <div style={{ display: "flex", gap: 16, flexWrap: "wrap", alignItems: "flex-start" }}>
                {[
                  { label: "Runtime", value: formatRuntime(movie.runtime_minutes) },
                  { label: "Status", value: movie.status },
                  { label: "TMDB", value: String(movie.tmdb_id) },
                  ...(movie.imdb_id ? [{ label: "IMDB", value: movie.imdb_id }] : []),
                ].map(({ label, value }) => (
                  <div
                    key={label}
                    style={{
                      background: "var(--color-bg-elevated)",
                      border: "1px solid var(--color-border-subtle)",
                      borderRadius: 6,
                      padding: "8px 14px",
                    }}
                  >
                    <div style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 2 }}>
                      {label}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--color-text-primary)" }}>{value}</div>
                  </div>
                ))}

                {/* Minimum availability selector */}
                <div
                  style={{
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    borderRadius: 6,
                    padding: "8px 14px",
                  }}
                >
                  <div style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 4 }}>
                    Min. Availability
                  </div>
                  <select
                    value={movie.minimum_availability || "released"}
                    onChange={(e) => handleMinimumAvailabilityChange(e.target.value)}
                    disabled={updateMovie.isPending}
                    style={{
                      fontSize: 13,
                      color: "var(--color-text-primary)",
                      background: "transparent",
                      border: "none",
                      padding: 0,
                      cursor: "pointer",
                      outline: "none",
                    }}
                  >
                    <option value="announced">Announced</option>
                    <option value="in_cinemas">In Cinemas</option>
                    <option value="released">Released</option>
                    <option value="tba">TBA</option>
                  </select>
                </div>

                {/* Preferred edition selector */}
                <div
                  style={{
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    borderRadius: 6,
                    padding: "8px 14px",
                  }}
                >
                  <div style={{ fontSize: 10, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 4 }}>
                    Preferred Edition
                  </div>
                  <select
                    value={movie.preferred_edition || ""}
                    onChange={(e) => handlePreferredEditionChange(e.target.value)}
                    disabled={updateMovie.isPending}
                    style={{
                      fontSize: 13,
                      color: movie.preferred_edition ? "var(--color-text-primary)" : "var(--color-text-muted)",
                      background: "transparent",
                      border: "none",
                      padding: 0,
                      cursor: "pointer",
                      outline: "none",
                    }}
                  >
                    <option value="">Any</option>
                    {editions?.map((ed) => (
                      <option key={ed} value={ed}>{ed}</option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Genres */}
              {movie.genres?.length > 0 && (
                <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                  {movie.genres.map((g) => (
                    <span
                      key={g}
                      style={{
                        padding: "3px 10px",
                        borderRadius: 4,
                        fontSize: 12,
                        background: "var(--color-bg-elevated)",
                        border: "1px solid var(--color-border-subtle)",
                        color: "var(--color-text-secondary)",
                      }}
                    >
                      {g}
                    </span>
                  ))}
                </div>
              )}

              {/* Overview */}
              {movie.overview && (
                <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", lineHeight: 1.65 }}>
                  {movie.overview}
                </p>
              )}

              {/* File status card */}
              {files && files.length > 0 && files[0].missing ? (
                <div
                  data-testid="file-status-card"
                  style={{
                    padding: "10px 14px",
                    borderRadius: 8,
                    background: "color-mix(in srgb, var(--color-danger) 8%, transparent)",
                    border: "1px solid color-mix(in srgb, var(--color-danger) 25%, transparent)",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, fontSize: 13 }}>
                    <span style={{ color: "var(--color-danger)", fontWeight: 600 }}>File not found</span>
                    <button
                      onClick={() => deleteFile.mutate({ fileId: files[0].id, deleteFromDisk: false })}
                      disabled={deleteFile.isPending}
                      style={{
                        background: "var(--color-bg-elevated)",
                        border: "1px solid var(--color-border-default)",
                        borderRadius: 5,
                        padding: "4px 10px",
                        fontSize: 12,
                        color: "var(--color-text-secondary)",
                        cursor: deleteFile.isPending ? "not-allowed" : "pointer",
                      }}
                    >
                      {deleteFile.isPending ? "Removing…" : "Remove record"}
                    </button>
                  </div>
                  <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 4 }}>
                    Prism imported this file{files[0].imported_at && <span> on {new Date(files[0].imported_at).toLocaleDateString()}</span>}, but it isn't there now:
                  </div>
                  <div
                    title={files[0].path}
                    style={{ fontSize: 11, fontFamily: "var(--font-family-mono)", color: "var(--color-text-secondary)", marginTop: 4, overflowWrap: "anywhere" }}
                  >
                    {files[0].path}
                  </div>
                </div>
              ) : files && files.length > 0 ? (
                <div
                  data-testid="file-status-card"
                  style={{
                    padding: "10px 14px",
                    borderRadius: 8,
                    background: "color-mix(in srgb, var(--color-success) 8%, transparent)",
                    border: "1px solid color-mix(in srgb, var(--color-success) 20%, transparent)",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13 }}>
                    <span style={{ color: "var(--color-success)", fontWeight: 600 }}>On Disk</span>
                    <span style={{ color: "var(--color-text-secondary)" }}>
                      {files[0].quality?.name || "Unknown quality"}
                    </span>
                  </div>
                  <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 4 }}>
                    {files[0].edition && <span>{files[0].edition} &middot; </span>}
                    {(files[0].size_bytes / (1024 * 1024 * 1024)).toFixed(1)} GB
                    {files[0].imported_at && <span> &middot; Imported {new Date(files[0].imported_at).toLocaleDateString()}</span>}
                  </div>
                  {files[0].mediainfo && files[0].quality &&
                    files[0].mediainfo.codec && files[0].quality.codec &&
                    files[0].mediainfo.codec !== files[0].quality.codec && (
                    <div style={{ fontSize: 12, color: "var(--color-warning)", marginTop: 4 }}>
                      Mismatch: actual codec is {files[0].mediainfo.codec}
                    </div>
                  )}
                </div>
              ) : (
                <div
                  data-testid="file-status-card"
                  style={{
                    padding: "10px 14px",
                    borderRadius: 8,
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    fontSize: 13,
                    color: "var(--color-text-muted)",
                  }}
                >
                  {movie.monitored ? "Missing — Monitored, searching" : "Missing — Not monitored"}
                </div>
              )}

              {/* Watch info */}
              {movie.watch_status && (
                <div
                  data-testid="watch-info"
                  style={{
                    padding: "8px 14px",
                    borderRadius: 8,
                    background: "var(--color-bg-elevated)",
                    border: "1px solid var(--color-border-subtle)",
                    fontSize: 13,
                    color: movie.watch_status.watched ? "var(--color-text-primary)" : "var(--color-text-muted)",
                  }}
                >
                  {movie.watch_status.watched ? (
                    <>
                      Watched {movie.watch_status.play_count} time{movie.watch_status.play_count !== 1 && "s"}
                      {movie.watch_status.last_watched_at && (
                        <span style={{ color: "var(--color-text-muted)" }}>
                          {" "}&middot; Last: {new Date(movie.watch_status.last_watched_at).toLocaleDateString()}
                        </span>
                      )}
                    </>
                  ) : (
                    "Not watched"
                  )}
                </div>
              )}

              {/* File path */}
              {movie.path && (
                <div>
                  <div style={{ fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 4 }}>
                    File Path
                  </div>
                  <code
                    style={{
                      display: "block",
                      padding: "8px 12px",
                      borderRadius: 6,
                      background: "var(--color-bg-elevated)",
                      border: "1px solid var(--color-border-subtle)",
                      fontSize: 12,
                      fontFamily: "var(--font-family-mono)",
                      color: "var(--color-text-secondary)",
                      overflowX: "auto",
                    }}
                  >
                    {movie.path}
                  </code>
                </div>
              )}

            </div>
          )}

          {tab === "files" && <FilesTab movieId={movie.id} />}
          {tab === "history" && <HistoryTab movieId={movie.id} />}
        </div>
      </div>

      {/* Credits & recommendations — full width below the two-column layout */}
      {tab === "overview" && (
        <div style={{ display: "flex", flexDirection: "column", gap: 20, marginTop: 24 }}>
          {/* Director / Writer */}
          {credits && credits.crew.length > 0 && (
            <div data-testid="crew-row" style={{ fontSize: 13, color: "var(--color-text-secondary)" }}>
              {credits.crew
                .filter((c) => c.job === "Director")
                .map((c, i) => (
                  <span key={c.id}>
                    {i > 0 && ", "}
                    Directed by <span style={{ color: "var(--color-text-primary)", fontWeight: 500 }}>{c.name}</span>
                  </span>
                ))}
              {credits.crew.some((c) => c.job === "Director") &&
                credits.crew.some((c) => c.job === "Screenplay" || c.job === "Writer") &&
                <span> &middot; </span>}
              {credits.crew
                .filter((c) => c.job === "Screenplay" || c.job === "Writer")
                .slice(0, 2)
                .map((c, i) => (
                  <span key={c.id}>
                    {i > 0 && ", "}
                    {i === 0 && "Written by "}
                    <span style={{ color: "var(--color-text-primary)", fontWeight: 500 }}>{c.name}</span>
                  </span>
                ))}
            </div>
          )}

          {/* Cast strip */}
          {credits && credits.cast.length > 0 && (
            <div data-testid="cast-strip">
              <div style={{ fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 8 }}>
                Cast
              </div>
              <div
                style={{ display: "flex", gap: 12, overflowX: "auto", paddingBottom: 4 }}
                onWheel={(e) => {
                  if (e.deltaY !== 0) {
                    e.currentTarget.scrollLeft += e.deltaY;
                    e.preventDefault();
                  }
                }}
              >
                {credits.cast.map((c) => (
                  <Link key={c.id} to={`/people/${c.id}`} style={{ textDecoration: "none", color: "inherit", display: "flex", flexDirection: "column", alignItems: "center", gap: 4, minWidth: 72, maxWidth: 72 }}>
                    {c.profile_path ? (
                      <img
                        src={`https://image.tmdb.org/t/p/w185${c.profile_path}`}
                        alt={c.name}
                        style={{ width: 48, height: 48, borderRadius: "50%", objectFit: "cover" }}
                      />
                    ) : (
                      <div
                        style={{
                          width: 48,
                          height: 48,
                          borderRadius: "50%",
                          background: "var(--color-bg-subtle)",
                          border: "1px solid var(--color-border-subtle)",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontSize: 14,
                          fontWeight: 600,
                          color: "var(--color-text-muted)",
                        }}
                      >
                        {c.name.charAt(0)}
                      </div>
                    )}
                    <span style={{ fontSize: 10, fontWeight: 500, color: "var(--color-text-primary)", textAlign: "center", lineHeight: 1.2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", width: "100%" }}>
                      {c.name}
                    </span>
                    <span style={{ fontSize: 9, color: "var(--color-text-muted)", textAlign: "center", lineHeight: 1.2, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", width: "100%" }}>
                      {c.character}
                    </span>
                  </Link>
                ))}
              </div>
            </div>
          )}

          {/* Similar movies */}
          {credits && credits.recommendations.length > 0 && (
            <div data-testid="similar-movies">
              <div style={{ fontSize: 11, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--color-text-muted)", marginBottom: 8 }}>
                You might also like
              </div>
              <div
                style={{ display: "flex", gap: 12, overflowX: "auto", paddingBottom: 4 }}
                onWheel={(e) => {
                  if (e.deltaY !== 0) {
                    e.currentTarget.scrollLeft += e.deltaY;
                    e.preventDefault();
                  }
                }}
              >
                {credits.recommendations.map((r) => (
                  <Link
                    key={r.tmdb_id}
                    to={r.in_library && r.movie_id ? `/movies/${r.movie_id}` : `/discover/${r.tmdb_id}`}
                    style={{ textDecoration: "none", minWidth: 100, maxWidth: 100 }}
                  >
                    <Poster
                      src={r.poster_path ? `https://image.tmdb.org/t/p/w185${r.poster_path}` : undefined}
                      title={r.title}
                      year={r.year}
                    />
                    <div style={{ marginTop: 6 }}>
                      <span style={{ fontSize: 11, fontWeight: 500, color: "var(--color-text-primary)", display: "block", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                        {r.title}
                      </span>
                      <span style={{ fontSize: 10, color: "var(--color-text-muted)" }}>
                        {r.year > 0 && r.year}
                      </span>
                      {r.in_library && (
                        <span style={{ fontSize: 9, padding: "1px 5px", borderRadius: 3, background: "color-mix(in srgb, var(--color-success) 15%, transparent)", color: "var(--color-success)", fontWeight: 600, marginLeft: 4 }}>
                          In Library
                        </span>
                      )}
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {showManualSearch && (
        <ManualSearchModal
          movieId={movie.id}
          movieTitle={movie.title}
          onClose={() => setShowManualSearch(false)}
        />
      )}

    </div>
  );
}
