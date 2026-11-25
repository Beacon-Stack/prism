import { AVAILABILITY, AVAILABILITY_ORDER, availabilityLabel, availabilityOf, type Availability } from "@/lib/availability";
import { useState, useMemo, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { LayoutGrid, List } from "lucide-react";
import { Poster } from "@/components/Poster";
import { toast } from "sonner";
import { useAllMovies } from "@/api/movies";
import { useLibraries } from "@/api/libraries";
import { useQualityMovies } from "@/api/stats";
import { useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/api/client";
import type { Movie } from "@/types";
import AddMovieDialog from "./AddMovieDialog";
import BulkEditModal from "./BulkEditModal";

// Shared styles

const ctrlStyle: React.CSSProperties = {
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  padding: "7px 11px",
  fontSize: 13,
  color: "var(--color-text-primary)",
  outline: "none",
};

const thStyle: React.CSSProperties = {
  textAlign: "left",
  padding: "10px 16px",
  fontSize: 11,
  fontWeight: 600,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  color: "var(--color-text-muted)",
  whiteSpace: "nowrap",
};

// Types

type MonitoredFilter = "all" | "monitored" | "unmonitored";
type OnDiskFilter = "all" | "on_disk" | "missing";
type SortField = "title" | "year" | "added_at";
type SortDir = "asc" | "desc";
type ViewMode = "grid" | "list";

// StatusBadge

function StatusBadge({ movie }: { movie: Movie }) {
  const state = availabilityOf(movie);
  const { color, hint } = AVAILABILITY[state];
  return (
    <span
      title={hint}
      style={{
        display: "inline-block",
        padding: "1px 6px",
        borderRadius: 3,
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: "0.04em",
        background: `color-mix(in srgb, ${color} 12%, transparent)`,
        color,
        whiteSpace: "nowrap",
      }}
    >
      {availabilityLabel(state, movie.download_progress)}
    </span>
  );
}

// PosterCard

function PosterCard({
  movie,
  selectionMode,
  isSelected,
  onToggle,
}: {
  movie: Movie;
  selectionMode: boolean;
  isSelected: boolean;
  onToggle: (id: string) => void;
}) {
  const navigate = useNavigate();
  const onDisk = !!movie.path;

  return (
    <div
      onClick={() =>
        selectionMode ? onToggle(movie.id) : navigate(`/movies/${movie.id}`)
      }
      style={{ cursor: "pointer" }}
    >
      {/* Poster */}
      <div
        style={{
          paddingBottom: "150%",
          position: "relative",
          borderRadius: 8,
          overflow: "hidden",
        }}
      >
        <div style={{ position: "absolute", inset: 0 }}>
          <Poster
            src={movie.poster_url}
            title={movie.title}
            year={movie.year}
            style={{ borderRadius: 0, width: "100%", height: "100%" }}
            imgStyle={{ borderRadius: 0 }}
          />
        </div>

        {/* Selection overlay */}
        {selectionMode && (
          <div
            style={{
              position: "absolute",
              inset: 0,
              background: isSelected
                ? "color-mix(in srgb, var(--color-accent) 35%, transparent)"
                : undefined,
              border: isSelected
                ? "2px solid var(--color-accent)"
                : "2px solid transparent",
              borderRadius: 8,
              display: "flex",
              alignItems: "flex-start",
              padding: 6,
              pointerEvents: "none",
            }}
          >
            <span
              style={{
                display: "inline-flex",
                width: 18,
                height: 18,
                borderRadius: 4,
                background: isSelected ? "var(--color-accent)" : "rgba(0,0,0,0.5)",
                border: isSelected ? "none" : "2px solid rgba(255,255,255,0.6)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 11,
                color: "white",
                fontWeight: 700,
              }}
            >
              {isSelected ? "✓" : ""}
            </span>
          </div>
        )}

        {/* Corner badge — rendered after overlay so it sits on top of it.
            pointerEvents:none so it doesn't block the overlay's click handler. */}
        <div style={{ position: "absolute", top: 6, right: 6, pointerEvents: "none" }}>
          {onDisk ? (
            <span
              style={{
                display: "inline-flex",
                width: 18,
                height: 18,
                borderRadius: "50%",
                background: "var(--color-success)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 10,
                color: "white",
                fontWeight: 700,
                boxShadow: "0 1px 3px rgba(0,0,0,0.4)",
              }}
            >
              ✓
            </span>
          ) : movie.monitored ? (
            <span
              style={{
                display: "inline-flex",
                width: 18,
                height: 18,
                borderRadius: "50%",
                background: "var(--color-warning)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 9,
                color: "white",
                fontWeight: 700,
                boxShadow: "0 1px 3px rgba(0,0,0,0.4)",
              }}
            >
              ●
            </span>
          ) : null}
        </div>

        {/* Watched badge */}
        {movie.watch_status?.watched && (
          <div
            data-testid={`watched-${movie.id}`}
            style={{
              position: "absolute",
              bottom: 6,
              left: 6,
              padding: "2px 6px",
              borderRadius: 3,
              background: "rgba(0,0,0,0.7)",
              backdropFilter: "blur(4px)",
              fontSize: 9,
              fontWeight: 600,
              color: "rgba(255,255,255,0.8)",
              pointerEvents: "none",
              letterSpacing: "0.04em",
            }}
          >
            WATCHED
          </div>
        )}
      </div>

      {/* Card footer */}
      <div style={{ paddingTop: 8 }}>
        <p
          style={{
            margin: 0,
            fontSize: 13,
            fontWeight: 500,
            color: "var(--color-text-primary)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {movie.title}
        </p>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 5,
            marginTop: 3,
            flexWrap: "wrap",
          }}
        >
          {movie.year > 0 && (
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
              {movie.year}
            </span>
          )}
          <StatusBadge movie={movie} />
        </div>
      </div>
    </div>
  );
}

// ListRow

function ListRow({
  movie,
  isLast,
  selectionMode,
  isSelected,
  onToggle,
}: {
  movie: Movie;
  isLast: boolean;
  selectionMode: boolean;
  isSelected: boolean;
  onToggle: (id: string) => void;
}) {
  const navigate = useNavigate();

  return (
    <tr
      onClick={() =>
        selectionMode ? onToggle(movie.id) : navigate(`/movies/${movie.id}`)
      }
      onMouseEnter={(e) => {
        if (!isSelected) {
          e.currentTarget.style.background =
            "color-mix(in srgb, var(--color-accent) 6%, transparent)";
        }
      }}
      onMouseLeave={(e) => {
        if (!isSelected) {
          e.currentTarget.style.background = "";
        }
      }}
      style={{
        borderBottom: isLast ? "none" : "1px solid var(--color-border-subtle)",
        background: isSelected
          ? "color-mix(in srgb, var(--color-accent) 8%, transparent)"
          : undefined,
        cursor: "pointer",
      }}
    >
      {/* Selection checkbox */}
      {selectionMode && (
        <td
          onClick={(e) => e.stopPropagation()}
          style={{ padding: "0 0 0 16px", width: 32, height: 60 }}
        >
          <input
            type="checkbox"
            checked={isSelected}
            onChange={() => onToggle(movie.id)}
            style={{
              width: 15,
              height: 15,
              cursor: "pointer",
              accentColor: "var(--color-accent)",
            }}
          />
        </td>
      )}
      {/* Poster thumb */}
      <td style={{ padding: "0 0 0 16px", height: 60, width: 44 }}>
        <div
          style={{
            width: 32,
            height: 48,
            borderRadius: 3,
            overflow: "hidden",
            background: "var(--color-bg-subtle)",
            flexShrink: 0,
          }}
        >
          <Poster
            src={movie.poster_url}
            title={movie.title}
            year={movie.year}
            style={{ borderRadius: 4, width: "100%", height: "100%" }}
            imgStyle={{ borderRadius: 4 }}
          />
        </div>
      </td>

      {/* Title */}
      <td
        style={{
          padding: "0 8px 0 12px",
          height: 60,
          color: "var(--color-text-primary)",
          fontWeight: 500,
          fontSize: 13,
        }}
      >
        {movie.title}
      </td>

      {/* Year */}
      <td
        style={{
          padding: "0 16px",
          height: 60,
          color: "var(--color-text-muted)",
          fontSize: 12,
          whiteSpace: "nowrap",
        }}
      >
        {movie.year || "—"}
      </td>

      {/* Status */}
      <td style={{ padding: "0 16px", height: 60 }}>
        <StatusBadge movie={movie} />
      </td>

      {/* Monitored */}
      <td style={{ padding: "0 16px", height: 60 }}>
        <span
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 5,
            fontSize: 12,
            color: movie.monitored
              ? "var(--color-success)"
              : "var(--color-text-muted)",
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              background: movie.monitored
                ? "var(--color-success)"
                : "var(--color-text-muted)",
            }}
          />
          {movie.monitored ? "Yes" : "No"}
        </span>
      </td>

      {/* On disk */}
      <td style={{ padding: "0 16px", height: 60, fontSize: 12 }}>
        {movie.path ? (
          <span style={{ color: "var(--color-success)" }}>On disk</span>
        ) : (
          <span
            style={{
              color: "var(--color-text-muted)",
              fontStyle: "italic",
            }}
          >
            Missing
          </span>
        )}
      </td>
    </tr>
  );
}

// Add Movie Dialog

function GridSkeleton({ posterSize }: { posterSize: number }) {
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 16 }}>
      {Array.from({ length: 24 }).map((_, i) => (
        <div key={i} style={{ width: posterSize, flexShrink: 0, transition: "width 150ms ease" }}>
          <div
            style={{
              paddingBottom: "150%",
              position: "relative",
              borderRadius: 8,
              overflow: "hidden",
            }}
          >
            <div className="skeleton" style={{ position: "absolute", inset: 0 }} />
          </div>
          <div
            className="skeleton"
            style={{ height: 14, borderRadius: 3, marginTop: 8, width: "85%" }}
          />
          <div
            className="skeleton"
            style={{ height: 12, borderRadius: 3, marginTop: 4, width: "55%" }}
          />
        </div>
      ))}
    </div>
  );
}

// Page

export default function Dashboard() {
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    return (localStorage.getItem("gallery-view") as ViewMode) || "grid";
  });

  const [posterSize, setPosterSize] = useState<number>(() => {
    return Number(localStorage.getItem("gallery-zoom")) || 160;
  });

  const [search, setSearch] = useState("");
  const [monitoredFilter, setMonitoredFilter] =
    useState<MonitoredFilter>("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [onDiskFilter, setOnDiskFilter] = useState<OnDiskFilter>("all");
  const [libraryFilter, setLibraryFilter] = useState("");
  const [sortField, setSortField] = useState<SortField>("title");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [showAdd, setShowAdd] = useState(false);
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [showBulkEdit, setShowBulkEdit] = useState(false);
  const [bulkDeleteState, setBulkDeleteState] = useState<"idle" | "confirming" | "deleting">("idle");
  const [bulkDeleteProgress, setBulkDeleteProgress] = useState<{ done: number; total: number } | null>(null);
  const bulkDeleteAbort = useRef(false);
  const qc = useQueryClient();

  // Quality filter from URL params (set by StatsPage tier clicks)
  const [searchParams, setSearchParams] = useSearchParams();
  const qualityResolution = searchParams.get("quality_resolution") ?? "";
  const qualitySource = searchParams.get("quality_source") ?? "";

  function clearQualityFilter() {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("quality_resolution");
      next.delete("quality_source");
      return next;
    });
  }

  function toggleSelectMode() {
    setSelectionMode((v) => !v);
    setSelectedIds(new Set());
    setBulkDeleteState("idle");
    setBulkDeleteProgress(null);
  }

  function toggleSelect(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function selectAll() {
    setSelectedIds(new Set(filtered.map((m) => m.id)));
  }

  async function handleBulkDelete() {
    const targets = (data?.movies ?? []).filter((m) => selectedIds.has(m.id));
    setBulkDeleteState("deleting");
    setBulkDeleteProgress({ done: 0, total: targets.length });
    bulkDeleteAbort.current = false;
    let done = 0;
    let failed = 0;
    for (const m of targets) {
      if (bulkDeleteAbort.current) break;
      try {
        await apiFetch<void>(`/movies/${m.id}`, { method: "DELETE" });
        done++;
      } catch {
        failed++;
      }
      setBulkDeleteProgress({ done: done + failed, total: targets.length });
    }
    qc.invalidateQueries({ queryKey: ["movies"] });
    if (failed > 0) {
      toast.error(`${failed} deletion${failed > 1 ? "s" : ""} failed`);
    } else if (!bulkDeleteAbort.current) {
      toast.success(`Deleted ${done} ${done === 1 ? "movie" : "movies"}`);
    }
    setBulkDeleteState("idle");
    setBulkDeleteProgress(null);
    setSelectionMode(false);
    setSelectedIds(new Set());
  }

  const { data, isLoading, error } = useAllMovies();
  const { data: libraries } = useLibraries();
  const hasQualityFilter = !!(qualityResolution || qualitySource);
  const { data: qualityMovieIds } = useQualityMovies(
    qualityResolution,
    qualitySource,
    hasQualityFilter
  );

  // Derive unique status values from actual data
  const statuses = useMemo(() => {
    if (!data?.movies) return [] as Availability[];
    const present = new Set(data.movies.map(availabilityOf));
    return AVAILABILITY_ORDER.filter((a) => present.has(a));
  }, [data?.movies]);

  // Client-side filtering + sorting
  const filtered = useMemo(() => {
    if (!data?.movies) return [] as Movie[];
    let result = data.movies;

    if (search.trim()) {
      const q = search.toLowerCase();
      result = result.filter(
        (m) =>
          m.title.toLowerCase().includes(q) ||
          m.original_title.toLowerCase().includes(q)
      );
    }

    if (monitoredFilter !== "all") {
      const want = monitoredFilter === "monitored";
      result = result.filter((m) => m.monitored === want);
    }

    if (statusFilter !== "all") {
      result = result.filter((m) => availabilityOf(m) === statusFilter);
    }

    if (onDiskFilter !== "all") {
      const want = onDiskFilter === "on_disk";
      result = result.filter((m) => !!m.path === want);
    }

    if (libraryFilter) {
      result = result.filter((m) => m.library_id === libraryFilter);
    }

    // Quality tier filter (from URL params, set by StatsPage tier clicks)
    if (qualityMovieIds) {
      const idSet = new Set(qualityMovieIds);
      result = result.filter((m) => idSet.has(m.id));
    }

    return [...result].sort((a, b) => {
      let cmp = 0;
      if (sortField === "title") cmp = a.title.localeCompare(b.title);
      else if (sortField === "year") cmp = (a.year || 0) - (b.year || 0);
      else if (sortField === "added_at")
        cmp = a.added_at.localeCompare(b.added_at);
      return sortDir === "asc" ? cmp : -cmp;
    });
  }, [
    data?.movies,
    search,
    monitoredFilter,
    statusFilter,
    onDiskFilter,
    libraryFilter,
    qualityMovieIds,
    sortField,
    sortDir,
  ]);

  function handleViewMode(mode: ViewMode) {
    setViewMode(mode);
    localStorage.setItem("gallery-view", mode);
  }

  function handlePosterSize(size: number) {
    setPosterSize(size);
    localStorage.setItem("gallery-zoom", String(size));
  }

  function clearFilters() {
    setSearch("");
    setMonitoredFilter("all");
    setStatusFilter("all");
    setOnDiskFilter("all");
    setLibraryFilter("");
    clearQualityFilter();
  }

  const totalCount = data?.total ?? 0;
  const hasFilters =
    !!search ||
    monitoredFilter !== "all" ||
    statusFilter !== "all" ||
    onDiskFilter !== "all" ||
    !!libraryFilter ||
    hasQualityFilter;

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          marginBottom: 20,
          gap: 16,
        }}
      >
        <div>
          <h1
            style={{
              margin: 0,
              fontSize: 20,
              fontWeight: 600,
              color: "var(--color-text-primary)",
              letterSpacing: "-0.01em",
            }}
          >
            Library
          </h1>
          {!isLoading && data && (
            <p
              style={{
                margin: "4px 0 0",
                fontSize: 13,
                color: "var(--color-text-secondary)",
              }}
            >
              {hasFilters
                ? `${filtered.length.toLocaleString()} of ${totalCount.toLocaleString()} movies`
                : `${totalCount.toLocaleString()} ${totalCount === 1 ? "movie" : "movies"}`}
            </p>
          )}
        </div>
        <div style={{ display: "flex", gap: 8, flexShrink: 0 }}>
          <button
            onClick={toggleSelectMode}
            style={{
              background: selectionMode
                ? "color-mix(in srgb, var(--color-accent) 15%, transparent)"
                : "var(--color-bg-elevated)",
              color: selectionMode
                ? "var(--color-accent)"
                : "var(--color-text-secondary)",
              border: `1px solid ${selectionMode ? "var(--color-accent)" : "var(--color-border-default)"}`,
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
            }}
          >
            {selectionMode ? "Cancel Select" : "Select"}
          </button>
          <button
            onClick={() => setShowAdd(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent)";
            }}
          >
            + Add Movie
          </button>
        </div>
      </div>

      {/* Filter bar */}
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 8,
          marginBottom: 20,
          alignItems: "center",
        }}
      >
        {/* Search */}
        <input
          type="search"
          placeholder="Search by title…"
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          style={{ ...ctrlStyle, minWidth: 180 }}
          onFocus={(e) => {
            (e.currentTarget as HTMLInputElement).style.borderColor =
              "var(--color-accent)";
          }}
          onBlur={(e) => {
            (e.currentTarget as HTMLInputElement).style.borderColor =
              "var(--color-border-default)";
          }}
        />

        {/* Monitored */}
        <select
          value={monitoredFilter}
          onChange={(e) =>
            setMonitoredFilter(e.currentTarget.value as MonitoredFilter)
          }
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All monitored</option>
          <option value="monitored">Monitored</option>
          <option value="unmonitored">Unmonitored</option>
        </select>

        {/* Status */}
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.currentTarget.value)}
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All statuses</option>
          {statuses.map((s) => (
            <option key={s} value={s}>
              {AVAILABILITY[s].label}
            </option>
          ))}
        </select>

        {/* On disk */}
        <select
          value={onDiskFilter}
          onChange={(e) =>
            setOnDiskFilter(e.currentTarget.value as OnDiskFilter)
          }
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="all">All files</option>
          <option value="on_disk">On disk</option>
          <option value="missing">Missing</option>
        </select>

        {/* Library — only shown when multiple libraries exist */}
        {libraries && libraries.length > 1 && (
          <select
            value={libraryFilter}
            onChange={(e) => setLibraryFilter(e.currentTarget.value)}
            style={{ ...ctrlStyle, cursor: "pointer" }}
          >
            <option value="">All libraries</option>
            {libraries.map((lib) => (
              <option key={lib.id} value={lib.id}>
                {lib.name}
              </option>
            ))}
          </select>
        )}

        {/* Sort field */}
        <select
          value={sortField}
          onChange={(e) => setSortField(e.currentTarget.value as SortField)}
          style={{ ...ctrlStyle, cursor: "pointer" }}
        >
          <option value="title">Title</option>
          <option value="year">Year</option>
          <option value="added_at">Date Added</option>
        </select>

        {/* Sort direction */}
        <button
          onClick={() => setSortDir((d) => (d === "asc" ? "desc" : "asc"))}
          title={sortDir === "asc" ? "Ascending — click to flip" : "Descending — click to flip"}
          style={{
            ...ctrlStyle,
            cursor: "pointer",
            padding: "7px 10px",
            fontFamily: "var(--font-family-mono)",
            fontSize: 12,
            color: "var(--color-text-secondary)",
          }}
        >
          {sortDir === "asc" ? "A→Z" : "Z→A"}
        </button>

        {/* Quality filter chip */}
        {hasQualityFilter && (
          <div
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 6,
              padding: "5px 10px",
              background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
              border: "1px solid color-mix(in srgb, var(--color-accent) 30%, transparent)",
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              color: "var(--color-accent)",
            }}
          >
            <span>
              {[qualityResolution, qualitySource].filter(Boolean).join(" ")}
            </span>
            <button
              onClick={clearQualityFilter}
              title="Clear quality filter"
              style={{
                background: "none",
                border: "none",
                padding: 0,
                cursor: "pointer",
                color: "var(--color-accent)",
                display: "flex",
                alignItems: "center",
                lineHeight: 1,
                fontSize: 14,
                opacity: 0.75,
              }}
              onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.opacity = "1"; }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.opacity = "0.75"; }}
            >
              ×
            </button>
          </div>
        )}

        {/* Spacer */}
        <div style={{ flex: 1 }} />

        {/* Zoom slider — grid mode only */}
        {viewMode === "grid" && (
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <svg width="13" height="13" viewBox="0 0 16 16" fill="none" style={{ color: "var(--color-text-muted)", flexShrink: 0 }}>
              <rect x="1" y="4" width="6" height="8" rx="1" stroke="currentColor" strokeWidth="1.5" />
            </svg>
            <input
              type="range"
              min={80}
              max={260}
              step={20}
              value={posterSize}
              onChange={(e) => handlePosterSize(Number(e.target.value))}
              title={`Thumbnail size: ${posterSize}px`}
              style={{ width: 72, accentColor: "var(--color-accent)", cursor: "pointer" }}
            />
            <svg width="17" height="17" viewBox="0 0 16 16" fill="none" style={{ color: "var(--color-text-muted)", flexShrink: 0 }}>
              <rect x="1" y="3" width="8" height="10" rx="1" stroke="currentColor" strokeWidth="1.5" />
            </svg>
          </div>
        )}

        {/* View toggle */}
        <div
          style={{
            display: "flex",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
            overflow: "hidden",
          }}
        >
          <button
            onClick={() => handleViewMode("grid")}
            title="Grid view"
            style={{
              background:
                viewMode === "grid"
                  ? "var(--color-bg-elevated)"
                  : "transparent",
              border: "none",
              padding: "6px 10px",
              cursor: "pointer",
              color:
                viewMode === "grid"
                  ? "var(--color-text-primary)"
                  : "var(--color-text-muted)",
              display: "flex",
              alignItems: "center",
            }}
          >
            <LayoutGrid size={16} strokeWidth={1.5} />
          </button>
          <button
            onClick={() => handleViewMode("list")}
            title="List view"
            style={{
              background:
                viewMode === "list"
                  ? "var(--color-bg-elevated)"
                  : "transparent",
              border: "none",
              borderLeft: "1px solid var(--color-border-default)",
              padding: "6px 10px",
              cursor: "pointer",
              color:
                viewMode === "list"
                  ? "var(--color-text-primary)"
                  : "var(--color-text-muted)",
              display: "flex",
              alignItems: "center",
            }}
          >
            <List size={16} strokeWidth={1.5} />
          </button>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        viewMode === "grid" ? (
          <GridSkeleton posterSize={posterSize} />
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {[1, 2, 3, 4, 5].map((i) => (
              <div
                key={i}
                className="skeleton"
                style={{ height: 60, borderRadius: 4 }}
              />
            ))}
          </div>
        )
      ) : error ? (
        <div
          style={{
            padding: 48,
            textAlign: "center",
            fontSize: 13,
            color: "var(--color-text-muted)",
          }}
        >
          Failed to load movies.
        </div>
      ) : !data?.movies.length ? (
        <div style={{ padding: 80, textAlign: "center" }}>
          <p
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 500,
              color: "var(--color-text-secondary)",
            }}
          >
            No movies in your library
          </p>
          <p
            style={{
              margin: "8px 0 20px",
              fontSize: 13,
              color: "var(--color-text-muted)",
            }}
          >
            Search TMDB to add your first movie.
          </p>
          <button
            onClick={() => setShowAdd(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background =
                "var(--color-accent)";
            }}
          >
            + Add Movie
          </button>
        </div>
      ) : filtered.length === 0 ? (
        <div style={{ padding: 48, textAlign: "center" }}>
          <p
            style={{
              margin: 0,
              fontSize: 14,
              color: "var(--color-text-muted)",
            }}
          >
            No movies match the current filters.
          </p>
          <button
            onClick={clearFilters}
            style={{
              marginTop: 12,
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "6px 14px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Clear filters
          </button>
        </div>
      ) : viewMode === "grid" ? (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 16 }}>
          {filtered.map((movie) => (
            <div key={movie.id} style={{ width: posterSize, flexShrink: 0, transition: "width 150ms ease" }}>
              <PosterCard
                movie={movie}
                selectionMode={selectionMode}
                isSelected={selectedIds.has(movie.id)}
                onToggle={toggleSelect}
              />
            </div>
          ))}
        </div>
      ) : (
        <div
          style={{
            background: "var(--color-bg-surface)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 8,
            boxShadow: "var(--shadow-card)",
            overflow: "hidden",
          }}
        >
          <table
            style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}
          >
            <thead>
              <tr
                style={{
                  borderBottom: "1px solid var(--color-border-subtle)",
                }}
              >
                {selectionMode && <th style={{ ...thStyle, width: 40 }} />}
                <th style={thStyle} />
                <th style={thStyle}>Title</th>
                <th style={thStyle}>Year</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>Monitored</th>
                <th style={thStyle}>File</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((movie, i) => (
                <ListRow
                  key={movie.id}
                  movie={movie}
                  isLast={i === filtered.length - 1}
                  selectionMode={selectionMode}
                  isSelected={selectedIds.has(movie.id)}
                  onToggle={toggleSelect}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Legend — only shown in grid view with at least one movie */}
      {viewMode === "grid" && filtered.length > 0 && (
        <div style={{ display: "flex", gap: 20, alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span
              style={{
                display: "inline-flex",
                width: 14,
                height: 14,
                borderRadius: "50%",
                background: "var(--color-success)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 8,
                color: "white",
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              ✓
            </span>
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>On disk</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span
              style={{
                display: "inline-flex",
                width: 14,
                height: 14,
                borderRadius: "50%",
                background: "var(--color-warning)",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 8,
                color: "white",
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              ●
            </span>
            <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Monitored, missing</span>
          </div>
        </div>
      )}

      {showAdd && <AddMovieDialog onClose={() => setShowAdd(false)} />}

      {/* Sticky selection toolbar */}
      {selectionMode && (
        <div
          style={{
            position: "fixed",
            bottom: 24,
            left: "50%",
            transform: "translateX(-50%)",
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 8,
            padding: "12px 20px",
            display: "flex",
            gap: 10,
            alignItems: "center",
            boxShadow: "0 4px 16px rgba(0,0,0,0.25)",
            zIndex: 50,
            whiteSpace: "nowrap",
          }}
        >
          <span
            style={{
              fontSize: 13,
              color: "var(--color-text-primary)",
              fontWeight: 500,
              minWidth: 80,
            }}
          >
            {selectedIds.size} {selectedIds.size === 1 ? "movie" : "movies"} selected
          </span>

          {bulkDeleteState === "confirming" ? (
            <>
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)" }}>
                Delete {selectedIds.size} {selectedIds.size === 1 ? "movie" : "movies"}?
              </span>
              <button
                onClick={handleBulkDelete}
                style={{
                  background: "var(--color-danger)",
                  color: "white",
                  border: "none",
                  borderRadius: 6,
                  padding: "7px 16px",
                  fontSize: 13,
                  fontWeight: 500,
                  cursor: "pointer",
                }}
              >
                Yes, delete
              </button>
              <button
                onClick={() => setBulkDeleteState("idle")}
                style={{
                  background: "transparent",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "7px 14px",
                  fontSize: 13,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Cancel
              </button>
            </>
          ) : bulkDeleteState === "deleting" ? (
            <>
              <span style={{ fontSize: 13, color: "var(--color-text-muted)" }}>
                Deleting {bulkDeleteProgress?.done ?? 0}/{bulkDeleteProgress?.total ?? 0}…
              </span>
              <button
                onClick={() => { bulkDeleteAbort.current = true; }}
                style={{
                  background: "transparent",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "7px 14px",
                  fontSize: 13,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Stop
              </button>
            </>
          ) : (
            <>
              {/* Select all visible */}
              {selectedIds.size < filtered.length && (
                <button
                  onClick={selectAll}
                  style={{
                    background: "transparent",
                    border: "1px solid var(--color-border-default)",
                    borderRadius: 6,
                    padding: "7px 14px",
                    fontSize: 13,
                    color: "var(--color-text-secondary)",
                    cursor: "pointer",
                  }}
                >
                  Select all {filtered.length}
                </button>
              )}

              {selectedIds.size > 0 && (
                <>
                  <button
                    onClick={() => setShowBulkEdit(true)}
                    style={{
                      background: "var(--color-accent)",
                      color: "var(--color-accent-fg)",
                      border: "none",
                      borderRadius: 6,
                      padding: "7px 16px",
                      fontSize: 13,
                      fontWeight: 500,
                      cursor: "pointer",
                    }}
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => setBulkDeleteState("confirming")}
                    style={{
                      background: "color-mix(in srgb, var(--color-danger) 12%, transparent)",
                      border: "1px solid color-mix(in srgb, var(--color-danger) 35%, transparent)",
                      borderRadius: 6,
                      padding: "7px 14px",
                      fontSize: 13,
                      color: "var(--color-danger)",
                      cursor: "pointer",
                    }}
                  >
                    Delete
                  </button>
                </>
              )}

              <button
                onClick={toggleSelectMode}
                style={{
                  background: "transparent",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "7px 14px",
                  fontSize: 13,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Cancel
              </button>
            </>
          )}
        </div>
      )}

      {showBulkEdit && (
        <BulkEditModal
          movies={data?.movies ?? []}
          selectedIds={selectedIds}
          onClose={() => {
            setShowBulkEdit(false);
            setSelectionMode(false);
            setSelectedIds(new Set());
          }}
        />
      )}
    </div>
  );
}
