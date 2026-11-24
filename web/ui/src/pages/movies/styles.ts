import type { CSSProperties } from "react";

export function actionBtn(color: string, bg: string): CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "5px 12px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap",
  };
}
