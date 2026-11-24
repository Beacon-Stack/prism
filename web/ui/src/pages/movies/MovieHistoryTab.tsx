import { useMovieHistory } from "@/api/movies";
import ScoreChip from "@/components/ScoreChip";
import { formatBytes } from "@/lib/utils";

function statusColor(status: string): string {
  if (status === "completed") return "var(--color-success)";
  if (status === "failed" || status === "removed") return "var(--color-danger)";
  return "var(--color-text-muted)";
}

export default function HistoryTab({ movieId }: { movieId: string }) {
  const { data: items, isLoading, error } = useMovieHistory(movieId);

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: "8px 0" }}>
        {[1, 2, 3].map((i) => (
          <div key={i} className="skeleton" style={{ height: 48, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
        Failed to load history: {(error as Error).message}
      </p>
    );
  }

  if (!items?.length) {
    return (
      <div style={{ padding: "32px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
          No history
        </p>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          No grabs have been recorded for this movie yet.
        </p>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 1 }}>
      {items.map((item) => (
        <div
          key={item.id}
          style={{
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 6,
            padding: "10px 14px",
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 12,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-text-primary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
              title={item.release_title}
            >
              {item.release_title}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 4, flexWrap: "wrap" }}>
              {item.release_resolution && (
                <span
                  style={{
                    display: "inline-block",
                    padding: "1px 6px",
                    borderRadius: 4,
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                    background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
                    color: "var(--color-accent)",
                  }}
                >
                  {item.release_resolution}
                </span>
              )}
              <ScoreChip breakdown={item.score_breakdown} />
              <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                {item.protocol}
              </span>
              {item.size > 0 && (
                <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                  {formatBytes(item.size)}
                </span>
              )}
            </div>
          </div>
          <div style={{ flexShrink: 0, textAlign: "right" }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: statusColor(item.download_status), textTransform: "capitalize" }}>
              {item.download_status}
            </div>
            <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 2 }}>
              {new Date(item.grabbed_at).toLocaleDateString()}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
