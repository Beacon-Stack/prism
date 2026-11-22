import { useSystemHealth, useSystemStatus } from "@/api/system";
import type { HealthStatus } from "@/types";
import Pill from "@/components/Pill";
import PageHeader from "@/components/PageHeader";
import SystemCard from "@beacon-shared/SystemCard";
import SystemInfo from "@beacon-shared/SystemInfo";
import TasksCard from "@beacon-shared/TasksCard";

function healthColor(status: HealthStatus): string {
  if (status === "healthy") return "var(--color-success)";
  if (status === "degraded") return "var(--color-warning)";
  return "var(--color-danger)";
}

function HealthSection() {
  const { data, isLoading, error } = useSystemHealth();

  return (
    <SystemCard title="Health">
      {isLoading && <div className="skeleton" style={{ height: 40, margin: 16, borderRadius: 4 }} />}
      {error && (
        <div style={{ padding: 16, fontSize: 13, color: "var(--color-text-secondary)" }}>Failed to load health data.</div>
      )}
      {data && (
        <>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              padding: "12px 16px",
              borderBottom: data.checks.length ? "1px solid var(--color-border-subtle)" : "none",
              fontSize: 13,
              fontWeight: 500,
              color: healthColor(data.status),
            }}
          >
            <span style={{ width: 8, height: 8, borderRadius: "50%", background: "currentColor", flexShrink: 0 }} />
            {data.status === "healthy" && "All systems healthy"}
            {data.status === "degraded" && "Degraded"}
            {data.status === "unhealthy" && "Unhealthy"}
          </div>
          {data.checks.map((check, i) => (
            <div
              key={check.name}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "10px 16px",
                borderBottom: i < data.checks.length - 1 ? "1px solid var(--color-border-subtle)" : "none",
                fontSize: 13,
              }}
            >
              <span style={{ width: 6, height: 6, borderRadius: "50%", background: healthColor(check.status), flexShrink: 0 }} />
              <span style={{ color: "var(--color-text-primary)", fontWeight: 500, minWidth: 140 }}>{check.name}</span>
              <span style={{ color: "var(--color-text-secondary)", minWidth: 0, overflowWrap: "anywhere" }}>{check.message}</span>
            </div>
          ))}
        </>
      )}
    </SystemCard>
  );
}

export default function SystemPage() {
  const { data: status } = useSystemStatus();
  const extraRows = status
    ? [{ label: "TMDB", value: <Pill ok={status.tmdb_enabled} labelTrue="Configured" labelFalse="Not configured" /> }]
    : [];

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <PageHeader title="System" description="What's running and how it's doing." />
      <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        <SystemInfo app="Prism" service="prism" image="ghcr.io/beacon-stack/prism" extraRows={extraRows} />
        <HealthSection />
        <TasksCard />
      </div>
    </div>
  );
}
