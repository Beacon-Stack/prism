import { useState } from "react";
import { Trash2 } from "lucide-react";
import { useConfirm } from "@beacon-shared/ConfirmDialog";
import { toast } from "sonner";
import { useMovieFiles, useDeleteMovieFile, useRenameMovie } from "@/api/movies";
import Modal from "@beacon-shared/Modal";
import type { RenamePreviewItem, MediaInfo, Quality } from "@/types";
import { formatBytes } from "@/lib/utils";
import { useMediainfoStatus, useScanMovieFile } from "@/api/mediainfo";

function RenameModal({
  movieId,
  preview,
  onClose,
}: {
  movieId: string;
  preview: RenamePreviewItem[];
  onClose: () => void;
}) {
  const rename = useRenameMovie();

  function basename(p: string): string {
    return p.split("/").pop() ?? p;
  }

  async function handleConfirm() {
    await rename.mutateAsync({ id: movieId, dryRun: false });
    toast.success(`Renamed ${preview.length} file${preview.length !== 1 ? "s" : ""}`);
    onClose();
  }

  return (
    <Modal onClose={onClose} width={560} maxWidth="90vw" maxHeight="80vh" innerStyle={{ padding: "24px 28px", overflow: "auto" }}>
        <h2 style={{ margin: "0 0 16px", fontSize: 16, fontWeight: 700, color: "var(--color-text-primary)" }}>
          Rename Files
        </h2>

        <div style={{ display: "flex", flexDirection: "column", gap: 6, marginBottom: 20 }}>
          {preview.map((item) => (
            <div
              key={item.file_id}
              style={{
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-subtle)",
                borderRadius: 6,
                padding: "10px 12px",
                display: "flex",
                flexDirection: "column",
                gap: 4,
              }}
            >
              <div
                style={{ fontSize: 11, color: "var(--color-text-muted)", fontFamily: "var(--font-family-mono)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
                title={item.old_path}
              >
                {basename(item.old_path)}
              </div>
              <div style={{ fontSize: 11, color: "var(--color-text-muted)", lineHeight: 1 }}>↓</div>
              <div
                style={{ fontSize: 11, color: "var(--color-success)", fontFamily: "var(--font-family-mono)", fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
                title={item.new_path}
              >
                {basename(item.new_path)}
              </div>
            </div>
          ))}
        </div>

        <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
          <button
            onClick={onClose}
            disabled={rename.isPending}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 5,
              padding: "7px 16px",
              fontSize: 13,
              color: "var(--color-text-secondary)",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={rename.isPending}
            style={{
              background: "var(--color-accent)",
              border: "none",
              borderRadius: 5,
              padding: "7px 16px",
              fontSize: 13,
              fontWeight: 600,
              color: "var(--color-accent-fg)",
              cursor: rename.isPending ? "not-allowed" : "pointer",
            }}
          >
            {rename.isPending ? "Renaming…" : "Rename"}
          </button>
        </div>
    </Modal>
  );
}

// MediaInfo helpers

// Normalise ffprobe codec names to Prism's Quality.codec values.
function normCodec(ffprobeCodec: string): string {
  const map: Record<string, string> = {
    x265: "x265", hevc: "x265", h265: "x265",
    x264: "x264", h264: "x264", avc: "x264",
    av1: "AV1", "AV1": "AV1",
    vp9: "VP9",
  };
  return map[ffprobeCodec?.toLowerCase()] ?? ffprobeCodec ?? "";
}

function hasMismatch(claimed: Quality, actual: MediaInfo): boolean {
  if (!actual) return false;
  const codecMismatch = claimed.codec && actual.codec && normCodec(actual.codec) !== claimed.codec;
  const resMismatch = claimed.resolution && actual.resolution && actual.resolution !== claimed.resolution;
  const hdrMismatch = claimed.hdr && actual.hdr_format && actual.hdr_format !== claimed.hdr;
  return !!(codecMismatch || resMismatch || hdrMismatch);
}

function MediainfoRow({ info, claimed }: { info: MediaInfo; claimed: Quality }) {
  const parts: string[] = [];
  if (info.resolution) parts.push(info.resolution);
  if (info.codec) parts.push(info.codec);
  if (info.hdr_format && info.hdr_format !== "SDR") parts.push(info.hdr_format);
  if (info.bit_depth && info.bit_depth > 8) parts.push(`${info.bit_depth}-bit`);
  if (info.audio_codec) {
    const audio = info.audio_channels === 8 ? `${info.audio_codec} 7.1`
      : info.audio_channels === 6 ? `${info.audio_codec} 5.1`
      : info.audio_codec;
    parts.push(audio);
  }
  const mismatch = hasMismatch(claimed, info);

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4, flexWrap: "wrap" }}>
      <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Actual:</span>
      <span style={{ fontSize: 11, color: "var(--color-text-secondary)" }}>
        {parts.join(" · ") || "—"}
      </span>
      {mismatch && (
        <span
          title={`Filename claims: ${[claimed.resolution, claimed.codec, claimed.hdr].filter(Boolean).join(" ")}\nActual: ${parts.join(" ")}`}
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 3,
            padding: "1px 6px",
            borderRadius: 4,
            fontSize: 10,
            fontWeight: 600,
            background: "color-mix(in srgb, var(--color-warning) 15%, transparent)",
            color: "var(--color-warning)",
            cursor: "help",
          }}
        >
          ⚠ Mismatch
        </span>
      )}
    </div>
  );
}

// Files tab

export default function FilesTab({ movieId }: { movieId: string }) {
  const { data: files, isLoading, error } = useMovieFiles(movieId);
  const deleteFile = useDeleteMovieFile(movieId);
  const rename = useRenameMovie();
  const scanFile = useScanMovieFile(movieId);
  const { data: mediainfoStatus } = useMediainfoStatus();
  const confirm = useConfirm();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [scanningId, setScanningId] = useState<string | null>(null);
  const [renamePreview, setRenamePreview] = useState<RenamePreviewItem[] | null>(null);

  async function handleDelete(fileId: string, deleteFromDisk: boolean) {
    setDeletingId(fileId);
    try {
      await deleteFile.mutateAsync({ fileId, deleteFromDisk });
      toast.success(deleteFromDisk ? "File deleted from disk" : "File record removed");
    } catch {
      toast.error("Failed to delete file");
    } finally {
      setDeletingId(null);
    }
  }

  async function handleScan(fileId: string) {
    setScanningId(fileId);
    try {
      await scanFile.mutateAsync({ fileId });
      toast.success("Scan queued — results will appear shortly");
    } catch {
      toast.error("Scan failed");
    } finally {
      setScanningId(null);
    }
  }

  async function handleRenamePreview() {
    const result = await rename.mutateAsync({ id: movieId, dryRun: true });
    if (!result.renamed?.length) {
      toast.info("File is already using the standard format");
      return;
    }
    setRenamePreview(result.renamed);
  }

  if (isLoading) {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: "8px 0" }}>
        {[1, 2].map((i) => (
          <div key={i} className="skeleton" style={{ height: 56, borderRadius: 6 }} />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 13, color: "var(--color-danger)", margin: 0 }}>
        Failed to load files: {(error as Error).message}
      </p>
    );
  }

  if (!files?.length) {
    return (
      <div style={{ padding: "32px 0", textAlign: "center" }}>
        <p style={{ margin: 0, fontSize: 14, color: "var(--color-text-secondary)", fontWeight: 500 }}>
          No files
        </p>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-muted)" }}>
          No files are associated with this movie yet.
        </p>
      </div>
    );
  }

  return (
    <>
      {renamePreview && (
        <RenameModal
          movieId={movieId}
          preview={renamePreview}
          onClose={() => setRenamePreview(null)}
        />
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
        {/* Rename toolbar */}
        <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 4 }}>
          <button
            onClick={handleRenamePreview}
            disabled={rename.isPending}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 5,
              padding: "5px 12px",
              fontSize: 12,
              color: "var(--color-text-secondary)",
              cursor: rename.isPending ? "not-allowed" : "pointer",
              whiteSpace: "nowrap",
            }}
            onMouseEnter={(e) => {
              if (!rename.isPending) (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
            }}
          >
            {rename.isPending ? "Checking…" : "Rename to standard format"}
          </button>
        </div>

        {files.map((file) => (
          <div
            key={file.id}
            style={{
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-subtle)",
              borderRadius: 6,
              padding: "12px 14px",
            }}
          >
            {/* Path */}
            <div
              style={{
                fontSize: 12,
                fontFamily: "var(--font-family-mono)",
                color: "var(--color-text-secondary)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
                marginBottom: 6,
              }}
              title={file.path}
            >
              {file.path}
            </div>

            {/* Meta row */}
            <div style={{ display: "flex", alignItems: "center", gap: 16, flexWrap: "wrap" }}>
              <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
                {formatBytes(file.size_bytes)}
              </span>
              {file.quality?.resolution && (
                <span
                  style={{
                    display: "inline-block",
                    padding: "1px 6px",
                    borderRadius: 4,
                    fontSize: 10,
                    fontWeight: 600,
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                    background: "color-mix(in srgb, var(--color-accent) 12%, transparent)",
                    color: "var(--color-accent)",
                  }}
                >
                  {[file.quality.resolution, file.quality.source].filter(Boolean).join(" ")}
                </span>
              )}
              {file.edition && (
                <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{file.edition}</span>
              )}
              <span style={{ fontSize: 12, color: "var(--color-text-muted)", marginLeft: "auto" }}>
                Imported {new Date(file.imported_at).toLocaleDateString()}
              </span>

              {/* Re-scan button (only when scanner is available) */}
              {mediainfoStatus?.available && (
                <button
                  onClick={() => handleScan(file.id)}
                  disabled={scanningId === file.id}
                  title="Re-scan with ffprobe"
                  style={{
                    background: "none",
                    border: "none",
                    cursor: scanningId === file.id ? "not-allowed" : "pointer",
                    color: "var(--color-text-muted)",
                    fontSize: 11,
                    padding: "2px 6px",
                    borderRadius: 4,
                    flexShrink: 0,
                  }}
                  onMouseEnter={(e) => {
                    if (scanningId !== file.id) (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-primary)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
                  }}
                >
                  {scanningId === file.id ? "…" : "↻ Scan"}
                </button>
              )}

              {/* Delete */}
              <button
                onClick={async () => {
                  const fromDisk = await confirm({
                    title: "Delete File",
                    message: "Also delete the file from disk? Choosing 'Delete' removes from disk. Cancel to remove the record only.",
                    confirmLabel: "Delete from Disk",
                  });
                  handleDelete(file.id, fromDisk);
                }}
                disabled={deletingId === file.id}
                title="Delete file"
                style={{
                  background: "none",
                  border: "none",
                  cursor: "pointer",
                  color: "var(--color-text-muted)",
                  display: "flex",
                  alignItems: "center",
                  padding: 4,
                  borderRadius: 4,
                  transition: "color 150ms ease",
                  flexShrink: 0,
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLButtonElement).style.color = "var(--color-danger)";
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
                }}
              >
                <Trash2 size={14} />
              </button>
            </div>

            {/* MediaInfo row (shown once ffprobe data is available) */}
            {file.mediainfo && (
              <MediainfoRow info={file.mediainfo} claimed={file.quality} />
            )}
          </div>
        ))}
      </div>
    </>
  );
}
