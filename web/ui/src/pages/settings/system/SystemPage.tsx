import { useSystemHealth } from "@/api/system";
import PageHeader from "@/components/PageHeader";
import HealthCard from "@beacon-shared/HealthCard";
import SystemInfo from "@beacon-shared/SystemInfo";
import TasksCard from "@beacon-shared/TasksCard";

const LEVEL = { healthy: "ok", degraded: "warning", unhealthy: "error" } as const;

function HealthSection() {
  const { data, isLoading, error, refetch } = useSystemHealth();
  return (
    <HealthCard
      overall={data?.status}
      lines={data?.checks.map((c) => ({ name: c.name, level: LEVEL[c.status], message: c.message }))}
      loading={isLoading}
      failed={!!error}
      onCheck={async () => {
        const r = await refetch();
        if (r.error) throw r.error;
      }}
    />
  );
}

export default function SystemPage() {
  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <PageHeader title="System" description="What's running and how it's doing." />
      <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        <SystemInfo app="Prism" service="prism" image="ghcr.io/beacon-stack/prism" />
        <HealthSection />
        <TasksCard />
      </div>
    </div>
  );
}
