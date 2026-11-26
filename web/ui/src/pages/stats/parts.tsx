// Pieces every card on the Stats page is built from.

export const tooltipStyle = {
  contentStyle: {
    background: "var(--color-bg-elevated)",
    border: "1px solid var(--color-border-subtle)",
    borderRadius: 8,
    fontSize: 12,
    color: "var(--color-text-primary)",
  },
  // Disable the CSS transition so the tooltip snaps to cursor position
  // instead of animating/chasing it, which causes visible jitter.
  wrapperStyle: { transition: "none" },
  cursor: { fill: "color-mix(in srgb, var(--color-accent) 8%, transparent)" },
};

export const axisStyle = { fontSize: 11, fill: "var(--color-text-muted)" };

// Skeleton

export function CardSkeleton({ height = 220 }: { height?: number }) {
  return (
    <div
      className="skeleton"
      style={{ borderRadius: 12, height, background: "var(--color-bg-elevated)" }}
    />
  );
}

// Card shell

export function Card({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        background: "var(--color-bg-elevated)",
        borderRadius: 12,
        border: "1px solid var(--color-border-subtle)",
        padding: "20px 24px",
      }}
    >
      <h2
        style={{
          margin: "0 0 18px",
          fontSize: 13,
          fontWeight: 600,
          color: "var(--color-text-muted)",
          textTransform: "uppercase",
          letterSpacing: "0.08em",
        }}
      >
        {title}
      </h2>
      {children}
    </div>
  );
}

// Stat block

export function StatBlock({
  label,
  value,
  accent,
}: {
  label: string;
  value: string | number;
  accent?: string;
}) {
  return (
    <div style={{ flex: 1, minWidth: 100 }}>
      <div
        style={{
          fontSize: 28,
          fontWeight: 700,
          color: accent ?? "var(--color-text-primary)",
          lineHeight: 1,
          marginBottom: 6,
        }}
      >
        {value}
      </div>
      <div style={{ fontSize: 12, color: "var(--color-text-muted)", fontWeight: 500 }}>
        {label}
      </div>
    </div>
  );
}

// Empty state

export function EmptyChart({ message }: { message: string }) {
  return (
    <p style={{ color: "var(--color-text-muted)", fontSize: 13, margin: "8px 0 0" }}>
      {message}
    </p>
  );
}
