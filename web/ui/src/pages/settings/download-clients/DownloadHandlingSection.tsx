import { useState, useEffect } from "react";
import { useDownloadHandling, useUpdateDownloadHandling } from "@/api/download-handling";
import type { DownloadHandling } from "@/types";
import { inputStyle } from "@/lib/styles";

export default function DownloadHandlingSection() {
  const { data, isLoading } = useDownloadHandling();
  const update = useUpdateDownloadHandling();

  const defaultForm: DownloadHandling = {
    enable_completed: true,
    check_interval_minutes: 1,
    redownload_failed: true,
    redownload_failed_interactive: false,
  };

  const [form, setForm] = useState<DownloadHandling>(defaultForm);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (data && !initialized) {
      setForm(data);
      setInitialized(true);
    }
  }, [data, initialized]);

  function set<K extends keyof DownloadHandling>(key: K, val: DownloadHandling[K]) {
    setForm((f) => ({ ...f, [key]: val }));
  }

  const isDirty = initialized && data && (
    form.enable_completed !== data.enable_completed ||
    form.check_interval_minutes !== data.check_interval_minutes ||
    form.redownload_failed !== data.redownload_failed ||
    form.redownload_failed_interactive !== data.redownload_failed_interactive
  );

  const sectionCardStyle: React.CSSProperties = {
    background: "var(--color-bg-surface)",
    border: "1px solid var(--color-border-subtle)",
    borderRadius: 8,
    boxShadow: "var(--shadow-card)",
    overflow: "hidden",
    marginTop: 24,
  };

  const sectionHeaderStyle: React.CSSProperties = {
    padding: "14px 20px",
    borderBottom: "1px solid var(--color-border-subtle)",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
  };

  const sectionBodyStyle: React.CSSProperties = {
    padding: "16px 20px",
    display: "flex",
    flexDirection: "column",
    gap: 14,
  };

  const toggleRowStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "10px 0",
    borderBottom: "1px solid var(--color-border-subtle)",
  };

  const lastToggleRowStyle: React.CSSProperties = {
    ...toggleRowStyle,
    borderBottom: "none",
    paddingBottom: 0,
  };

  if (isLoading) {
    return (
      <div style={sectionCardStyle}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>Completed Download Handling</span>
        </div>
        <div style={{ padding: 20 }}>
          <div className="skeleton" style={{ height: 36, borderRadius: 4 }} />
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Completed Download Handling */}
      <div style={sectionCardStyle}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Completed Download Handling
          </span>
          {isDirty && (
            <button
              onClick={() => update.mutate(form)}
              disabled={update.isPending}
              style={{
                background: update.isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
                color: update.isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
                border: "none",
                borderRadius: 5,
                padding: "5px 14px",
                fontSize: 12,
                fontWeight: 500,
                cursor: update.isPending ? "not-allowed" : "pointer",
              }}
            >
              {update.isPending ? "Saving…" : "Save Changes"}
            </button>
          )}
        </div>
        <div style={sectionBodyStyle}>
          <div style={toggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>Enable</p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Automatically import completed downloads into your library
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.enable_completed}
                onChange={(e) => set("enable_completed", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
          <div style={lastToggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: form.enable_completed ? "var(--color-text-primary)" : "var(--color-text-muted)" }}>
                Check For Finished Downloads Interval
              </p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                How often to check for completed downloads (minutes)
              </p>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <input
                type="number"
                min="1"
                max="60"
                value={form.check_interval_minutes}
                onChange={(e) => set("check_interval_minutes", Math.max(1, parseInt(e.currentTarget.value, 10) || 1))}
                disabled={!form.enable_completed}
                style={{
                  ...inputStyle,
                  width: 72,
                  textAlign: "right",
                  opacity: form.enable_completed ? 1 : 0.4,
                }}
              />
              <span style={{ fontSize: 13, color: "var(--color-text-secondary)", whiteSpace: "nowrap" }}>minutes</span>
            </div>
          </div>
        </div>
      </div>

      {/* Failed Download Handling */}
      <div style={{ ...sectionCardStyle, marginTop: 16 }}>
        <div style={sectionHeaderStyle}>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Failed Download Handling
          </span>
        </div>
        <div style={sectionBodyStyle}>
          <div style={toggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>Redownload Failed</p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Automatically search for a replacement when a download fails
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.redownload_failed}
                onChange={(e) => set("redownload_failed", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
          <div style={lastToggleRowStyle}>
            <div>
              <p style={{ margin: 0, fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
                Redownload Failed from Interactive Search
              </p>
              <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
                Include failures triggered by manual interactive search
              </p>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
              <input
                type="checkbox"
                checked={form.redownload_failed_interactive}
                onChange={(e) => set("redownload_failed_interactive", e.currentTarget.checked)}
                style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
              />
            </label>
          </div>
        </div>
      </div>
    </>
  );
}
