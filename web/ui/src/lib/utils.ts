// Common formatters now live in web-shared/utils.ts; release-sort helpers
// stay here because they depend on @/types.
import type { Release } from "@/types";

export { formatBytes, formatDate } from "@beacon-shared/utils";

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
