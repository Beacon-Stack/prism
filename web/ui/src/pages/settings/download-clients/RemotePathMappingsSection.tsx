import { useState } from "react";
import TableScroll from "@beacon-shared/TableScroll";
import { useRemotePathMappings, useCreateRemotePathMapping, useDeleteRemotePathMapping } from "@/api/download-handling";
import type { RemotePathMapping } from "@/types";
import { inputStyle, labelStyle, fieldStyle, actionBtn } from "@/lib/styles";

interface AddMappingForm {
  host: string;
  remote_path: string;
  local_path: string;
}

export default function RemotePathMappingsSection() {
  const { data: mappings, isLoading } = useRemotePathMappings();
  const create = useCreateRemotePathMapping();
  const del = useDeleteRemotePathMapping();

  const [adding, setAdding] = useState(false);
  const [addForm, setAddForm] = useState<AddMappingForm>({ host: "", remote_path: "", local_path: "" });
  const [addError, setAddError] = useState<string | null>(null);

  function setField(key: keyof AddMappingForm, val: string) {
    setAddForm((f) => ({ ...f, [key]: val }));
    setAddError(null);
  }

  function handleAdd() {
    if (!addForm.host.trim()) { setAddError("Host is required."); return; }
    if (!addForm.remote_path.trim()) { setAddError("Remote path is required."); return; }
    if (!addForm.local_path.trim()) { setAddError("Local path is required."); return; }
    create.mutate(
      { host: addForm.host.trim(), remote_path: addForm.remote_path.trim(), local_path: addForm.local_path.trim() },
      {
        onSuccess: () => {
          setAdding(false);
          setAddForm({ host: "", remote_path: "", local_path: "" });
        },
        onError: (e) => setAddError(e.message),
      }
    );
  }

  function focusBorder(e: React.FocusEvent<HTMLInputElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function blurBorder(e: React.FocusEvent<HTMLInputElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
  }

  const monoInput: React.CSSProperties = {
    ...inputStyle,
    fontFamily: "var(--font-family-mono)",
    fontSize: 12,
  };

  return (
    <div
      style={{
        background: "var(--color-bg-surface)",
        border: "1px solid var(--color-border-subtle)",
        borderRadius: 8,
        boxShadow: "var(--shadow-card)",
        overflow: "hidden",
        marginTop: 16,
      }}
    >
      <div
        style={{
          padding: "14px 20px",
          borderBottom: "1px solid var(--color-border-subtle)",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <div>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text-primary)" }}>
            Remote Path Mappings
          </span>
          <p style={{ margin: "2px 0 0", fontSize: 12, color: "var(--color-text-secondary)" }}>
            Translate download client paths to local filesystem paths
          </p>
        </div>
        {!adding && (
          <button
            onClick={() => setAdding(true)}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 5,
              padding: "5px 14px",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
              flexShrink: 0,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            + Add
          </button>
        )}
      </div>

      {isLoading ? (
        <div style={{ padding: 20 }}>
          <div className="skeleton" style={{ height: 36, borderRadius: 4 }} />
        </div>
      ) : (
        <>
          {/* Existing mappings */}
          {mappings && mappings.length > 0 ? (
            <TableScroll minWidth={600}>
              <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
                <thead>
                  <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                    {["Host", "Remote Path", "Local Path", ""].map((h) => (
                    <th
                      key={h}
                      style={{
                        textAlign: "left",
                        padding: "10px 16px",
                        fontSize: 11,
                        fontWeight: 600,
                        letterSpacing: "0.08em",
                        textTransform: "uppercase",
                        color: "var(--color-text-muted)",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {mappings.map((m: RemotePathMapping, i: number) => (
                  <tr
                    key={m.id}
                    style={{ borderBottom: i < mappings.length - 1 ? "1px solid var(--color-border-subtle)" : "none" }}
                  >
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-primary)", fontWeight: 500 }}>
                      {m.host}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-secondary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
                      {m.remote_path}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, color: "var(--color-text-secondary)", fontFamily: "var(--font-family-mono)", fontSize: 12 }}>
                      {m.local_path}
                    </td>
                    <td style={{ padding: "0 16px", height: 48, width: 1 }}>
                      <button
                        onClick={() => del.mutate(m.id)}
                        disabled={del.isPending}
                        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 12%, transparent)")}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
              </table>
            </TableScroll>
          ) : !adding ? (
            <div style={{ padding: 32, textAlign: "center" }}>
              <p style={{ margin: 0, fontSize: 13, color: "var(--color-text-secondary)" }}>No remote path mappings configured</p>
              <p style={{ margin: "4px 0 0", fontSize: 12, color: "var(--color-text-muted)" }}>
                Add a mapping if your download client uses different paths than this host.
              </p>
            </div>
          ) : null}

          {/* Add form */}
          {adding && (
            <div
              style={{
                padding: 16,
                borderTop: mappings && mappings.length > 0 ? "1px solid var(--color-border-subtle)" : undefined,
                display: "flex",
                flexDirection: "column",
                gap: 12,
              }}
            >
              <p style={{ margin: 0, fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase", color: "var(--color-text-muted)" }}>
                New Mapping
              </p>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 12 }}>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Host *</label>
                  <input
                    style={inputStyle}
                    value={addForm.host}
                    onChange={(e) => setField("host", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="192.168.1.100"
                    autoFocus
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Remote Path *</label>
                  <input
                    style={monoInput}
                    value={addForm.remote_path}
                    onChange={(e) => setField("remote_path", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="/downloads"
                  />
                </div>
                <div style={fieldStyle}>
                  <label style={labelStyle}>Local Path *</label>
                  <input
                    style={monoInput}
                    value={addForm.local_path}
                    onChange={(e) => setField("local_path", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="/mnt/downloads"
                  />
                </div>
              </div>
              {addError && (
                <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>{addError}</p>
              )}
              <div style={{ display: "flex", gap: 8 }}>
                <button
                  onClick={handleAdd}
                  disabled={create.isPending}
                  style={{
                    background: create.isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
                    color: create.isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
                    border: "none",
                    borderRadius: 5,
                    padding: "6px 14px",
                    fontSize: 12,
                    fontWeight: 500,
                    cursor: create.isPending ? "not-allowed" : "pointer",
                  }}
                >
                  {create.isPending ? "Adding…" : "Add Mapping"}
                </button>
                <button
                  onClick={() => { setAdding(false); setAddForm({ host: "", remote_path: "", local_path: "" }); setAddError(null); }}
                  style={{
                    background: "none",
                    border: "1px solid var(--color-border-default)",
                    borderRadius: 5,
                    padding: "6px 14px",
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    cursor: "pointer",
                  }}
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
