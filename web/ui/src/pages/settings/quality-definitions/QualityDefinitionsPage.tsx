import { useState, useEffect } from "react";
import { useQualityDefinitions, useUpdateQualityDefinitions } from "@/api/quality-definitions";
import type { QualityDefinition } from "@/types";

// ── Helpers ───────────────────────────────────────────────────────────────────

// Default size constraints seeded in the migration — used for Reset to Defaults.
const DEFAULTS: Record<string, { min: number; max: number }> = {
  "sd-dvd-xvid-none":        { min: 0,  max: 3   },
  "sd-hdtv-x264-none":       { min: 0,  max: 3   },
  "720p-hdtv-x264-none":     { min: 2,  max: 20  },
  "720p-webdl-x264-none":    { min: 2,  max: 20  },
  "720p-webrip-x264-none":   { min: 2,  max: 20  },
  "720p-bluray-x264-none":   { min: 2,  max: 30  },
  "1080p-hdtv-x264-none":    { min: 4,  max: 40  },
  "1080p-webdl-x264-none":   { min: 4,  max: 40  },
  "1080p-webrip-x265-none":  { min: 4,  max: 40  },
  "1080p-bluray-x265-none":  { min: 4,  max: 95  },
  "1080p-remux-x265-none":   { min: 17, max: 400 },
  "2160p-webdl-x265-hdr10":  { min: 15, max: 250 },
  "2160p-bluray-x265-hdr10": { min: 15, max: 250 },
  "2160p-remux-x265-hdr10":  { min: 35, max: 800 },
};

function resolutionBadgeColor(resolution: string): string {
  switch (resolution) {
    case "2160p": return "var(--color-accent)";
    case "1080p": return "var(--color-success)";
    case "720p":  return "var(--color-warning)";
    default:      return "var(--color-text-muted)";
  }
}

// ── Number input ──────────────────────────────────────────────────────────────

interface SizeInputProps {
  value: number;
  onChange: (v: number) => void;
  placeholder: string;
}

function SizeInput({ value, onChange, placeholder }: SizeInputProps) {
  const [focused, setFocused] = useState(false);
  return (
    <input
      type="number"
      min={0}
      step={0.1}
      value={value === 0 ? "" : value}
      placeholder={placeholder}
      onChange={(e) => {
        const v = parseFloat(e.target.value);
        onChange(isNaN(v) ? 0 : v);
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      style={{
        width: 90,
        background: "var(--color-bg-elevated)",
        border: `1px solid ${focused ? "var(--color-accent)" : "var(--color-border-default)"}`,
        borderRadius: 5,
        padding: "5px 8px",
        fontSize: 13,
        color: "var(--color-text-primary)",
        outline: "none",
        textAlign: "right",
        fontFamily: "var(--font-family-mono)",
      }}
    />
  );
}

// ── Table row ─────────────────────────────────────────────────────────────────

interface RowState {
  min: number;
  max: number;
}

interface DefinitionRowProps {
  def: QualityDefinition;
  row: RowState;
  isLast: boolean;
  onChange: (id: string, field: "min" | "max", value: number) => void;
  onReset: (id: string) => void;
}

function DefinitionRow({ def, row, isLast, onChange, onReset }: DefinitionRowProps) {
  const hasDefault = DEFAULTS[def.id] !== undefined;
  const defaultMatch =
    hasDefault &&
    row.min === DEFAULTS[def.id].min &&
    row.max === DEFAULTS[def.id].max;

  return (
    <tr style={{ borderBottom: isLast ? "none" : "1px solid var(--color-border-subtle)" }}>
      {/* Name */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle" }}>
        <span style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
          {def.name}
        </span>
      </td>

      {/* Resolution badge */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle", whiteSpace: "nowrap" }}>
        <span
          style={{
            display: "inline-block",
            background: `color-mix(in srgb, ${resolutionBadgeColor(def.resolution)} 15%, transparent)`,
            color: resolutionBadgeColor(def.resolution),
            borderRadius: 4,
            padding: "2px 7px",
            fontSize: 11,
            fontWeight: 600,
            letterSpacing: "0.04em",
            textTransform: "uppercase",
          }}
        >
          {def.resolution}
        </span>
      </td>

      {/* Source + codec */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle", fontSize: 12, color: "var(--color-text-secondary)", whiteSpace: "nowrap" }}>
        {def.source}
        {def.codec !== "unknown" && (
          <span style={{ marginLeft: 6, color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)" }}>
            {def.codec}
          </span>
        )}
      </td>

      {/* HDR */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle", fontSize: 12, color: "var(--color-text-muted)", whiteSpace: "nowrap" }}>
        {def.hdr !== "none" ? def.hdr : "—"}
      </td>

      {/* Min size */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <SizeInput
            value={row.min}
            onChange={(v) => onChange(def.id, "min", v)}
            placeholder="0"
          />
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>MB/min</span>
        </div>
      </td>

      {/* Max size */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <SizeInput
            value={row.max}
            onChange={(v) => onChange(def.id, "max", v)}
            placeholder="∞"
          />
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>MB/min</span>
        </div>
      </td>

      {/* Reset */}
      <td style={{ padding: "10px 20px", verticalAlign: "middle" }}>
        {hasDefault && !defaultMatch && (
          <button
            onClick={() => onReset(def.id)}
            style={{
              background: "none",
              border: "none",
              padding: "3px 8px",
              fontSize: 12,
              color: "var(--color-text-muted)",
              cursor: "pointer",
              borderRadius: 4,
            }}
            title="Reset to default"
          >
            Reset
          </button>
        )}
      </td>
    </tr>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function QualityDefinitionsPage() {
  const { data, isLoading, error } = useQualityDefinitions();
  const updateMutation = useUpdateQualityDefinitions();

  // Local editable state: id → { min, max }
  const [rows, setRows] = useState<Record<string, RowState>>({});
  const [dirty, setDirty] = useState(false);

  // Initialise / reset when data loads.
  useEffect(() => {
    if (!data) return;
    const initial: Record<string, RowState> = {};
    for (const d of data) {
      initial[d.id] = { min: d.min_size, max: d.max_size };
    }
    setRows(initial);
    setDirty(false);
  }, [data]);

  function handleChange(id: string, field: "min" | "max", value: number) {
    setRows((prev) => ({ ...prev, [id]: { ...prev[id], [field]: value } }));
    setDirty(true);
  }

  function handleReset(id: string) {
    const defaults = DEFAULTS[id];
    if (!defaults) return;
    setRows((prev) => ({ ...prev, [id]: { min: defaults.min, max: defaults.max } }));
    setDirty(true);
  }

  function handleResetAll() {
    if (!data) return;
    const reset: Record<string, RowState> = {};
    for (const d of data) {
      const def = DEFAULTS[d.id];
      reset[d.id] = def ? { min: def.min, max: def.max } : { min: d.min_size, max: d.max_size };
    }
    setRows(reset);
    setDirty(true);
  }

  async function handleSave() {
    if (!data) return;
    const updates = data.map((d) => ({
      id: d.id,
      min_size: rows[d.id]?.min ?? d.min_size,
      max_size: rows[d.id]?.max ?? d.max_size,
    }));
    await updateMutation.mutateAsync(updates);
    setDirty(false);
  }

  const defs = data ?? [];

  return (
    <div style={{ padding: 24, maxWidth: 1000, display: "flex", flexDirection: "column", gap: 24 }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
            Quality Definitions
          </h1>
          <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
            Set acceptable file-size ranges (MB per minute of runtime) for each quality level. Used to
            filter out suspiciously small or large releases.
          </p>
        </div>

        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <button
            onClick={handleResetAll}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "7px 14px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Reset All to Defaults
          </button>
          <button
            onClick={handleSave}
            disabled={!dirty || updateMutation.isPending}
            style={{
              background: dirty ? "var(--color-accent)" : "var(--color-bg-elevated)",
              border: `1px solid ${dirty ? "var(--color-accent)" : "var(--color-border-default)"}`,
              borderRadius: 6,
              padding: "7px 18px",
              fontSize: 13,
              fontWeight: 600,
              color: dirty ? "#fff" : "var(--color-text-muted)",
              cursor: dirty ? "pointer" : "default",
              transition: "background 0.15s, border-color 0.15s",
            }}
          >
            {updateMutation.isPending ? "Saving…" : "Save Changes"}
          </button>
        </div>
      </div>

      {/* Table card */}
      <div style={{ background: "var(--color-bg-surface)", border: "1px solid var(--color-border-subtle)", borderRadius: 8, boxShadow: "var(--shadow-card)", overflow: "hidden" }}>
        {isLoading ? (
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 16 }}>
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="skeleton" style={{ height: 14, borderRadius: 3 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 32, textAlign: "center", color: "var(--color-danger)", fontSize: 13 }}>
            Failed to load quality definitions. Please try again.
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                {[
                  { label: "Quality", note: "" },
                  { label: "Resolution", note: "" },
                  { label: "Source", note: "" },
                  { label: "HDR", note: "" },
                  { label: "Min Size", note: "MB/min" },
                  { label: "Max Size", note: "MB/min" },
                  { label: "", note: "" },
                ].map(({ label, note }) => (
                  <th
                    key={label || "_action"}
                    style={{
                      textAlign: "left",
                      padding: "8px 20px",
                      fontSize: 11,
                      fontWeight: 600,
                      letterSpacing: "0.08em",
                      textTransform: "uppercase",
                      color: "var(--color-text-muted)",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {label}
                    {note && (
                      <span style={{ marginLeft: 4, fontSize: 10, fontWeight: 400, letterSpacing: 0, textTransform: "none" }}>
                        ({note})
                      </span>
                    )}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {defs.map((def, idx) => (
                <DefinitionRow
                  key={def.id}
                  def={def}
                  row={rows[def.id] ?? { min: def.min_size, max: def.max_size }}
                  isLast={idx === defs.length - 1}
                  onChange={handleChange}
                  onReset={handleReset}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Help text */}
      {!isLoading && !error && defs.length > 0 && (
        <div style={{ fontSize: 12, color: "var(--color-text-muted)", lineHeight: 1.6 }}>
          <strong style={{ color: "var(--color-text-secondary)" }}>Min Size:</strong> Releases smaller than this (in MB per minute of runtime) are considered too small and may be low-quality fakes.
          Set to <code style={{ fontFamily: "var(--font-family-mono)" }}>0</code> for no minimum.{" "}
          <strong style={{ color: "var(--color-text-secondary)" }}>Max Size:</strong> Releases larger than this are excluded.
          Set to <code style={{ fontFamily: "var(--font-family-mono)" }}>0</code> for no limit.
        </div>
      )}
    </div>
  );
}
