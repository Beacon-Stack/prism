import { useEffect } from "react";
import {
  Activity,
  BookOpen,
  Film,
} from "lucide-react";
import Shell from "@beacon-shared/Shell";
import { mainNav, settingsNav } from "./nav";
import UpdatedNotice from "@beacon-shared/UpdatedNotice";
import { useSystemHealth } from "@/api/system";
import { useWebSocket } from "@/api/websocket";
import { applyTheme } from "@/theme";
import { useCommandPalette } from "@/components/command-palette/useCommandPalette";
import { CommandPalette } from "@/components/command-palette/CommandPalette";

// HealthDot reads /api/system/health and renders a colored dot + label
// in the sidebar footer. Lives here (not web-shared) because it's bound
// to Prism's health API contract.
function HealthDot() {
  const { data: health } = useSystemHealth();
  const allOk = !health || health.status === "healthy";
  const color = allOk ? "var(--color-success)" : "var(--color-danger)";
  const label = allOk ? "All systems healthy" : "Health issues detected";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: "8px",
        padding: "0 12px",
        height: "36px",
        color: "var(--color-text-muted)",
        fontSize: "12px",
      }}
      title={label}
    >
      <Activity size={16} strokeWidth={1.5} style={{ color, flexShrink: 0 }} />
      <span style={{ color }}>{label}</span>
    </div>
  );
}

// AppIcon wraps the Film glyph in Prism's accent-color tile.
function AppIcon() {
  return (
    <div
      style={{
        width: 32,
        height: 32,
        borderRadius: "8px",
        background: "var(--color-accent)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        flexShrink: 0,
      }}
    >
      <Film size={18} color="white" strokeWidth={2} />
    </div>
  );
}

// Custom docs link rendered as a sidebar footer extra rather than
// going through Shell's docsUrl prop, since Prism wants a specific
// project-domain link. The shared Shell's generic docs link works
// for the case where the URL is the only thing that varies.
function DocsLink() {
  return (
    <a
      href="https://prism.video/docs"
      target="_blank"
      rel="noopener noreferrer"
      style={{
        display: "flex",
        alignItems: "center",
        gap: "8px",
        padding: "0 12px",
        height: "36px",
        color: "var(--color-text-muted)",
        fontSize: "12px",
        textDecoration: "none",
        borderRadius: "6px",
      }}
    >
      <BookOpen size={16} strokeWidth={1.5} style={{ flexShrink: 0 }} />
      <span>Docs</span>
    </a>
  );
}

export default function PrismShell() {
  useWebSocket();
  useEffect(() => {
    applyTheme();
  }, []);

  const commandPalette = useCommandPalette();

  return (
    <Shell
      appName="Prism"
      appIcon={<AppIcon />}
      mainNav={mainNav}
      settingsNav={settingsNav}
      collapsedStorageKey="sidebar-collapsed"
      sidebarFooterExtras={
        <>
          <HealthDot />
          <DocsLink />
        </>
      }
      overlay={
        <>
          {commandPalette.isOpen && <CommandPalette onClose={commandPalette.close} />}
          <UpdatedNotice app="Prism" />
        </>
      }
    />
  );
}
