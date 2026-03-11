import type { Release } from "@/types";

export function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function formatDate(iso: string, includeYear = false): string {
  const opts: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  };
  if (includeYear) opts.year = "numeric";
  return new Date(iso).toLocaleString(undefined, opts);
}

export type ReleaseSortField = "size" | "seeds" | "age";

export const RELEASE_SORT_LABELS: Record<ReleaseSortField, string> = {
  seeds: "Seeds",
  size: "Size",
  age: "Age",
};

export function sortReleases(releases: Release[], field: ReleaseSortField, dir: "asc" | "desc"): Release[] {
  const sorted = [...releases].sort((a, b) => {
    switch (field) {
      case "size": return a.size - b.size;
      case "seeds": return (a.seeds ?? 0) - (b.seeds ?? 0);
      case "age": return (a.age_days ?? 0) - (b.age_days ?? 0);
    }
  });
  return dir === "desc" ? sorted.reverse() : sorted;
}
