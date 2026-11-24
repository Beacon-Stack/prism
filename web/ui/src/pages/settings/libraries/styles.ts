import type { CSSProperties } from "react";

export function smallBtn(color: string, bg: string): CSSProperties {
  return {
    background: bg,
    border: "none",
    borderRadius: 4,
    padding: "3px 8px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap" as const,
  };
}
