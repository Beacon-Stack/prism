import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type {
  ImportListConfig,
  ImportListRequest,
  ImportExclusion,
  ImportExclusionRequest,
  ImportListSyncResult,
} from "@/types";

// ── Import Lists ─────────────────────────────────────────────────────────────

export function useImportLists() {
  return useQuery({
    queryKey: ["importlists"],
    queryFn: () => apiFetch<ImportListConfig[]>("/importlists"),
  });
}

export function useCreateImportList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ImportListRequest) =>
      apiFetch<ImportListConfig>("/importlists", { method: "POST", body: JSON.stringify(body) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["importlists"] });
      toast.success("Import list saved");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useUpdateImportList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: ImportListRequest & { id: string }) =>
      apiFetch<ImportListConfig>(`/importlists/${id}`, { method: "PUT", body: JSON.stringify(body) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["importlists"] });
      toast.success("Import list saved");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useDeleteImportList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/importlists/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["importlists"] });
      toast.success("Import list deleted");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useTestImportList() {
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/importlists/${id}/test`, { method: "POST" }),
    onSuccess: () => toast.success("Connection test passed"),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useSyncAllImportLists() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiFetch<ImportListSyncResult>("/importlists/sync", { method: "POST" }),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: ["importlists"] });
      toast.success(`Sync complete: ${result.movies_added} added, ${result.movies_skipped} skipped`);
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useSyncImportList() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiFetch<ImportListSyncResult>(`/importlists/${id}/sync`, { method: "POST" }),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: ["importlists"] });
      toast.success(`Sync complete: ${result.movies_added} added, ${result.movies_skipped} skipped`);
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

// ── Preview ──────────────────────────────────────────────────────────────────

export interface ImportListPreviewItem {
  tmdb_id: number;
  title: string;
  year: number;
  poster_path?: string;
}

export function useImportListPreview() {
  return useMutation({
    mutationFn: (body: { kind: string; settings?: Record<string, unknown> }) =>
      apiFetch<ImportListPreviewItem[]>("/importlists/preview", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    onError: (err) => toast.error((err as Error).message),
  });
}

// ── Import Exclusions ────────────────────────────────────────────────────────

export function useImportExclusions() {
  return useQuery({
    queryKey: ["importlists", "exclusions"],
    queryFn: () => apiFetch<ImportExclusion[]>("/importlists/exclusions"),
  });
}

export function useCreateImportExclusion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ImportExclusionRequest) =>
      apiFetch<ImportExclusion>("/importlists/exclusions", { method: "POST", body: JSON.stringify(body) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["importlists", "exclusions"] });
      toast.success("Exclusion added");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useDeleteImportExclusion() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/importlists/exclusions/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["importlists", "exclusions"] });
      toast.success("Exclusion removed");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

// ── Plex Auth ──────────────────────────────────────────────────────────────

interface PlexPin {
  id: number;
  code: string;
  auth_url: string;
}

interface PlexPinStatus {
  claimed: boolean;
  token?: string;
}

export function useCreatePlexPin() {
  return useMutation({
    mutationFn: () => apiFetch<PlexPin>("/plex/pin", { method: "POST" }),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function checkPlexPin(id: number): Promise<PlexPinStatus> {
  return apiFetch<PlexPinStatus>(`/plex/pin/${id}`);
}
