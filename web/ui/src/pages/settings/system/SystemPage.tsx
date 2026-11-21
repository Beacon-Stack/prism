import { useState } from "react";
import {
  useSystemStatus,
  useSystemHealth,
  useTasks,
  useRunTask,
  useSystemLogs,
} from "@/api/system";
import { useMovies } from "@/api/movies";
import { useQueue } from "@/api/queue";
import type { HealthStatus, LogEntry } from "@/types";
import { card, sectionHeader } from "@/lib/styles";
import Pill from "@/components/Pill";
import TableScroll from "@beacon-shared/TableScroll";
import UpdateCheck from "@beacon-shared/UpdateCheck";

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(" ");
}

function healthColor(status: HealthStatus): string {
  if (status === "healthy") return "var(--color-success)";
  if (status === "degraded") return "var(--color-warning)";
  return "var(--color-danger)";
}

function SkeletonRow({ width = "100%", height = 16 }: { width?: string | number; height?: number }) {
  return (
    <div
      className="skeleton"
      style={{ width, height, borderRadius: 4, marginBottom: 8 }}
    />
  );
}

// ── Stats strip ───────────────────────────────────────────────────────────────

function StatsStrip() {
  const movies = useMovies({ per_page: 1 });
  const queue = useQueue();

  const stats: { label: string; value: string | number; loading: boolean }[] =
    [
      {
        label: "Total Movies",
        value: movies.data?.total ?? 0,
        loading: movies.isLoading,
      },
      {
        label: "Downloading",
        value: queue.data?.length ?? 0,
        loading: queue.isLoading,
      },
    ];

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))",
        gap: 12,
        marginBottom: 24,
      }}
    >
      {stats.map(({ label, value, loading }) => (
        <div
          key={label}
          style={{
            background: "var(--color-bg-surface)",
            border: "1px solid var(--color-border-subtle)",
            borderRadius: 8,
            padding: "16px 20px",
            boxShadow: "var(--shadow-card)",
          }}
        >
          <span
            style={{
              display: "block",
              fontSize: 11,
              fontWeight: 600,
              letterSpacing: "0.08em",
              textTransform: "uppercase",
              color: "var(--color-text-muted)",
              marginBottom: 6,
            }}
          >
            {label}
          </span>
          {loading ? (
            <div
              className="skeleton"
              style={{ height: 28, width: 60, borderRadius: 4 }}
            />
          ) : (
            <span
              style={{
                fontSize: 26,
                fontWeight: 700,
                color: "var(--color-text-primary)",
                letterSpacing: "-0.02em",
                lineHeight: 1,
              }}
            >
              {value}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

// ── Section 1: Status ─────────────────────────────────────────────────────────

function StatusSection() {
  const { data, isLoading } = useSystemStatus();
  return (
    <>
      <div style={card}>
        <p style={sectionHeader}>Status</p>
        {isLoading ? (
          <div>
            <SkeletonRow width="60%" />
            <SkeletonRow width="40%" />
            <SkeletonRow width="50%" />
            <SkeletonRow width="45%" />
          </div>
        ) : data ? (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "160px 1fr",
              rowGap: 10,
              fontSize: 13,
            }}
          >
            <span style={{ color: "var(--color-text-secondary)" }}>Application</span>
            <span style={{ color: "var(--color-text-primary)" }}>
              {data.app_name} {data.version}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Updates</span>
            <UpdateCheck app="Prism" service="prism" image="ghcr.io/beacon-stack/prism" />

            <span style={{ color: "var(--color-text-secondary)" }}>Go Version</span>
            <span style={{ color: "var(--color-text-primary)" }}>{data.go_version}</span>

            <span style={{ color: "var(--color-text-secondary)" }}>Build Time</span>
            <span style={{ color: "var(--color-text-primary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
              {data.build_time}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Database</span>
            <span style={{ color: "var(--color-text-primary)" }}>
              {data.db_type}
              {data.db_path && (
                <span style={{ display: "block", fontSize: 11, fontFamily: "var(--font-family-mono)", color: "var(--color-text-muted)", marginTop: 2 }}>
                  {data.db_path}
                </span>
              )}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>Uptime</span>
            <span style={{ color: "var(--color-text-primary)" }}>{formatUptime(data.uptime_seconds)}</span>

            <span style={{ color: "var(--color-text-secondary)" }}>Started</span>
            <span style={{ color: "var(--color-text-primary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
              {data.start_time}
            </span>

            <span style={{ color: "var(--color-text-secondary)" }}>TMDB</span>
            <Pill ok={data.tmdb_enabled} labelTrue="Configured" labelFalse="Not configured" />
          </div>
        ) : null}
      </div>
    </>
  );
}

// ── Section 2: Health ─────────────────────────────────────────────────────────

function HealthSection() {
  const { data, isLoading, error } = useSystemHealth();

  return (
    <div style={card}>
      <p style={sectionHeader}>Health</p>
      {isLoading ? (
        <div>
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
        </div>
      ) : error ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          Failed to load health data.
        </p>
      ) : data ? (
        <div>
          {/* Overall status */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              marginBottom: 16,
              fontSize: 13,
              fontWeight: 500,
              color: healthColor(data.status),
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: "currentColor",
                flexShrink: 0,
              }}
            />
            {data.status === "healthy" && "All systems healthy"}
            {data.status === "degraded" && "Degraded"}
            {data.status === "unhealthy" && "Unhealthy"}
          </div>

          {/* Individual checks */}
          {data.checks.map((check, i) => (
            <div
              key={check.name}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                height: 40,
                borderBottom:
                  i < data.checks.length - 1
                    ? "1px solid var(--color-border-subtle)"
                    : "none",
              }}
            >
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: "50%",
                  background: healthColor(check.status),
                  flexShrink: 0,
                }}
              />
              <span style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500, minWidth: 140 }}>
                {check.name}
              </span>
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)" }}>
                {check.message}
              </span>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

// ── Section 3: Tasks ──────────────────────────────────────────────────────────

function TasksSection() {
  const { data, isLoading } = useTasks();
  const runTask = useRunTask();
  const [triggered, setTriggered] = useState<string | null>(null);

  function handleRun(name: string) {
    runTask.mutate(name, {
      onSuccess: () => {
        setTriggered(name);
        setTimeout(() => setTriggered((prev) => (prev === name ? null : prev)), 2000);
      },
    });
  }

  return (
    <div style={card}>
      <p style={sectionHeader}>Tasks</p>
      {isLoading ? (
        <div>
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
          <SkeletonRow height={32} />
        </div>
      ) : !data?.length ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          No tasks registered.
        </p>
      ) : (
        <TableScroll minWidth={500}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr>
                {["Task", "Interval", ""].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: "left",
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: "0.08em",
                      textTransform: "uppercase",
                      color: "var(--color-text-muted)",
                      paddingBottom: 8,
                      borderBottom: "1px solid var(--color-border-subtle)",
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
          <tbody>
            {data.map((task, i) => {
              const isPending = runTask.isPending && runTask.variables === task.name;
              const wasTriggered = triggered === task.name;

              return (
                <tr key={task.name}>
                  <td
                    style={{
                      height: 44,
                      color: "var(--color-text-primary)",
                      fontWeight: 500,
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                      paddingRight: 16,
                    }}
                  >
                    {task.name}
                  </td>
                  <td
                    style={{
                      height: 44,
                      color: "var(--color-text-secondary)",
                      fontFamily: "var(--font-family-mono)",
                      fontSize: 12,
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                      paddingRight: 16,
                    }}
                  >
                    {task.interval}
                  </td>
                  <td
                    style={{
                      height: 44,
                      textAlign: "right",
                      borderBottom:
                        i < data.length - 1
                          ? "1px solid var(--color-border-subtle)"
                          : "none",
                    }}
                  >
                    {wasTriggered ? (
                      <span style={{ fontSize: 12, color: "var(--color-success)" }}>
                        Triggered ✓
                      </span>
                    ) : (
                      <button
                        disabled={isPending}
                        onClick={() => handleRun(task.name)}
                        style={{
                          background: "var(--color-bg-elevated)",
                          border: "1px solid var(--color-border-default)",
                          color: isPending ? "var(--color-text-muted)" : "var(--color-text-secondary)",
                          borderRadius: 6,
                          padding: "4px 12px",
                          fontSize: 12,
                          cursor: isPending ? "not-allowed" : "pointer",
                        }}
                        onMouseEnter={(e) => {
                          if (!isPending) {
                            (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-subtle)";
                            (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
                          }
                        }}
                        onMouseLeave={(e) => {
                          (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
                          (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
                        }}
                      >
                        {isPending ? "Running…" : "Run Now"}
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
          </table>
        </TableScroll>
      )}
    </div>
  );
}

// ── Section 4: Logs ──────────────────────────────────────────────────────────

const LOG_LEVELS = ["all", "debug", "info", "warn", "error"] as const;

function levelBadgeColor(level: string): { color: string; bg: string } {
  switch (level) {
    case "DEBUG":
      return { color: "var(--color-text-muted)", bg: "var(--color-bg-subtle)" };
    case "INFO":
      return { color: "var(--color-accent)", bg: "color-mix(in srgb, var(--color-accent) 12%, transparent)" };
    case "WARN":
      return { color: "var(--color-warning)", bg: "color-mix(in srgb, var(--color-warning) 12%, transparent)" };
    case "ERROR":
      return { color: "var(--color-danger)", bg: "color-mix(in srgb, var(--color-danger) 12%, transparent)" };
    default:
      return { color: "var(--color-text-secondary)", bg: "var(--color-bg-subtle)" };
  }
}

function formatLogTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

function LogRow({ entry, isLast }: { entry: LogEntry; isLast: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const hasFields = entry.fields && Object.keys(entry.fields).length > 0;
  const badge = levelBadgeColor(entry.level);

  return (
    <>
      <tr
        onClick={hasFields ? () => setExpanded(!expanded) : undefined}
        style={{ cursor: hasFields ? "pointer" : "default" }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLTableRowElement).style.background = "var(--color-bg-subtle)";
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLTableRowElement).style.background = "transparent";
        }}
      >
        <td
          style={{
            height: 36,
            fontSize: 12,
            fontFamily: "var(--font-family-mono)",
            color: "var(--color-text-muted)",
            whiteSpace: "nowrap",
            paddingRight: 12,
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
          }}
        >
          {formatLogTime(entry.time)}
        </td>
        <td
          style={{
            height: 36,
            paddingRight: 12,
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
          }}
        >
          <span
            style={{
              display: "inline-block",
              padding: "1px 6px",
              borderRadius: 3,
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: "0.04em",
              color: badge.color,
              background: badge.bg,
              minWidth: 42,
              textAlign: "center",
            }}
          >
            {entry.level}
          </span>
        </td>
        <td
          style={{
            height: 36,
            fontSize: 13,
            color: "var(--color-text-primary)",
            borderBottom: !expanded && !isLast ? "1px solid var(--color-border-subtle)" : "none",
            verticalAlign: "middle",
            maxWidth: 0,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {entry.message}
          {hasFields && (
            <span style={{ color: "var(--color-text-muted)", fontSize: 11, marginLeft: 6 }}>
              {expanded ? "▾" : "▸"}
            </span>
          )}
        </td>
      </tr>
      {expanded && hasFields && (
        <tr>
          <td colSpan={3} style={{ borderBottom: !isLast ? "1px solid var(--color-border-subtle)" : "none", padding: 0 }}>
            <pre
              style={{
                margin: 0,
                padding: "8px 12px 8px 40px",
                fontSize: 11,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-text-secondary)",
                background: "var(--color-bg-elevated)",
                borderRadius: 0,
                whiteSpace: "pre-wrap",
                wordBreak: "break-all",
              }}
            >
              {JSON.stringify(entry.fields, null, 2)}
            </pre>
          </td>
        </tr>
      )}
    </>
  );
}

function LogsSection() {
  const [level, setLevel] = useState<string>("all");
  const filterLevel = level === "all" ? undefined : level;
  const { data, isLoading, error } = useSystemLogs(filterLevel, 200);

  return (
    <div style={card}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
        <p style={{ ...sectionHeader, marginBottom: 0 }}>Logs</p>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Level:</span>
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 4,
              padding: "2px 6px",
              fontSize: 12,
              color: "var(--color-text-primary)",
              cursor: "pointer",
            }}
          >
            {LOG_LEVELS.map((l) => (
              <option key={l} value={l}>
                {l === "all" ? "All" : l.toUpperCase()}
              </option>
            ))}
          </select>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
            Auto-refresh 10s
          </span>
        </div>
      </div>
      {isLoading ? (
        <div>
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
          <SkeletonRow height={28} />
        </div>
      ) : error ? (
        <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
          Failed to load logs.
        </p>
      ) : !data?.length ? (
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
          No log entries{filterLevel ? ` at ${filterLevel.toUpperCase()} level` : ""}.
        </p>
      ) : (
        <div style={{ maxHeight: 420, overflowY: "auto" }}>
          <TableScroll minWidth={600}>
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr>
                {["Time", "Level", "Message"].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: "left",
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: "0.08em",
                      textTransform: "uppercase",
                      color: "var(--color-text-muted)",
                      paddingBottom: 8,
                      borderBottom: "1px solid var(--color-border-subtle)",
                      position: "sticky",
                      top: 0,
                      background: "var(--color-bg-surface)",
                      zIndex: 1,
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((entry, i) => (
                <LogRow key={`${entry.time}-${i}`} entry={entry} isLast={i === data.length - 1} />
              ))}
            </tbody>
          </table>
          </TableScroll>
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function SystemPage() {
  return (
    <div style={{ padding: 24, maxWidth: 800 }}>
      <div style={{ marginBottom: 24 }}>
        <h1
          style={{
            fontSize: 20,
            fontWeight: 600,
            color: "var(--color-text-primary)",
            margin: 0,
            marginBottom: 4,
            letterSpacing: "-0.01em",
          }}
        >
          System
        </h1>
        <p style={{ fontSize: 13, color: "var(--color-text-secondary)", margin: 0 }}>
          Runtime status, health checks, and configuration.
        </p>
      </div>

      <StatsStrip />
      <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
        <StatusSection />
        <HealthSection />
        <TasksSection />
        <LogsSection />
      </div>
    </div>
  );
}
