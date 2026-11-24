import { useState, useEffect } from "react";
import Modal from "@beacon-shared/Modal";
import TableScroll from "@beacon-shared/TableScroll";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import { useLibraries, useCreateLibrary, useUpdateLibrary, useDeleteLibrary, useScanLibrary } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import { useMediaManagement } from "@/api/media-management";
import { DirPicker } from "@/components/DirPicker";
import type { Library, LibraryRequest } from "@/types";
import { inputStyle, labelStyle, fieldStyle } from "@/lib/styles";
import DiskScanModal from "./DiskScanModal";

// Form state

interface FormState {
  name: string;
  root_path: string;
  default_quality_profile_id: string;
  min_free_space_gb: string; // string for controlled input, parsed on submit
  naming_format: string;
  folder_format: string;
}

function emptyForm(): FormState {
  return { name: "", root_path: "", default_quality_profile_id: "", min_free_space_gb: "0", naming_format: "", folder_format: "" };
}

function libraryToForm(lib: Library): FormState {
  return {
    name: lib.name,
    root_path: lib.root_path,
    default_quality_profile_id: lib.default_quality_profile_id ?? "",
    min_free_space_gb: String(lib.min_free_space_gb),
    naming_format: lib.naming_format ?? "",
    folder_format: lib.folder_format ?? "",
  };
}

function formToRequest(f: FormState): LibraryRequest {
  return {
    name: f.name.trim(),
    root_path: f.root_path.trim(),
    default_quality_profile_id: f.default_quality_profile_id || undefined,
    min_free_space_gb: parseInt(f.min_free_space_gb, 10) || 0,
    naming_format: f.naming_format.trim() || undefined,
    folder_format: f.folder_format.trim() || undefined,
  };
}

// Library edit modal

interface LibraryModalProps {
  editing: Library | null;
  onClose: () => void;
}

function LibraryModal({ editing, onClose }: LibraryModalProps) {
  const [form, setForm] = useState<FormState>(
    editing ? libraryToForm(editing) : emptyForm()
  );
  const [error, setError] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);

  const { data: profiles } = useQualityProfiles();
  const { data: mm } = useMediaManagement();
  const createLib = useCreateLibrary();
  const updateLib = useUpdateLibrary();

  const isPending = createLib.isPending || updateLib.isPending;

  // Seed the quality profile dropdown to the first profile once the list
  // loads. Without this, the state is "" while the <select> visually shows
  // the first <option>, so submitting without touching the dropdown fails
  // validation on the server.
  useEffect(() => {
    if (form.default_quality_profile_id) return;
    const first = profiles?.[0]?.id;
    if (first) {
      setForm((f) => ({ ...f, default_quality_profile_id: first }));
    }
  }, [profiles, form.default_quality_profile_id]);

  function set(field: keyof FormState, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
    setError(null);
  }

  function handleSubmit() {
    if (!form.name.trim()) { setError("Name is required."); return; }
    if (!form.root_path.trim()) { setError("Root path is required."); return; }
    if (!form.default_quality_profile_id) { setError("Quality profile is required."); return; }

    const body = formToRequest(form);

    if (editing) {
      updateLib.mutate({ id: editing.id, ...body }, { onSuccess: onClose, onError: (e) => setError(e.message) });
    } else {
      createLib.mutate(body, { onSuccess: onClose, onError: (e) => setError(e.message) });
    }
  }

  function onInputFocus(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-accent)";
  }
  function onInputBlur(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    (e.currentTarget as HTMLElement).style.borderColor = "var(--color-border-default)";
  }

  return (
    <Modal onClose={onClose} width={520} innerStyle={{ padding: 24, gap: 20 }}>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
            {editing ? "Edit Library" : "Add Library"}
          </h2>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              fontSize: 18,
              lineHeight: 1,
              padding: "4px 6px",
              borderRadius: 4,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)"; }}
          >
            ✕
          </button>
        </div>

        {/* Fields */}
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div style={fieldStyle}>
            <label style={labelStyle}>Name *</label>
            <input
              style={inputStyle}
              value={form.name}
              onChange={(e) => set("name", e.currentTarget.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder="e.g. Movies"
              autoFocus
            />
          </div>

          <div style={fieldStyle}>
            <label style={labelStyle}>Root Path *</label>
            <div style={{ display: "flex", gap: 6 }}>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12, flex: 1 }}
                value={form.root_path}
                onChange={(e) => set("root_path", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder="/movies"
              />
              <button
                type="button"
                onClick={() => setPickerOpen(true)}
                title="Browse for folder"
                style={{
                  flexShrink: 0,
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  padding: "0 12px",
                  fontSize: 13,
                  color: "var(--color-text-secondary)",
                  cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
                onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-accent)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)"; }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--color-border-default)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)"; }}
              >
                Browse…
              </button>
            </div>
            <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 4, lineHeight: 1.4 }}>
              The root folder where movies are stored. In Docker, this must match a mounted volume (e.g. <code style={{ fontSize: 11 }}>/movies</code>).
            </span>
          </div>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
            <div style={fieldStyle}>
              <label style={labelStyle}>Quality Profile</label>
              <select
                style={{ ...inputStyle, cursor: "pointer" }}
                value={form.default_quality_profile_id}
                onChange={(e) => set("default_quality_profile_id", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
              >
                {profiles?.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>Min Free Space (GB)</label>
              <input
                style={inputStyle}
                type="number"
                min="0"
                value={form.min_free_space_gb}
                onChange={(e) => set("min_free_space_gb", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
              />
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>File Naming Format Override</label>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                value={form.naming_format}
                onChange={(e) => set("naming_format", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder={mm?.standard_movie_format ?? "Leave blank to use global default"}
              />
            </div>

            <div style={fieldStyle}>
              <label style={labelStyle}>Folder Format Override</label>
              <input
                style={{ ...inputStyle, fontFamily: "var(--font-family-mono)", fontSize: 12 }}
                value={form.folder_format}
                onChange={(e) => set("folder_format", e.currentTarget.value)}
                onFocus={onInputFocus}
                onBlur={onInputBlur}
                placeholder={mm?.movie_folder_format ?? "Leave blank to use global default"}
              />
            </div>
          </div>
        </div>

        {/* Error */}
        {error && (
          <p style={{ margin: 0, fontSize: 12, color: "var(--color-danger)" }}>{error}</p>
        )}

        {/* Footer */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "none"; }}
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={isPending}
            style={{
              background: isPending ? "var(--color-bg-subtle)" : "var(--color-accent)",
              color: isPending ? "var(--color-text-muted)" : "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 20px",
              fontSize: 13,
              fontWeight: 500,
              cursor: isPending ? "not-allowed" : "pointer",
            }}
            onMouseEnter={(e) => {
              if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)";
            }}
            onMouseLeave={(e) => {
              if (!isPending) (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)";
            }}
          >
            {isPending ? "Saving…" : editing ? "Save Changes" : "Add Library"}
          </button>
        </div>

      <DirPicker
        key={pickerOpen ? "open" : "closed"}
        open={pickerOpen}
        value={form.root_path}
        onSelect={(path) => set("root_path", path)}
        onClose={() => setPickerOpen(false)}
      />
    </Modal>
  );
}

// Disk scan modal

interface RowActionsProps {
  library: Library;
  onEdit: () => void;
  onImport: () => void;
}

function RowActions({ library, onEdit, onImport }: RowActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [scanned, setScanned] = useState(false);
  const deleteLib = useDeleteLibrary();
  const scanLib = useScanLibrary();

  function handleScan() {
    scanLib.mutate(library.id, {
      onSuccess: () => {
        setScanned(true);
        setTimeout(() => setScanned(false), 2000);
      },
    });
  }

  if (confirming) {
    return (
      <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
        <span style={{ fontSize: 12, color: "var(--color-text-secondary)" }}>Delete?</span>
        <button
          onClick={() => deleteLib.mutate(library.id, { onSuccess: () => setConfirming(false) })}
          disabled={deleteLib.isPending}
          style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 15%, transparent)")}
        >
          {deleteLib.isPending ? "…" : "Yes"}
        </button>
        <button
          onClick={() => setConfirming(false)}
          style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
        >
          No
        </button>
      </div>
    );
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 6, justifyContent: "flex-end" }}>
      {scanned ? (
        <span style={{ fontSize: 12, color: "var(--color-success)" }}>Scanning ✓</span>
      ) : (
        <button
          onClick={handleScan}
          disabled={scanLib.isPending}
          style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
        >
          {scanLib.isPending ? "…" : "Scan"}
        </button>
      )}
      <button
        onClick={onImport}
        style={actionBtn("var(--color-accent)", "color-mix(in srgb, var(--color-accent) 12%, transparent)")}
      >
        Import
      </button>
      <button
        onClick={onEdit}
        style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
      >
        Edit
      </button>
      <button
        onClick={() => setConfirming(true)}
        style={actionBtn("var(--color-danger)", "color-mix(in srgb, var(--color-danger) 12%, transparent)")}
      >
        Delete
      </button>
    </div>
  );
}

function actionBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 12,
    color,
    cursor: "pointer",
  };
}

// Page

export default function LibraryList() {
  const { data, isLoading, error } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const [modal, setModal] = useState<{ open: boolean; editing: Library | null }>({
    open: false,
    editing: null,
  });
  const [importLibrary, setImportLibrary] = useState<Library | null>(null);

  const profileMap = Object.fromEntries((profiles ?? []).map((p) => [p.id, p.name]));

  function openCreate() { setModal({ open: true, editing: null }); }
  function openEdit(lib: Library) { setModal({ open: true, editing: lib }); }
  function closeModal() { setModal({ open: false, editing: null }); }

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <PageHeader
        title="Libraries"
        description="Media root folders scanned for movie files."
        docsUrl={DOCS_URLS.libraries}
        action={
          <button
            onClick={openCreate}
            style={{
              background: "var(--color-accent)",
              color: "var(--color-accent-fg)",
              border: "none",
              borderRadius: 6,
              padding: "8px 16px",
              fontSize: 13,
              fontWeight: 500,
              cursor: "pointer",
              flexShrink: 0,
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent-hover)"; }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = "var(--color-accent)"; }}
          >
            + Add Library
          </button>
        }
      />

      {/* Table card */}
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-subtle)",
          borderRadius: 8,
          boxShadow: "var(--shadow-card)",
          overflow: "hidden",
        }}
      >
        {isLoading ? (
          <div style={{ padding: 20, display: "flex", flexDirection: "column", gap: 12 }}>
            {[1, 2, 3].map((i) => (
              <div key={i} className="skeleton" style={{ height: 44, borderRadius: 4 }} />
            ))}
          </div>
        ) : error ? (
          <div style={{ padding: 24, fontSize: 13, color: "var(--color-text-muted)" }}>
            Failed to load libraries.
          </div>
        ) : !data?.length ? (
          <div style={{ padding: 48, textAlign: "center" }}>
            <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
              No libraries configured
            </p>
            <p style={{ margin: "6px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
              Add a library to start tracking movies.
            </p>
          </div>
        ) : (
          <TableScroll minWidth={800}>
            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: "1px solid var(--color-border-subtle)" }}>
                  {["Name", "Root Path", "Quality Profile", "Min Free Space", ""].map((h) => (
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
              {data.map((lib, i) => (
                <tr
                  key={lib.id}
                  style={{
                    borderBottom: i < data.length - 1 ? "1px solid var(--color-border-subtle)" : "none",
                  }}
                >
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-primary)", fontWeight: 500 }}>
                    {lib.name}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, maxWidth: 260 }}>
                    <span
                      style={{
                        display: "block",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                        fontFamily: "var(--font-family-mono)",
                        fontSize: 12,
                        color: "var(--color-text-secondary)",
                      }}
                      title={lib.root_path}
                    >
                      {lib.root_path}
                    </span>
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)" }}>
                    {lib.default_quality_profile_id
                      ? (profileMap[lib.default_quality_profile_id] ?? "—")
                      : <span style={{ color: "var(--color-text-muted)" }}>None</span>}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, color: "var(--color-text-secondary)", whiteSpace: "nowrap" }}>
                    {lib.min_free_space_gb > 0 ? `${lib.min_free_space_gb} GB` : <span style={{ color: "var(--color-text-muted)" }}>None</span>}
                  </td>
                  <td style={{ padding: "0 16px", height: 52, width: 1 }}>
                    <RowActions
                      library={lib}
                      onEdit={() => openEdit(lib)}
                      onImport={() => setImportLibrary(lib)}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
            </table>
          </TableScroll>
        )}
      </div>

      {/* Edit/create modal */}
      {modal.open && (
        <LibraryModal editing={modal.editing} onClose={closeModal} />
      )}

      {/* Disk import modal */}
      {importLibrary && (
        <DiskScanModal library={importLibrary} onClose={() => setImportLibrary(null)} />
      )}
    </div>
  );
}
