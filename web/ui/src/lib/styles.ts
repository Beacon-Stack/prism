import type { CSSProperties } from "react";

// Style objects and small helpers shared by forms and lists across pages.

export const inputStyle: CSSProperties = {
  width: "100%",
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  padding: "8px 12px",
  fontSize: 13,
  color: "var(--color-text-primary)",
  outline: "none",
  boxSizing: "border-box",
};

export const labelStyle: CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
};

export const fieldStyle: CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 0,
};

export function actionBtn(color: string, bg: string): CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap",
  };
}

export function strSetting(settings: Record<string, unknown>, key: string): string {
  const v = settings[key];
  return typeof v === "string" ? v : "";
}

// Shim — canonical implementation lives in web-shared/styles.ts
export { card, sectionHeader } from "@beacon-shared/styles";
