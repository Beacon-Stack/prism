import { useState } from "react";
import { toast } from "sonner";
import { useUpdateMovie } from "@/api/movies";
import { useQualityProfiles } from "@/api/quality-profiles";
import Modal from "@beacon-shared/Modal";
import type { Movie } from "@/types";

type MonitoredChoice = "unchanged" | "yes" | "no";

export default function BulkEditModal({
  movies,
  selectedIds,
  onClose,
}: {
  movies: Movie[];
  selectedIds: Set<string>;
  onClose: () => void;
}) {
  const { data: profiles } = useQualityProfiles();
  const updateMovie = useUpdateMovie();
  const [monitoredVal, setMonitoredVal] = useState<MonitoredChoice>("unchanged");
  const [profileId, setProfileId] = useState("");
  const [minAvail, setMinAvail] = useState("");
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null);

  async function handleApply() {
    const targets = movies.filter((m) => selectedIds.has(m.id));
    setProgress({ done: 0, total: targets.length });
    let done = 0;
    let failed = 0;
    for (const m of targets) {
      try {
        await updateMovie.mutateAsync({
          id: m.id,
          title: m.title,
          monitored: monitoredVal === "unchanged" ? m.monitored : monitoredVal === "yes",
          library_id: m.library_id,
          quality_profile_id: profileId || m.quality_profile_id,
          minimum_availability: minAvail || m.minimum_availability,
        });
        done++;
      } catch {
        failed++;
      }
      setProgress({ done: done + failed, total: targets.length });
    }
    if (failed > 0) {
      toast.error(`${failed} update${failed > 1 ? "s" : ""} failed`);
    } else {
      toast.success(`Updated ${done} ${done === 1 ? "movie" : "movies"}`);
    }
    onClose();
  }

  const count = selectedIds.size;

  return (
    <Modal onClose={onClose} width={420} maxWidth="calc(100vw - 32px)">
        {/* Header */}
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            padding: "16px 20px",
            borderBottom: "1px solid var(--color-border-subtle)",
          }}
        >
          <span
            style={{
              fontSize: 15,
              fontWeight: 600,
              color: "var(--color-text-primary)",
            }}
          >
            Edit {count} {count === 1 ? "movie" : "movies"}
          </span>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              fontSize: 16,
              color: "var(--color-text-muted)",
              cursor: "pointer",
              padding: 4,
              lineHeight: 1,
            }}
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div
          style={{
            padding: 20,
            display: "flex",
            flexDirection: "column",
            gap: 16,
          }}
        >
          {/* Monitored */}
          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 600,
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                marginBottom: 8,
              }}
            >
              Monitored
            </label>
            <div style={{ display: "flex", gap: 6 }}>
              {(["unchanged", "yes", "no"] as MonitoredChoice[]).map((v) => (
                <button
                  key={v}
                  onClick={() => setMonitoredVal(v)}
                  style={{
                    padding: "6px 14px",
                    borderRadius: 6,
                    fontSize: 13,
                    cursor: "pointer",
                    border: `1px solid ${monitoredVal === v ? "var(--color-accent)" : "var(--color-border-default)"}`,
                    background:
                      monitoredVal === v
                        ? "color-mix(in srgb, var(--color-accent) 15%, transparent)"
                        : "var(--color-bg-elevated)",
                    color:
                      monitoredVal === v
                        ? "var(--color-accent)"
                        : "var(--color-text-secondary)",
                    fontWeight: monitoredVal === v ? 600 : 400,
                  }}
                >
                  {v.charAt(0).toUpperCase() + v.slice(1)}
                </button>
              ))}
            </div>
          </div>

          {/* Quality Profile */}
          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 600,
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                marginBottom: 8,
              }}
            >
              Quality Profile
            </label>
            <select
              value={profileId}
              onChange={(e) => setProfileId(e.currentTarget.value)}
              style={{
                width: "100%",
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "7px 11px",
                fontSize: 13,
                color: "var(--color-text-primary)",
                outline: "none",
                cursor: "pointer",
              }}
            >
              <option value="">Unchanged</option>
              {profiles?.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>

          {/* Min Availability */}
          <div>
            <label
              style={{
                display: "block",
                fontSize: 12,
                fontWeight: 600,
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                marginBottom: 8,
              }}
            >
              Minimum Availability
            </label>
            <select
              value={minAvail}
              onChange={(e) => setMinAvail(e.currentTarget.value)}
              style={{
                width: "100%",
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                padding: "7px 11px",
                fontSize: 13,
                color: "var(--color-text-primary)",
                outline: "none",
                cursor: "pointer",
              }}
            >
              <option value="">Unchanged</option>
              <option value="announced">Announced</option>
              <option value="in_cinemas">In Cinemas</option>
              <option value="released">Released</option>
              <option value="tba">TBA</option>
            </select>
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            gap: 10,
            padding: "14px 20px",
            borderTop: "1px solid var(--color-border-subtle)",
          }}
        >
          {progress && (
            <span
              style={{
                fontSize: 12,
                color: "var(--color-text-muted)",
                marginRight: "auto",
              }}
            >
              Updating {progress.done}/{progress.total}…
            </span>
          )}
          <button
            onClick={onClose}
            disabled={!!progress && progress.done < progress.total}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleApply}
            disabled={!!progress}
            style={{
              background: progress ? "var(--color-bg-subtle)" : "var(--color-accent)",
              color: progress ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 18px",
              fontSize: 13,
              fontWeight: 500,
              cursor: progress ? "not-allowed" : "pointer",
            }}
          >
            {progress ? "Applying…" : "Apply"}
          </button>
        </div>
    </Modal>
  );
}
