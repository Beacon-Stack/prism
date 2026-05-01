import { useEffect } from "react";
import {
  Activity,
  ArrowDownToLine,
  Ban,
  BarChart2,
  Bell,
  BookOpen,
  Bookmark,
  CalendarDays,
  Compass,
  Download,
  Film,
  Gauge,
  History,
  KeyRound,
  Layers,
  Library,
  ListPlus,
  MonitorPlay,
  Paintbrush,
  RefreshCw,
  ScanLine,
  Search,
  Server,
  Settings2,
  ShieldOff,
  SlidersHorizontal,
  LayoutDashboard,
} from "lucide-react";
import Shell, { type NavItem } from "@beacon-shared/Shell";
import { useSystemHealth } from "@/api/system";
import { useWebSocket } from "@/api/websocket";
import { applyTheme } from "@/theme";
import { useCommandPalette } from "@/components/command-palette/useCommandPalette";
import { CommandPalette } from "@/components/command-palette/CommandPalette";

const mainNav: NavItem[] = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/activity", icon: Activity, label: "Activity" },
  { to: "/discover", icon: Compass, label: "Discover" },
  { to: "/calendar", icon: CalendarDays, label: "Calendar" },
  { to: "/wanted", icon: Bookmark, label: "Wanted" },
  { to: "/library-sync", icon: RefreshCw, label: "Library Sync" },
  { to: "/stats", icon: BarChart2, label: "Statistics" },
  { to: "/queue", icon: Download, label: "Queue" },
  { to: "/history", icon: History, label: "History" },
];

const settingsNav: NavItem[] = [
  { to: "/settings/libraries", icon: Library, label: "Libraries" },
  { to: "/settings/media-management", icon: Film, label: "Media Management" },
  { to: "/settings/media-scanning", icon: ScanLine, label: "Media Scanning" },
  { to: "/settings/quality-profiles", icon: SlidersHorizontal, label: "Quality Profiles" },
  { to: "/settings/quality-definitions", icon: Gauge, label: "Quality Definitions" },
  { to: "/settings/custom-formats", icon: Layers, label: "Custom Formats" },
  { to: "/settings/indexers", icon: Search, label: "Indexers" },
  { to: "/settings/download-clients", icon: Settings2, label: "Download Clients" },
  { to: "/settings/notifications", icon: Bell, label: "Notifications" },
  { to: "/settings/media-servers", icon: MonitorPlay, label: "Media Servers" },
  { to: "/settings/import-lists", icon: ListPlus, label: "Import Lists" },
  { to: "/settings/import-exclusions", icon: ShieldOff, label: "Import Exclusions" },
  { to: "/settings/blocklist", icon: Ban, label: "Blocklist" },
  { to: "/settings/import", icon: ArrowDownToLine, label: "Import" },
  { to: "/settings/providers", icon: KeyRound, label: "Providers" },
  { to: "/settings/system", icon: Server, label: "System" },
  { to: "/settings/app", icon: Paintbrush, label: "App Settings" },
];

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
        commandPalette.isOpen && (
          <CommandPalette onClose={commandPalette.close} />
        )
      }
    />
  );
}
