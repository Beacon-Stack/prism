import { useState } from "react";
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader2,
  RefreshCw,
  ArrowLeft,
} from "lucide-react";
import { useMediaServers } from "@/api/mediaservers";
import { usePlexSections, usePlexSyncPreview, usePlexSyncImport } from "@/api/plexsync";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import type {
  MediaServerConfig,
  PlexSection,
  PlexSyncPreviewResult,
  PlexSyncMovie,
  PrismSyncMovie,
  PlexSyncImportResult,
} from "@/types";

// ── Shared styles ─────────────────────────────────────────────────────────────

const card: React.CSSProperties = {
  background: "var(--color-bg-surface)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 8,
  padding: 24,
};

const btnPrimary: React.CSSProperties = {
  padding: "8px 20px",
  background: "var(--color-accent)",
  color: "var(--color-accent-fg)",
  border: "none",
  borderRadius: 6,
  fontSize: 13,
  fontWeight: 500,
  cursor: "pointer",
  display: "inline-flex",
  alignItems: "center",
  gap: 8,
};

const btnSecondary: React.CSSProperties = {
  padding: "8px 20px",
  background: "transparent",
  color: "var(--color-text-secondary)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  fontSize: 13,
  fontWeight: 500,
  cursor: "pointer",
};

const selectStyle: React.CSSProperties = {
  width: "100%",
  padding: "8px 12px",
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  color: "var(--color-text-primary)",
  fontSize: 13,
  outline: "none",
  boxSizing: "border-box",
};

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
  letterSpacing: "0.02em",
};

const badge = (bg: string, color: string): React.CSSProperties => ({
  display: "inline-block",
  padding: "2px 8px",
  borderRadius: 4,
  fontSize: 11,
  fontWeight: 500,
  background: bg,
  color,
});

// ── Step 1: Select server + section ──────────────────────────────────────────

function SelectStep({
  onPreview,
}: {
  onPreview: (serverID: string, preview: PlexSyncPreviewResult) => void;
}) {
  const { data: servers, isLoading: loadingServers } = useMediaServers();
  const sectionsMut = usePlexSections();
  const previewMut = usePlexSyncPreview();

  const [serverID, setServerID] = useState("");
  const [sections, setSections] = useState<PlexSection[]>([]);
  const [sectionKey, setSectionKey] = useState("");

  const plexServers = (servers ?? []).filter((s: MediaServerConfig) => s.kind === "plex");

  function handleServerChange(id: string) {
    setServerID(id);
    setSections([]);
    setSectionKey("");
    if (!id) return;
    sectionsMut.mutate(id, {
      onSuccess: (data) => {
        setSections(data);
        if (data.length === 1) setSectionKey(data[0].key);
      },
    });
  }

  function handleCompare() {
    if (!serverID || !sectionKey) return;
    previewMut.mutate(
      { id: serverID, sectionKey },
      { onSuccess: (result) => onPreview(serverID, result) },
    );
  }

  const isLoading = sectionsMut.isPending || previewMut.isPending;

  return (
    <div style={{ ...card, maxWidth: 500 }}>
      <div style={{ fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)", marginBottom: 20 }}>
        Compare Libraries
      </div>

      <div style={{ marginBottom: 16 }}>
        <label style={labelStyle}>Media Server</label>
        {loadingServers ? (
          <div className="skeleton" style={{ height: 36, borderRadius: 6 }} />
        ) : plexServers.length === 0 ? (
          <div style={{ fontSize: 12, color: "var(--color-text-muted)", padding: "8px 0" }}>
            No media servers configured. Add one in Settings → Media Servers.
          </div>
        ) : (
          <select
            style={selectStyle}
            value={serverID}
            onChange={(e) => handleServerChange(e.target.value)}
            disabled={isLoading}
          >
            <option value="">Select a server…</option>
            {plexServers.map((s: MediaServerConfig) => (
              <option key={s.id} value={s.id}>{s.name}</option>
            ))}
          </select>
        )}
      </div>

      {sections.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          <label style={labelStyle}>Library Section</label>
          <select
            style={selectStyle}
            value={sectionKey}
            onChange={(e) => setSectionKey(e.target.value)}
            disabled={isLoading}
          >
            <option value="">Select a section…</option>
            {sections.map((s) => (
              <option key={s.key} value={s.key}>{s.title}</option>
            ))}
          </select>
        </div>
      )}

      {(sectionsMut.error || previewMut.error) && (
        <div
          style={{
            padding: "8px 12px",
            background: "rgba(239,68,68,0.08)",
            border: "1px solid var(--color-danger)",
            borderRadius: 6,
            fontSize: 12,
            color: "var(--color-danger)",
            marginBottom: 16,
          }}
        >
          {(sectionsMut.error ?? previewMut.error)?.message}
        </div>
      )}

      <button
        style={btnPrimary}
        onClick={handleCompare}
        disabled={!serverID || !sectionKey || isLoading}
      >
        {isLoading && <Loader2 size={14} style={{ animation: "spin 1s linear infinite" }} />}
        {previewMut.isPending ? "Comparing…" : "Compare"}
      </button>
    </div>
  );
}

// ── Step 2: Diff view ────────────────────────────────────────────────────────

function DiffView({
  serverID,
  preview,
  onImportDone,
  onBack,
}: {
  serverID: string;
  preview: PlexSyncPreviewResult;
  onImportDone: (result: PlexSyncImportResult) => void;
  onBack: () => void;
}) {
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const importMut = usePlexSyncImport();

  const [selected, setSelected] = useState<Set<number>>(
    () => new Set(preview.in_plex_only.map((m) => m.tmdb_id)),
  );
  const [libraryID, setLibraryID] = useState("");
  const [profileID, setProfileID] = useState("");
  const [tab, setTab] = useState<"plex" | "prism">("plex");

  function toggleAll() {
    if (selected.size === preview.in_plex_only.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(preview.in_plex_only.map((m) => m.tmdb_id)));
    }
  }

  function toggle(id: number) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelected(next);
  }

  function handleImport() {
    if (!libraryID || !profileID || selected.size === 0) return;
    importMut.mutate(
      {
        id: serverID,
        tmdb_ids: [...selected],
        library_id: libraryID,
        quality_profile_id: profileID,
        monitored: true,
      },
      { onSuccess: (result) => onImportDone(result) },
    );
  }

  const summaryItems = [
    { label: "In sync", value: preview.already_synced, color: "var(--color-success)" },
    { label: "Server only", value: preview.in_plex_only.length, color: "var(--color-accent)" },
    { label: "Prism only", value: preview.in_prism_only.length, color: "var(--color-warning)" },
    { label: "Unmatched", value: preview.unmatched, color: "var(--color-text-muted)" },
  ];

  return (
    <div>
      <button
        onClick={onBack}
        style={{
          background: "none",
          border: "none",
          cursor: "pointer",
          display: "flex",
          alignItems: "center",
          gap: 6,
          color: "var(--color-text-secondary)",
          fontSize: 13,
          marginBottom: 16,
          padding: 0,
        }}
      >
        <ArrowLeft size={14} /> Back
      </button>

      {/* Summary bar */}
      <div
        style={{
          ...card,
          display: "flex",
          gap: 24,
          marginBottom: 20,
          padding: "16px 24px",
          flexWrap: "wrap",
        }}
      >
        <div style={{ fontSize: 12, color: "var(--color-text-muted)", marginRight: "auto" }}>
          <span style={{ fontWeight: 600 }}>{preview.plex_total}</span> movies on server
        </div>
        {summaryItems.map((s) => (
          <div key={s.label} style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>
            <span style={{ fontWeight: 700, color: s.color, marginRight: 4 }}>{s.value}</span>
            {s.label}
          </div>
        ))}
      </div>

      {/* Tab toggle */}
      <div style={{ display: "flex", gap: 0, marginBottom: 16 }}>
        {(["plex", "prism"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
              background: tab === t ? "var(--color-bg-surface)" : "transparent",
              color: tab === t ? "var(--color-text-primary)" : "var(--color-text-muted)",
              border: `1px solid ${tab === t ? "var(--color-border-default)" : "transparent"}`,
              borderBottom: tab === t ? "1px solid var(--color-bg-surface)" : "1px solid var(--color-border-default)",
              borderRadius: t === "plex" ? "6px 0 0 0" : "0 6px 0 0",
            }}
          >
            {t === "plex"
              ? `Server Only (${preview.in_plex_only.length})`
              : `Prism Only (${preview.in_prism_only.length})`}
          </button>
        ))}
        <div style={{ flex: 1, borderBottom: "1px solid var(--color-border-default)" }} />
      </div>

      {/* Plex-only tab */}
      {tab === "plex" && (
        <div style={card}>
          {preview.in_plex_only.length === 0 ? (
            <div style={{ fontSize: 13, color: "var(--color-text-muted)", textAlign: "center", padding: 24 }}>
              <CheckCircle size={24} style={{ marginBottom: 8, opacity: 0.5 }} />
              <div>All server movies are already in Prism.</div>
            </div>
          ) : (
            <>
              {/* Import controls */}
              <div
                style={{
                  display: "flex",
                  gap: 12,
                  alignItems: "flex-end",
                  marginBottom: 16,
                  flexWrap: "wrap",
                }}
              >
                <div style={{ minWidth: 180 }}>
                  <label style={labelStyle}>Library</label>
                  <select style={selectStyle} value={libraryID} onChange={(e) => setLibraryID(e.target.value)}>
                    <option value="">Select…</option>
                    {(libraries ?? []).map((l) => (
                      <option key={l.id} value={l.id}>{l.name}</option>
                    ))}
                  </select>
                </div>
                <div style={{ minWidth: 180 }}>
                  <label style={labelStyle}>Quality Profile</label>
                  <select style={selectStyle} value={profileID} onChange={(e) => setProfileID(e.target.value)}>
                    <option value="">Select…</option>
                    {(profiles ?? []).map((p) => (
                      <option key={p.id} value={p.id}>{p.name}</option>
                    ))}
                  </select>
                </div>
                <button
                  style={btnPrimary}
                  onClick={handleImport}
                  disabled={!libraryID || !profileID || selected.size === 0 || importMut.isPending}
                >
                  {importMut.isPending && (
                    <Loader2 size={14} style={{ animation: "spin 1s linear infinite" }} />
                  )}
                  Import {selected.size} movie{selected.size !== 1 ? "s" : ""}
                </button>
              </div>

              {/* Select all */}
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                  padding: "8px 0",
                  borderBottom: "1px solid var(--color-border-default)",
                  marginBottom: 4,
                }}
              >
                <input
                  type="checkbox"
                  checked={selected.size === preview.in_plex_only.length}
                  onChange={toggleAll}
                  style={{ accentColor: "var(--color-accent)" }}
                />
                <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>
                  Select all ({preview.in_plex_only.length})
                </span>
              </div>

              {/* Movie list */}
              <div style={{ maxHeight: 400, overflowY: "auto" }}>
                {preview.in_plex_only
                  .sort((a, b) => a.title.localeCompare(b.title))
                  .map((m: PlexSyncMovie) => (
                  <label
                    key={m.tmdb_id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      padding: "6px 0",
                      borderBottom: "1px solid var(--color-border-default)",
                      cursor: "pointer",
                      fontSize: 13,
                      color: "var(--color-text-primary)",
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={selected.has(m.tmdb_id)}
                      onChange={() => toggle(m.tmdb_id)}
                      style={{ accentColor: "var(--color-accent)" }}
                    />
                    <span style={{ flex: 1 }}>{m.title}</span>
                    <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{m.year}</span>
                  </label>
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {/* Prism-only tab */}
      {tab === "prism" && (
        <div style={card}>
          {preview.in_prism_only.length === 0 ? (
            <div style={{ fontSize: 13, color: "var(--color-text-muted)", textAlign: "center", padding: 24 }}>
              <CheckCircle size={24} style={{ marginBottom: 8, opacity: 0.5 }} />
              <div>All Prism movies are on the server.</div>
            </div>
          ) : (
            <div style={{ maxHeight: 400, overflowY: "auto" }}>
              {preview.in_prism_only
                .sort((a, b) => a.title.localeCompare(b.title))
                .map((m: PrismSyncMovie) => (
                <div
                  key={m.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    padding: "6px 0",
                    borderBottom: "1px solid var(--color-border-default)",
                    fontSize: 13,
                    color: "var(--color-text-primary)",
                  }}
                >
                  <span style={{ flex: 1 }}>{m.title}</span>
                  <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{m.year}</span>
                  <span
                    style={badge(
                      m.status === "downloaded"
                        ? "rgba(34,197,94,0.1)"
                        : m.status === "wanted"
                          ? "rgba(234,179,8,0.1)"
                          : "rgba(148,163,184,0.1)",
                      m.status === "downloaded"
                        ? "var(--color-success)"
                        : m.status === "wanted"
                          ? "var(--color-warning)"
                          : "var(--color-text-muted)",
                    )}
                  >
                    {m.status}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Step 3: Results ──────────────────────────────────────────────────────────

function ResultView({
  result,
  onDone,
}: {
  result: PlexSyncImportResult;
  onDone: () => void;
}) {
  return (
    <div style={{ ...card, maxWidth: 500 }}>
      <div style={{ fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)", marginBottom: 20 }}>
        Import Complete
      </div>

      <div style={{ display: "flex", gap: 24, marginBottom: 20 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 13 }}>
          <CheckCircle size={16} color="var(--color-success)" />
          <span style={{ color: "var(--color-text-primary)" }}>{result.imported} imported</span>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 13 }}>
          <AlertCircle size={16} color="var(--color-warning)" />
          <span style={{ color: "var(--color-text-primary)" }}>{result.skipped} skipped</span>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 13 }}>
          <XCircle size={16} color="var(--color-danger)" />
          <span style={{ color: "var(--color-text-primary)" }}>{result.failed} failed</span>
        </div>
      </div>

      {result.errors.length > 0 && (
        <div
          style={{
            padding: "10px 14px",
            background: "rgba(239,68,68,0.06)",
            border: "1px solid rgba(239,68,68,0.2)",
            borderRadius: 6,
            marginBottom: 20,
            maxHeight: 200,
            overflowY: "auto",
          }}
        >
          {result.errors.map((e, i) => (
            <div key={i} style={{ fontSize: 12, color: "var(--color-danger)", padding: "2px 0" }}>
              {e}
            </div>
          ))}
        </div>
      )}

      <button style={btnSecondary} onClick={onDone}>
        Compare again
      </button>
    </div>
  );
}

// ── Main page ────────────────────────────────────────────────────────────────

type Step =
  | { kind: "select" }
  | { kind: "diff"; serverID: string; preview: PlexSyncPreviewResult }
  | { kind: "result"; result: PlexSyncImportResult };

export default function LibrarySyncPage() {
  const [step, setStep] = useState<Step>({ kind: "select" });

  return (
    <div style={{ padding: "24px 32px", maxWidth: 900 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 24 }}>
        <RefreshCw size={20} color="var(--color-text-muted)" />
        <h1 style={{ fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", margin: 0 }}>
          Library Sync
        </h1>
      </div>

      <p style={{ fontSize: 13, color: "var(--color-text-secondary)", marginBottom: 24, maxWidth: 600 }}>
        Compare your media server library with Prism to find movies that exist in one
        but not the other. Import server-only movies into Prism with one click.
      </p>

      {step.kind === "select" && (
        <SelectStep
          onPreview={(serverID, preview) => setStep({ kind: "diff", serverID, preview })}
        />
      )}

      {step.kind === "diff" && (
        <DiffView
          serverID={step.serverID}
          preview={step.preview}
          onImportDone={(result) => setStep({ kind: "result", result })}
          onBack={() => setStep({ kind: "select" })}
        />
      )}

      {step.kind === "result" && (
        <ResultView
          result={step.result}
          onDone={() => setStep({ kind: "select" })}
        />
      )}
    </div>
  );
}
