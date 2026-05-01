import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";

// New split-category vocabulary, plus legacy "grab"/"import" the
// backend still accepts for back-compat.
export type ActivityCategory =
  | "grab_succeeded"
  | "grab_failed"
  | "import_succeeded"
  | "import_failed"
  | "task"
  | "health"
  | "movie"
  | "grab"
  | "import";

export interface Activity {
  id: string;
  type: string;
  category: ActivityCategory;
  movie_id?: string;
  title: string;
  detail?: Record<string, unknown>;
  created_at: string;
}

export interface ActivityListResult {
  activities: Activity[];
  total: number;
}

export interface ActivityFilters {
  category?: string;
  since?: string;
  limit?: number;
}

export function useActivity(filters?: ActivityFilters) {
  const params = new URLSearchParams();
  if (filters?.category) params.set("category", filters.category);
  if (filters?.since) params.set("since", filters.since);
  if (filters?.limit) params.set("limit", String(filters.limit));
  const qs = params.toString();

  return useQuery({
    queryKey: ["activity", filters],
    queryFn: () =>
      apiFetch<ActivityListResult>(`/activity${qs ? `?${qs}` : ""}`),
    refetchInterval: 15_000,
  });
}

// ── Needs attention ──────────────────────────────────────────────────────────
//
// Powers the Activity-page rail of the same name. Items come from two
// sources server-side:
//   - grab_history rows where download_status ∈ {'failed','removed'}
//   - activity_log rows where category = 'import_failed'
// Both bucketed within the requested window.

export type AttentionKind = "grab_failed" | "import_failed";

export interface AttentionItem {
  kind: AttentionKind;
  grab_id?: string;
  movie_id?: string;
  release_title: string;
  detail?: string;
  created_at: string;
}

export interface AttentionResult {
  items: AttentionItem[];
  counts: {
    grab_failed: number;
    import_failed: number;
  };
}

export function useNeedsAttention(windowHours = 48) {
  return useQuery({
    queryKey: ["activity", "needs-attention", windowHours],
    queryFn: () =>
      apiFetch<AttentionResult>(`/activity/needs-attention?hours=${windowHours}`),
    refetchInterval: 30_000,
  });
}
