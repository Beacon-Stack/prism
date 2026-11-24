// The sidebar's pages, shared by the sidebar (Shell.tsx) and the command
// palette (commands.ts), so the two always list the same pages in the
// same order with the same icons.
import {
  Activity,
  ArrowDownToLine,
  Ban,
  BarChart2,
  Bell,
  Bookmark,
  CalendarDays,
  Compass,
  Download,
  FileText,
  Film,
  Gauge,
  History,
  KeyRound,
  Layers,
  LayoutDashboard,
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
} from "lucide-react";
import type { NavItem } from "@beacon-shared/Shell";

// Shared with Pilot/Prism: the same items in the same order, so the two
// apps feel alike; each app's own item goes last.
export const mainNav: NavItem[] = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/discover", icon: Compass, label: "Discover" },
  { to: "/activity", icon: Activity, label: "Activity" },
  { to: "/calendar", icon: CalendarDays, label: "Calendar" },
  { to: "/wanted", icon: Bookmark, label: "Wanted" },
  { to: "/stats", icon: BarChart2, label: "Statistics" },
  { to: "/queue", icon: Download, label: "Queue" },
  { to: "/history", icon: History, label: "History" },
  { to: "/library-sync", icon: RefreshCw, label: "Library Sync" },
];

// Shared with Pilot/Prism: the same items in the same order; each app's
// own items sit right after the closest shared one.
export const settingsNav: NavItem[] = [
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
  { to: "/settings/system/logs", icon: FileText, label: "Logs" },
  { to: "/settings/app", icon: Paintbrush, label: "App Settings" },
];
