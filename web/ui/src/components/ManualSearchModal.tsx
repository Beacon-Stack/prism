import { useState, useCallback, useMemo } from "react";
import {
  X,
  Loader2,
  ArrowUp,
  ArrowDown,
  Download,
  Wifi,
  TriangleAlert,
} from "lucide-react";
import { toast } from "sonner";
import { useMovieReleases, useGrabRelease, useMovie, type GrabReleaseRequest } from "@/api/movies";
import { useMovieFiles } from "@/api/movies";
import { useQualityProfiles } from "@/api/quality-profiles";
import type { Release, QualityConflict } from "@/types";
import { formatBytes } from "@beacon-shared/utils";
import QualityBadge from "@/components/QualityBadge";
import Modal from "@beacon-shared/Modal";

// ── Helpers ───────────────────────────────────────────────────────────────────

type SortField = "seeds" | "size" | "age";
type SortDir = "asc" | "desc";

function formatAge(days: number | undefined): string {
  if (days === undefined || days <= 0) return "—";
  if (days < 1) return "Today";
  if (days === 1) return "1d";
  if (days < 30) return `${Math.floor(days)}d`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo`;
  return `${Math.floor(months / 12)}y`;
}

// seedHealth maps a seed count to a colour and label. Ported verbatim from
// Pilot's ManualSearchModal so both apps share the same buckets.
function seedHealth(seeds: number | undefined): { color: string; label: string } {
  const s = seeds ?? 0;
  if (s === 0) return { color: "var(--color-danger)", label: "Dead" };
  if (s <= 2) return { color: "var(--color-warning)", label: "Poor" };
  if (s <= 10) return { color: "var(--color-text-secondary)", label: "OK" };
  if (s <= 50) return { color: "var(--color-success)", label: "Good" };
  return { color: "var(--color-success)", label: "Great" };
}

// closeOnGrab returns the user's preference for whether the modal should
// close after a successful grab. Persisted in localStorage so users can
// flip the default (stay-open) without a backend setting.
function shouldCloseOnGrab(): boolean {
  if (typeof window === "undefined") return false;
  return window.localStorage.getItem("manualSearchModal.closeOnGrab") === "true";
}

// ── Conflict pills ────────────────────────────────────────────────────────────

function ConflictPills({ conflicts }: { conflicts: QualityConflict[] }) {
  if (conflicts.length === 0) return null;
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 4, marginTop: 4 }}>
      {conflicts.map((c, i) => {
        const isWarning = c.severity === "warning";
        return (
          <span
            key={i}
            title={`${c.dimension}: ${c.current} → ${c.candidate}`}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 3,
              padding: "1px 6px",
              borderRadius: 3,
              fontSize: 10,
              fontWeight: 500,
              background: isWarning
                ? "color-mix(in srgb, var(--color-warning) 14%, transparent)"
                : "color-mix(in srgb, var(--color-text-muted) 10%, transparent)",
              color: isWarning ? "var(--color-warning)" : "var(--color-text-muted)",
              border: `1px solid ${
                isWarning
                  ? "color-mix(in srgb, var(--color-warning) 30%, transparent)"
                  : "var(--color-border-subtle)"
              }`,
            }}
          >
            {isWarning && <TriangleAlert size={9} strokeWidth={2} />}
            {c.summary}
          </span>
        );
      })}
    </div>
  );
}

// ── Release row ───────────────────────────────────────────────────────────────

interface ReleaseRowProps {
  release: Release;
  grabbed: boolean;
  grabError?: string;
  onGrab: () => void;
  isPending: boolean;
}

function ReleaseRow({ release, grabbed, grabError, onGrab, isPending }: ReleaseRowProps) {
  const dead = (release.seeds ?? 0) === 0;
  const health = seedHealth(release.seeds);
  const hasConflicts = (release.conflicts?.length ?? 0) > 0;

  return (
    <tr
      style={{
        borderBottom: "1px solid var(--color-border-subtle)",
        opacity: dead && !grabbed ? 0.55 : 1,
        background: hasConflicts
          ? "color-mix(in srgb, var(--color-warning) 5%, transparent)"
          : undefined,
      }}
    >
      {/* Title */}
      <td
        style={{
          padding: "10px 12px",
          fontSize: 12,
          color: "var(--color-text-primary)",
          maxWidth: 380,
        }}
      >
        <div
          style={{
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            fontFamily: "var(--font-family-mono)",
          }}
          title={release.title}
        >
          {release.title}
        </div>
        <div
          style={{
            fontSize: 11,
            color: "var(--color-text-muted)",
            marginTop: 3,
            display: "flex",
            alignItems: "center",
            gap: 8,
            flexWrap: "wrap",
          }}
        >
          <span>{release.indexer}</span>
          {dead && (
            <span
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 2,
                color: "var(--color-danger)",
                fontSize: 10,
              }}
            >
              <TriangleAlert size={10} /> No seeders
            </span>
          )}
        </div>
        {hasConflicts && release.conflicts && <ConflictPills conflicts={release.conflicts} />}
        {grabError && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-danger)" }}>
            {grabError}
          </p>
        )}
      </td>

      {/* Quality + Edition */}
      <td style={{ padding: "10px 12px", width: 170 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
          <QualityBadge quality={release.quality} />
          {release.edition && (
            <span
              style={{
                display: "inline-block",
                padding: "1px 6px",
                borderRadius: 4,
                fontSize: 10,
                fontWeight: 600,
                background: "color-mix(in srgb, var(--color-info, #3b82f6) 15%, transparent)",
                color: "var(--color-info, #3b82f6)",
              }}
            >
              {release.edition}
            </span>
          )}
        </div>
      </td>

      {/* Size */}
      <td
        style={{
          padding: "10px 12px",
          fontSize: 12,
          color: "var(--color-text-secondary)",
          whiteSpace: "nowrap",
          width: 86,
        }}
      >
        {formatBytes(release.size)}
      </td>

      {/* Seeds */}
      <td
        style={{
          padding: "10px 12px",
          width: 80,
          fontSize: 12,
          fontWeight: 600,
          color: health.color,
          whiteSpace: "nowrap",
        }}
        title={`${release.seeds ?? 0} seeders / ${release.peers ?? 0} peers — ${health.label}`}
      >
        <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
          <Wifi size={12} strokeWidth={1.5} />
          {release.seeds ?? 0}
        </span>
      </td>

      {/* Age */}
      <td
        style={{
          padding: "10px 12px",
          fontSize: 12,
          color: "var(--color-text-muted)",
          whiteSpace: "nowrap",
          width: 60,
        }}
      >
        {formatAge(release.age_days)}
      </td>

      {/* Grab */}
      <td style={{ padding: "10px 12px", width: 80 }}>
        {grabbed ? (
          <span
            style={{
              fontSize: 11,
              color: "var(--color-success)",
              fontWeight: 600,
              whiteSpace: "nowrap",
            }}
          >
            Grabbed ✓
          </span>
        ) : (
          <button
            onClick={onGrab}
            disabled={isPending}
            title={dead ? "No seeders — grab anyway?" : "Grab this release"}
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: "5px 8px",
              background: dead ? "var(--color-bg-elevated)" : "var(--color-accent)",
              border: dead ? "1px solid var(--color-border-default)" : "none",
              borderRadius: 5,
              cursor: isPending ? "not-allowed" : "pointer",
              color: dead ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              opacity: isPending ? 0.6 : 1,
            }}
          >
            <Download size={13} strokeWidth={2} />
          </button>
        )}
      </td>
    </tr>
  );
}

// ── Modal ─────────────────────────────────────────────────────────────────────

interface ManualSearchModalProps {
  movieId: string;
  movieTitle: string;
  onClose: () => void;
}

export function ManualSearchModal({ movieId, movieTitle, onClose }: ManualSearchModalProps) {
  const { data: releases, isLoading, error, refetch } = useMovieReleases(movieId);
  const grab = useGrabRelease();

  // Profile + current-file lookup for the header note. All three hooks hit
  // React Query's shared cache, so when this modal is opened from MovieDetail
  // the data is already loaded; opening from WantedPage costs at most three
  // small extra requests on first open.
  const { data: movie } = useMovie(movieId);
  const { data: profiles } = useQualityProfiles();
  const { data: files } = useMovieFiles(movieId);

  const profileName = useMemo(() => {
    if (!movie || !profiles) return undefined;
    return profiles.find((p) => p.id === movie.quality_profile_id)?.name;
  }, [movie, profiles]);

  const currentLabel = useMemo(() => {
    if (!files || files.length === 0) return undefined;
    const best = files[0];
    return best.quality?.name || `${best.quality?.resolution ?? ""} ${best.quality?.source ?? ""}`.trim();
  }, [files]);

  const [grabbedGuids, setGrabbedGuids] = useState<Set<string>>(new Set());
  const [pendingGuids, setPendingGuids] = useState<Set<string>>(new Set());
  const [grabErrors, setGrabErrors] = useState<Record<string, string>>({});
  const [sort, setSort] = useState<{ field: SortField; dir: SortDir } | null>(null);

  const sortedReleases = useMemo(() => {
    if (!releases) return [];
    if (!sort) return releases;
    const arr = [...releases];
    arr.sort((a, b) => {
      let av: number;
      let bv: number;
      switch (sort.field) {
        case "seeds":
          av = a.seeds ?? 0;
          bv = b.seeds ?? 0;
          break;
        case "size":
          av = a.size;
          bv = b.size;
          break;
        case "age":
          av = a.age_days ?? 0;
          bv = b.age_days ?? 0;
          break;
      }
      return sort.dir === "desc" ? bv - av : av - bv;
    });
    return arr;
  }, [releases, sort]);

  function toggleSort(field: SortField) {
    setSort((prev) => {
      if (prev?.field !== field) return { field, dir: "desc" };
      if (prev.dir === "desc") return { field, dir: "asc" };
      return null; // third click resets to API order
    });
  }

  const handleGrab = useCallback(
    (release: Release) => {
      const body: GrabReleaseRequest & { movieId: string } = {
        movieId,
        guid: release.guid,
        title: release.title,
        protocol: release.protocol,
        download_url: release.download_url,
        size: release.size,
      };
      setPendingGuids((prev) => new Set([...prev, release.guid]));
      grab.mutate(body, {
        onSuccess: () => {
          setPendingGuids((prev) => {
            const n = new Set(prev);
            n.delete(release.guid);
            return n;
          });
          setGrabbedGuids((prev) => new Set([...prev, release.guid]));
          toast.success("Sent to download client");
          if (shouldCloseOnGrab()) {
            onClose();
          }
        },
        onError: (e) => {
          setPendingGuids((prev) => {
            const n = new Set(prev);
            n.delete(release.guid);
            return n;
          });
          setGrabErrors((prev) => ({ ...prev, [release.guid]: e.message }));
          setTimeout(
            () =>
              setGrabErrors((prev) => {
                const n = { ...prev };
                delete n[release.guid];
                return n;
              }),
            5000,
          );
        },
      });
    },
    [movieId, grab, onClose],
  );

  const liveCount = releases?.filter((r) => (r.seeds ?? 0) > 0).length ?? 0;

  const sortIcon = (field: SortField) => {
    if (sort?.field !== field) return null;
    return sort.dir === "desc" ? (
      <ArrowDown size={10} strokeWidth={2.5} />
    ) : (
      <ArrowUp size={10} strokeWidth={2.5} />
    );
  };

  const thStyle: React.CSSProperties = {
    textAlign: "left",
    padding: "8px 12px",
    fontSize: 11,
    fontWeight: 600,
    letterSpacing: "0.06em",
    textTransform: "uppercase",
    color: "var(--color-text-muted)",
    borderBottom: "1px solid var(--color-border-subtle)",
    position: "sticky",
    top: 0,
    background: "var(--color-bg-surface)",
  };

  const sortableThStyle = (field: SortField): React.CSSProperties => ({
    ...thStyle,
    cursor: "pointer",
    userSelect: "none",
    color: sort?.field === field ? "var(--color-accent)" : "var(--color-text-muted)",
  });

  return (
    <Modal onClose={onClose} width={800} maxHeight="calc(100vh - 64px)">
      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "18px 20px",
          borderBottom: "1px solid var(--color-border-subtle)",
          flexShrink: 0,
        }}
      >
        <div>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
            {movieTitle}
          </h2>
          {!isLoading && releases && releases.length > 0 && (
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>
              {releases.length} results · {liveCount} with seeders
            </div>
          )}
        </div>
        <button
          onClick={onClose}
          style={{
            background: "none",
            border: "none",
            cursor: "pointer",
            color: "var(--color-text-muted)",
            display: "flex",
            padding: 4,
          }}
        >
          <X size={18} />
        </button>
      </div>

      {/* Profile / Current note — replaces the old "Why?" panel */}
      {(profileName || currentLabel) && (
        <div
          style={{
            padding: "8px 20px",
            borderBottom: "1px solid var(--color-border-subtle)",
            background: "var(--color-bg-surface)",
            fontSize: 11,
            color: "var(--color-text-muted)",
            display: "flex",
            gap: 16,
            flexWrap: "wrap",
            flexShrink: 0,
          }}
        >
          {profileName && (
            <span>
              Quality Profile:{" "}
              <span style={{ color: "var(--color-text-secondary)", fontWeight: 500 }}>
                {profileName}
              </span>
            </span>
          )}
          <span>
            Current:{" "}
            <span style={{ color: "var(--color-text-secondary)", fontWeight: 500 }}>
              {currentLabel || "(none)"}
            </span>
          </span>
        </div>
      )}

      {/* Body */}
      <div style={{ overflowY: "auto", flex: 1 }}>
        {/* Loading */}
        {isLoading && (
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
              padding: "56px 24px",
              gap: 12,
              color: "var(--color-text-muted)",
              fontSize: 13,
            }}
          >
            <Loader2
              size={28}
              strokeWidth={1.5}
              style={{ color: "var(--color-accent)", animation: "spin 1s linear infinite" }}
            />
            Searching indexers...
          </div>
        )}

        {/* Error */}
        {error && (
          <div style={{ margin: 20 }}>
            <div
              style={{
                padding: 16,
                background: "color-mix(in srgb, var(--color-danger) 10%, transparent)",
                border: "1px solid color-mix(in srgb, var(--color-danger) 30%, transparent)",
                borderRadius: 8,
                color: "var(--color-danger)",
                fontSize: 13,
              }}
            >
              {(error as Error).message ?? "Search failed. Check that indexers are configured and reachable."}
            </div>
            <div style={{ display: "flex", justifyContent: "center", marginTop: 12 }}>
              <button
                onClick={() => refetch()}
                style={{
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "6px 14px",
                  fontSize: 12,
                  fontWeight: 500,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                }}
              >
                Retry
              </button>
            </div>
          </div>
        )}

        {/* Empty */}
        {!isLoading && !error && releases && releases.length === 0 && (
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              padding: "56px 24px",
              gap: 12,
              fontSize: 14,
              color: "var(--color-text-muted)",
            }}
          >
            No releases found.
            <button
              onClick={() => refetch()}
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "6px 14px",
                fontSize: 12,
                fontWeight: 500,
                color: "var(--color-text-secondary)",
                cursor: "pointer",
              }}
            >
              Search Again
            </button>
          </div>
        )}

        {/* Results table */}
        {!isLoading && !error && releases && releases.length > 0 && (
          <table
            style={{
              width: "100%",
              borderCollapse: "collapse",
              fontSize: 13,
            }}
          >
            <thead>
              <tr>
                <th style={thStyle}>Release</th>
                <th style={thStyle}>Quality</th>
                <th style={sortableThStyle("size")} onClick={() => toggleSort("size")}>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                    Size {sortIcon("size")}
                  </span>
                </th>
                <th style={sortableThStyle("seeds")} onClick={() => toggleSort("seeds")}>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                    Seeds {sortIcon("seeds")}
                  </span>
                </th>
                <th style={sortableThStyle("age")} onClick={() => toggleSort("age")}>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                    Age {sortIcon("age")}
                  </span>
                </th>
                <th style={thStyle}></th>
              </tr>
            </thead>
            <tbody>
              {sortedReleases.map((release) => (
                <ReleaseRow
                  key={release.guid}
                  release={release}
                  grabbed={grabbedGuids.has(release.guid)}
                  grabError={grabErrors[release.guid]}
                  onGrab={() => handleGrab(release)}
                  isPending={pendingGuids.has(release.guid)}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </Modal>
  );
}
