import { useState, useCallback, useMemo, useEffect } from "react";
import {
  X,
  Loader2,
  ArrowUp,
  ArrowDown,
  Download,
  Wifi,
  TriangleAlert,
  History,
} from "lucide-react";
import { toast } from "sonner";
import { useMovieReleases, useGrabRelease, useMovie, type GrabReleaseRequest } from "@/api/movies";
import { useMovieFiles } from "@/api/movies";
import { useQualityProfiles } from "@/api/quality-profiles";
import type { Release, QualityConflict } from "@/types";
import { formatBytes } from "@beacon-shared/utils";
import QualityBadge from "@/components/QualityBadge";
import Modal from "@beacon-shared/Modal";
import { useConfirm } from "@beacon-shared/ConfirmDialog";

// ── Helpers ───────────────────────────────────────────────────────────────────

type SortField = "seeds" | "size" | "age" | "quality";
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

// Theater recordings: filmed or captured in a cinema (CAM, telesync) or an
// unfinished workprint. Hidden from manual search unless asked for; quality
// profiles already reject them for auto-grab.
const THEATER_SOURCES = new Set(["cam", "telesync", "workprint"]);
const isTheaterRecording = (r: Release) =>
  THEATER_SOURCES.has((r.quality?.source ?? "").toLowerCase());

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
  const reasons = [
    ...(release.filter_reasons ?? []),
    ...(isTheaterRecording(release) ? ["theater recording (CAM/TS)"] : []),
  ];
  const filtered = reasons.length > 0;
  const alreadyGrabbed = !!release.already_grabbed_at;
  const grabbedAgeLabel = alreadyGrabbed
    ? formatAge((Date.now() - new Date(release.already_grabbed_at!).getTime()) / 86400000)
    : "";

  return (
    <tr
      style={{
        borderBottom: "1px solid var(--color-border-subtle)",
        // Flagged-by-filter rows are gray. Dead-but-not-flagged still
        // get the lighter dim. Both are still clickable.
        opacity: filtered && !grabbed ? 0.45 : dead && !grabbed ? 0.55 : 1,
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
          {release.other_sources && release.other_sources.length > 0 && (
            <span
              title={`Same torrent also reported by: ${release.other_sources.join(", ")}. Seed count shown is the median across all sources.`}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 2,
                padding: "1px 6px",
                borderRadius: 8,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                color: "var(--color-text-muted)",
                fontSize: 10,
              }}
            >
              +{release.other_sources.length} more
            </span>
          )}
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
          {filtered && (
            <span
              title={reasons.join("; ")}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 2,
                padding: "1px 6px",
                borderRadius: 8,
                background: "color-mix(in srgb, var(--color-warning) 15%, transparent)",
                color: "var(--color-warning)",
                fontSize: 10,
                fontWeight: 600,
              }}
            >
              Filtered
            </span>
          )}
          {alreadyGrabbed && (
            <span
              title={`Previously grabbed ${grabbedAgeLabel} ago — status: ${release.already_grabbed_status ?? "unknown"}`}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 3,
                padding: "1px 6px",
                borderRadius: 8,
                background: "color-mix(in srgb, var(--color-warning) 15%, transparent)",
                color: "var(--color-warning)",
                border: "1px solid color-mix(in srgb, var(--color-warning) 30%, transparent)",
                fontSize: 10,
                fontWeight: 500,
              }}
            >
              <History size={10} /> Already grabbed {grabbedAgeLabel} ago
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
            title={
              filtered
                ? `Filtered: ${reasons.join("; ")} — grab anyway?`
                : dead
                ? "No seeders — grab anyway?"
                : "Grab this release"
            }
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
  const { data, isLoading, error, refetch, isFetching } = useMovieReleases(movieId);
  const allReleases = data?.releases;
  const [showTheater, setShowTheater] = useState(false);
  const theaterCount = useMemo(() => (allReleases ?? []).filter(isTheaterRecording).length, [allReleases]);
  const releases = useMemo(
    () => (showTheater || !allReleases ? allReleases : allReleases.filter((r) => !isTheaterRecording(r))),
    [allReleases, showTheater],
  );
  const indexerOutcomes = data?.indexers ?? [];
  const failedIndexers = indexerOutcomes.filter((o) => o.error);
  const pendingIndexers = indexerOutcomes.filter((o) => o.pending && !o.error);
  // Results stream in as indexers answer; the last update has done=true.
  const streaming = isFetching && data !== undefined && !data.done;
  const waitingOn = streaming ? indexerOutcomes.filter((o) => o.searching) : [];
  const answered = indexerOutcomes.length - waitingOn.length;
  const nothingYet = isLoading || (streaming && (!releases || releases.length === 0));

  // While Cloudflare-walled indexers are still solving server-side,
  // quietly search again every 30s: their results land in Pulse's cache
  // and merge in. Capped so a stuck indexer doesn't poll forever; the
  // previous results stay on screen during each search.
  const [pendingRetries, setPendingRetries] = useState(0);
  const maxPendingRetries = 5;
  useEffect(() => {
    if (pendingIndexers.length === 0 || pendingRetries >= maxPendingRetries || isFetching) return;
    const t = setTimeout(() => {
      setPendingRetries((n) => n + 1);
      void refetch();
    }, 30_000);
    return () => clearTimeout(t);
  }, [pendingIndexers.length, pendingRetries, isFetching, refetch]);
  const grab = useGrabRelease();
  const confirm = useConfirm();

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
        case "quality":
          // Sort on quality_score (resolution*100 + source*10 + codec).
          av = a.quality_score ?? 0;
          bv = b.quality_score ?? 0;
          break;
      }
      const primary = sort.dir === "desc" ? bv - av : av - bv;
      // Same-quality tiebreak: more seeds wins. Matches the auto-search
      // ranking convention so the manual list mirrors it visually.
      if (primary !== 0 || sort.field !== "quality") return primary;
      return (b.seeds ?? 0) - (a.seeds ?? 0);
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
    async (release: Release) => {
      // Already-grabbed guardrail: when grab_history has a row for this
      // release's GUID, prompt the user to confirm before re-grabbing.
      // Soft warning; confirmed grabs proceed normally.
      if (release.already_grabbed_at) {
        const ageDays = Math.max(
          0,
          Math.floor((Date.now() - new Date(release.already_grabbed_at).getTime()) / 86400000),
        );
        const ageLabel = ageDays === 0 ? "today" : ageDays === 1 ? "1 day ago" : `${ageDays} days ago`;
        const status = release.already_grabbed_status || "unknown";
        const ok = await confirm({
          title: "Already grabbed",
          message: `You grabbed this exact release ${ageLabel} (status: ${status}). Grab it again?`,
          confirmLabel: "Grab again",
          danger: false,
        });
        if (!ok) return;
      }
      const body: GrabReleaseRequest & { movieId: string } = {
        movieId,
        guid: release.guid,
        title: release.title,
        // Forward the indexer UUID so grab_history rows are attributed.
        // Without this, Top Indexers stats stay permanently empty even
        // after successful downloads.
        indexer_id: release.indexer_id,
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
    [movieId, grab, confirm, onClose],
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
          {!nothingYet && releases && releases.length > 0 && (
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>
              {releases.length} results · {liveCount} with seeders
            </div>
          )}
          {!nothingYet && theaterCount > 0 && (
            <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginTop: 2 }}>
              {showTheater
                ? `Showing ${theaterCount} theater recording${theaterCount === 1 ? "" : "s"} (CAM/TS) · `
                : `${theaterCount} theater recording${theaterCount === 1 ? "" : "s"} (CAM/TS) hidden · `}
              <button
                onClick={() => setShowTheater((v) => !v)}
                style={{
                  background: "none",
                  border: "none",
                  padding: 0,
                  font: "inherit",
                  color: "var(--color-accent)",
                  cursor: "pointer",
                  textDecoration: "underline",
                }}
              >
                {showTheater ? "Hide" : "Show"}
              </button>
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
        {/* Loading: nothing to show yet */}
        {nothingYet && !error && (
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
            {indexerOutcomes.length > 0
              ? `Searching indexers… ${answered} of ${indexerOutcomes.length} answered`
              : "Searching indexers..."}
          </div>
        )}

        {/* Results so far, with the indexers still being waited on. */}
        {!nothingYet && !error && waitingOn.length > 0 && (
          <div style={{ margin: "12px 20px 0", fontSize: 12, color: "var(--color-text-muted)", display: "flex", alignItems: "center", gap: 8 }}>
            <Loader2 size={13} style={{ color: "var(--color-accent)", animation: "spin 1s linear infinite", flexShrink: 0 }} />
            Still waiting on {waitingOn.map((o) => o.indexer_name).join(", ")}
          </div>
        )}

        {/* Cloudflare-walled indexers keep solving server-side; their
            results merge in automatically. */}
        {!nothingYet && !error && !streaming && pendingIndexers.length > 0 && pendingRetries < maxPendingRetries && (
          <div
            role="status"
            style={{
              margin: "12px 20px 0",
              padding: "10px 14px",
              background: "color-mix(in srgb, var(--color-accent) 8%, transparent)",
              border: "1px solid color-mix(in srgb, var(--color-accent) 25%, transparent)",
              borderRadius: 8,
              fontSize: 12,
              color: "var(--color-text-secondary)",
              display: "flex",
              alignItems: "center",
              gap: 8,
            }}
          >
            <Loader2 size={13} style={{ color: "var(--color-accent)", animation: "spin 1s linear infinite", flexShrink: 0 }} />
            <span>
              Still searching {pendingIndexers.map((o) => o.indexer_name).join(", ")}. Cloudflare solving takes a minute or two;
              results appear here automatically.
            </span>
          </div>
        )}
        {!nothingYet && !error && !streaming && pendingIndexers.length > 0 && pendingRetries >= maxPendingRetries && (
          <div
            style={{
              margin: "12px 20px 0",
              padding: "10px 14px",
              background: "color-mix(in srgb, var(--color-warning) 8%, transparent)",
              border: "1px solid color-mix(in srgb, var(--color-warning) 30%, transparent)",
              borderRadius: 8,
              fontSize: 12,
              color: "var(--color-warning)",
            }}
          >
            {pendingIndexers.map((o) => o.indexer_name).join(", ")} did not respond after several attempts. They may be down;
            use Search Again to retry.
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
        {!nothingYet && !error && !streaming && releases && releases.length === 0 && (
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
            {theaterCount > 0 ? "Only theater recordings (CAM/TS) found." : "No releases found."}
            {failedIndexers.length > 0 && (
              <span style={{ fontSize: 12, maxWidth: 520, textAlign: "center" }}>
                {failedIndexers.map((o) => `${o.indexer_name}: ${o.error}`).join(" · ")}
              </span>
            )}
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
        {!nothingYet && !error && releases && releases.length > 0 && (
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
                <th style={sortableThStyle("quality")} onClick={() => toggleSort("quality")}>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                    Quality {sortIcon("quality")}
                  </span>
                </th>
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
