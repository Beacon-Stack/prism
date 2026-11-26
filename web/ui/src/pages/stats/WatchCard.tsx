import { Link } from "react-router-dom";
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { WatchStats } from "@/api/stats";
import { axisStyle, Card, StatBlock, tooltipStyle } from "./parts";

// What has been watched, from the history synced from media servers.

/** 95 → "1h 35m", 3000 → "2d 2h". */
function watchTime(minutes: number): string {
  const days = Math.floor(minutes / 1440);
  const hours = Math.floor((minutes % 1440) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  return hours > 0 ? `${hours}h ${minutes % 60}m` : `${minutes}m`;
}

/** "2026-11" → "Nov". */
function monthLabel(month: string): string {
  const [y, m] = month.split("-").map(Number);
  return new Date(y, m - 1, 1).toLocaleDateString(undefined, { month: "short" });
}

function monthYear(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: "short", year: "numeric" });
}

const heading: React.CSSProperties = { fontSize: 11, color: "var(--color-text-muted)", marginBottom: 10 };
const list: React.CSSProperties = { listStyle: "none", margin: 0, padding: 0, display: "flex", flexDirection: "column", gap: 8 };
const row: React.CSSProperties = { display: "flex", justifyContent: "space-between", gap: 12, fontSize: 13 };
const title: React.CSSProperties = { minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" };
const movieLink: React.CSSProperties = { color: "var(--color-text-primary)", textDecoration: "none", fontWeight: 500 };
const aside: React.CSSProperties = { color: "var(--color-text-secondary)", whiteSpace: "nowrap" };

export default function WatchCard({ data }: { data: WatchStats }) {
  if (data.plays === 0) {
    return (
      <Card title="Watching">
        <p style={{ color: "var(--color-text-muted)", fontSize: 13, margin: "8px 0 0" }}>
          No watch history yet. Connect Plex, Jellyfin or Emby in{" "}
          <Link to="/settings/media-servers" style={{ color: "var(--color-accent)", textDecoration: "none", fontWeight: 500 }}>
            Settings → Media Servers
          </Link>{" "}
          and what gets watched shows up here.
        </p>
      </Card>
    );
  }

  const share = data.owned > 0 ? Math.round((data.watched / data.owned) * 100) : 0;
  const months = data.by_month.map((m) => ({ ...m, label: monthLabel(m.month) }));

  return (
    <Card title="Watching">
      <div style={{ display: "flex", gap: 32, flexWrap: "wrap", marginBottom: 24 }}>
        <StatBlock label={`Watched: ${data.watched.toLocaleString()} of ${data.owned.toLocaleString()} movies`} value={`${share}%`} />
        <StatBlock label="Plays" value={data.plays.toLocaleString()} />
        <StatBlock label="Time watched" value={watchTime(data.minutes_watched)} />
      </div>

      <div style={heading}>Plays per month</div>
      <ResponsiveContainer width="100%" height={140}>
        <BarChart data={months} margin={{ top: 4, right: 0, left: -32, bottom: 0 }}>
          <XAxis dataKey="label" tick={axisStyle} axisLine={false} tickLine={false} />
          <YAxis tick={axisStyle} axisLine={false} tickLine={false} allowDecimals={false} />
          <Tooltip
            contentStyle={tooltipStyle.contentStyle}
            wrapperStyle={tooltipStyle.wrapperStyle}
            cursor={tooltipStyle.cursor}
            formatter={(v: number | undefined) => [(v ?? 0).toLocaleString(), "Plays"]}
          />
          <Bar dataKey="plays" fill="var(--color-accent)" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 32, marginTop: 24 }}>
        <div>
          <div style={heading}>Most watched</div>
          <ol data-testid="most-watched" style={list}>
            {data.most_watched.map((m) => (
              <li key={m.movie_id} style={row}>
                <span style={title}>
                  <Link to={`/movies/${m.movie_id}`} style={movieLink}>
                    {m.title}
                  </Link>
                  {m.year > 0 && <span style={{ color: "var(--color-text-muted)" }}> {m.year}</span>}
                </span>
                <span style={aside}>
                  {m.plays} {m.plays === 1 ? "play" : "plays"} · last {monthYear(m.last_watched_at)}
                </span>
              </li>
            ))}
          </ol>
        </div>

        <div>
          <div style={heading}>Waiting longest to be watched</div>
          {data.longest_unwatched.length === 0 ? (
            <p style={{ margin: 0, fontSize: 13, color: "var(--color-text-muted)" }}>Nothing. Every movie here has been watched.</p>
          ) : (
            <ol data-testid="longest-unwatched" style={list}>
              {data.longest_unwatched.map((m) => (
                <li key={m.movie_id} style={row}>
                  <span style={title}>
                    <Link to={`/movies/${m.movie_id}`} style={movieLink}>
                      {m.title}
                    </Link>
                    {m.year > 0 && <span style={{ color: "var(--color-text-muted)" }}> {m.year}</span>}
                  </span>
                  <span style={aside}>added {monthYear(m.added_at)}</span>
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </Card>
  );
}
