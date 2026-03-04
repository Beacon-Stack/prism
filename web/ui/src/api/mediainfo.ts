import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface MediainfoStatus {
  available: boolean;
  ffprobe_path?: string;
}

export function useMediainfoStatus() {
  return useQuery({
    queryKey: ["mediainfo", "status"],
    queryFn: () => apiFetch<MediainfoStatus>("/mediainfo/status"),
    staleTime: 60_000,
  });
}

export function useScanMovieFile(movieId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ fileId }: { fileId: string }) =>
      apiFetch<void>(`/movies/${movieId}/files/${fileId}/scan`, { method: "POST" }),
    onSuccess: () => {
      // Refetch files after a short delay to pick up the background scan result.
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["movies", movieId, "files"] });
      }, 3000);
    },
  });
}

export function useScanAll() {
  return useMutation({
    mutationFn: () => apiFetch<void>("/mediainfo/scan-all", { method: "POST" }),
  });
}
