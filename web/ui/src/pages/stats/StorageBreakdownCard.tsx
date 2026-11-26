import { useState } from "react";
import { Link } from "react-router-dom";
import type { StorageBreakdown, StorageForecast, StorageSlice } from "@/api/stats";
import { formatBytes } from "@/lib/utils";
import { Card, EmptyChart, StatBlock } from "./parts";

// Where the disk space goes: by resolution, genre or decade, the largest
// movies, and when the disks fill at the recent rate.

const GROUPS = [
  { key: "by_resolution", label: "Resolution" },
  { key: "by_genre", label: "Genre" },
  { key: "by_decade", label: "Decade" },
] as const;

type GroupKey = (typeof GROUPS)[number]["key"];

/** Bytes per hour of video, or null when no runtime is known. */
function perHour(s: StorageSlice): string | null {
  return s.runtime_minutes > 0 ? `${formatBytes(Math.round(s.timed_bytes / (s.runtime_minutes / 60)))}/h` : null;
}

/** "about 3 weeks", "about 14 months", "over 10 years". */
function roughly(days: number): string {
  if (days < 1) return "under a day";
  if (days < 14) return `about ${Math.round(days)} day${Math.round(days) === 1 ? "" : "s"}`;
  if (days < 60) return `about ${Math.round(days / 7)} weeks`;
  if (days < 730) return `about ${Math.round(days / 30.4)} months`;
  if (days < 3650) return `about ${Math.round(days / 365)} years`;
  return "over 10 years";
}

function ForecastBlock({ forecast }: { forecast: StorageForecast }) {
  const free = formatBytes(forecast.free_bytes);
  if (forecast.days_until_full === undefined) {
    return <StatBlock label="Free, and the library isn't growing" value={free} />;
  }
  const weekly = formatBytes(forecast.bytes_per_day * 7);
  const low = forecast.days_until_full < 60;
  return (
    <StatBlock
      label={`Until the disk is full: ${free} free, growing ${weekly} a week`}
      value={roughly(forecast.days_until_full)}
      accent={low ? "var(--color-warning)" : undefined}
    />
  );
}

const heading: React.CSSProperties = { fontSize: 11, color: "var(--color-text-muted)", marginBottom: 10 };

export default function StorageBreakdownCard({ data }: { data: StorageBreakdown }) {
  const [group, setGroup] = useState<GroupKey>("by_resolution");

  if (data.total.bytes === 0) {
    return (
      <Card title="Where the Space Goes">
        <EmptyChart message="Nothing on disk yet." />
      </Card>
    );
  }

  const slices = data[group];
  const max = Math.max(...slices.map((s) => s.bytes), 1);
  const density = perHour(data.total);

  return (
    <Card title="Where the Space Goes">
      <div style={{ display: "flex", gap: 32, flexWrap: "wrap", marginBottom: 24 }}>
        {density && <StatBlock label="Per hour of video" value={density} />}
        <StatBlock label="Average per movie" value={formatBytes(Math.round(data.total.bytes / Math.max(data.total.movies, 1)))} />
        {data.forecast && <ForecastBlock forecast={data.forecast} />}
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", gap: 32 }}>
        <div>
          <div role="group" aria-label="Group space by" style={{ display: "flex", gap: 6, marginBottom: 14 }}>
            {GROUPS.map((g) => {
              const active = g.key === group;
              return (
                <button
                  key={g.key}
                  type="button"
                  aria-pressed={active}
                  onClick={() => setGroup(g.key)}
                  style={{
                    padding: "4px 10px",
                    borderRadius: 6,
                    border: active ? "1px solid var(--color-accent)" : "1px solid var(--color-border-default)",
                    background: active ? "color-mix(in srgb, var(--color-accent) 12%, transparent)" : "transparent",
                    color: active ? "var(--color-accent)" : "var(--color-text-secondary)",
                    fontSize: 12,
                    fontWeight: 500,
                    cursor: "pointer",
                  }}
                >
                  {g.label}
                </button>
              );
            })}
          </div>

          <ul data-testid="storage-slices" style={{ listStyle: "none", margin: 0, padding: 0, display: "flex", flexDirection: "column", gap: 10 }}>
            {slices.map((s) => (
              <li key={s.label}>
                <div style={{ display: "flex", justifyContent: "space-between", gap: 12, fontSize: 12, marginBottom: 4 }}>
                  <span style={{ color: "var(--color-text-primary)", fontWeight: 500 }}>{s.label}</span>
                  <span style={{ color: "var(--color-text-muted)", whiteSpace: "nowrap" }}>
                    {formatBytes(s.bytes)} · {s.movies.toLocaleString()} {s.movies === 1 ? "movie" : "movies"}
                    {perHour(s) && ` · ${perHour(s)}`}
                  </span>
                </div>
                <div style={{ height: 6, borderRadius: 3, background: "var(--color-bg-subtle)" }}>
                  <div style={{ height: "100%", borderRadius: 3, width: `${(s.bytes / max) * 100}%`, background: "var(--color-accent)" }} />
                </div>
              </li>
            ))}
          </ul>
          {group === "by_genre" && (
            <p style={{ margin: "12px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
              A movie counts toward each of its genres, so these add up to more than the total.
            </p>
          )}
        </div>

        <div>
          <div style={heading}>Largest movies</div>
          <ol data-testid="largest-movies" style={{ listStyle: "none", margin: 0, padding: 0, display: "flex", flexDirection: "column", gap: 8 }}>
            {data.largest.map((m) => (
              <li key={m.movie_id} style={{ display: "flex", justifyContent: "space-between", gap: 12, fontSize: 13 }}>
                <span style={{ minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  <Link to={`/movies/${m.movie_id}`} style={{ color: "var(--color-text-primary)", textDecoration: "none", fontWeight: 500 }}>
                    {m.title}
                  </Link>
                  <span style={{ color: "var(--color-text-muted)" }}>
                    {m.year > 0 && ` ${m.year}`} · {m.resolution}
                  </span>
                </span>
                <span style={{ color: "var(--color-text-secondary)", whiteSpace: "nowrap", fontVariantNumeric: "tabular-nums" }}>{formatBytes(m.bytes)}</span>
              </li>
            ))}
          </ol>
        </div>
      </div>
    </Card>
  );
}
