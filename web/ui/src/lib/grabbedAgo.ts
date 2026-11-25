// grabbedAgo says how long ago a release was grabbed, to follow the words
// "Already grabbed": "today", "3d ago", "2mo ago", "1y ago". Under a day
// is "today", never "Today ago".
export function grabbedAgo(grabbedAt: string, now: number = Date.now()): string {
  const days = Math.floor((now - new Date(grabbedAt).getTime()) / 86_400_000);
  if (!Number.isFinite(days) || days < 1) return "today";
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}
