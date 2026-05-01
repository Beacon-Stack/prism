import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";

export interface HaulRecord {
  info_hash: string;
  name: string;
  save_path: string;
  category: string;
  added_at: string;
  completed_at: string; // empty string when not completed
  removed_at: string;   // empty string when still present
  requester: string;
  series_id: string;
  episode_id: string;
  tmdb_id: number;
  season: number;
  episode: number;
}

export function useMovieHaulHistory(movieId: string) {
  return useQuery({
    queryKey: ["movies", movieId, "haul-history"],
    queryFn: () =>
      apiFetch<{ records: HaulRecord[] }>(`/movies/${movieId}/haul-history`)
        .then((r) => r.records),
    enabled: !!movieId,
    // Silent failure: if Haul isn't configured the endpoint returns []
    throwOnError: false,
  });
}

export function useReimportFromHaul(movieId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (infoHash: string) =>
      apiFetch<{ status: string }>("/import/from-haul", {
        method: "POST",
        body: JSON.stringify({ info_hash: infoHash }),
      }),
    onSuccess: () => {
      toast.success("Re-import queued");
      void qc.invalidateQueries({ queryKey: ["movies", movieId] });
      void qc.invalidateQueries({ queryKey: ["movies", movieId, "haul-history"] });
    },
    onError: (err) => toast.error((err as Error).message),
  });
}
