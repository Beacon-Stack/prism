import { useState, useEffect, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useLookupMovies } from "@/api/movies";
import { useRunTask } from "@/api/system";
import {
  NAV_COMMANDS,
  ACTION_COMMANDS,
  filterCommands,
  type Command,
  type ActionCommand,
} from "./commands";
import type { Movie, MovieListResponse, TMDBResult } from "@/types";
import { Film } from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

interface PaletteItem {
  id: string;
  category: "navigation" | "movie" | "action";
  label: string;
  subtitle?: string;
  icon?: React.ElementType;
  posterUrl?: string;
  inLibrary?: boolean;
  onSelect: () => void;
}

interface CommandPaletteProps {
  onClose: () => void;
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function getCachedMovies(qc: ReturnType<typeof useQueryClient>): Map<number, Movie> {
  const map = new Map<number, Movie>();
  // The movies query uses ["movies", filters] as key. Try to find any cached result.
  const cache = qc.getQueriesData<MovieListResponse>({ queryKey: ["movies"] });
  for (const [, data] of cache) {
    if (data?.movies) {
      for (const m of data.movies) {
        map.set(m.tmdb_id, m);
      }
    }
  }
  return map;
}

// ── Component ────────────────────────────────────────────────────────────────

export function CommandPalette({ onClose }: CommandPaletteProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const lookup = useLookupMovies();
  const runTask = useRunTask();

  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const [debouncedQuery, setDebouncedQuery] = useState("");

  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<Element | null>(null);

  // Capture previous focus and lock body scroll
  useEffect(() => {
    previousFocus.current = document.activeElement;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "";
      if (previousFocus.current instanceof HTMLElement) {
        previousFocus.current.focus();
      }
    };
  }, []);

  // Debounce query for movie search
  useEffect(() => {
    if (query.length < 2) {
      setDebouncedQuery("");
      return;
    }
    const timer = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(timer);
  }, [query]);

  // Fire movie lookup when debounced query changes
  useEffect(() => {
    if (debouncedQuery) {
      lookup.mutate({ query: debouncedQuery });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedQuery]);

  // Reset active index when query changes
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  // ── Build flat item list ───────────────────────────────────────────────────

  const handleAction = useCallback(
    (cmd: ActionCommand) => {
      runTask.mutate(cmd.taskName, {
        onSuccess: () => toast.success(`${cmd.label} triggered`),
      });
      onClose();
    },
    [runTask, onClose],
  );

  const handleNav = useCallback(
    (cmd: Command) => {
      cmd.onSelect(navigate);
      onClose();
    },
    [navigate, onClose],
  );

  const handleMovie = useCallback(
    (_movie: TMDBResult, libraryMovie: Movie | undefined) => {
      if (libraryMovie) {
        navigate(`/movies/${libraryMovie.id}`);
      } else {
        // Navigate to dashboard — user can add from there
        navigate("/");
      }
      onClose();
    },
    [navigate, onClose],
  );

  // Build items
  const filteredNav = filterCommands(NAV_COMMANDS, query);
  const filteredActions = filterCommands(ACTION_COMMANDS, query);
  const cachedMovies = getCachedMovies(queryClient);
  const movieResults: TMDBResult[] =
    query.length >= 2 && lookup.data ? lookup.data : [];

  const items: PaletteItem[] = [];

  // Navigation
  for (const cmd of filteredNav) {
    items.push({
      id: cmd.id,
      category: "navigation",
      label: cmd.label,
      icon: cmd.icon,
      onSelect: () => handleNav(cmd),
    });
  }

  // Movies
  for (const movie of movieResults) {
    const libraryMovie = cachedMovies.get(movie.tmdb_id);
    items.push({
      id: `movie:${movie.tmdb_id}`,
      category: "movie",
      label: movie.title,
      subtitle: movie.year ? String(movie.year) : undefined,
      posterUrl: movie.poster_path
        ? `https://image.tmdb.org/t/p/w92${movie.poster_path}`
        : undefined,
      inLibrary: !!libraryMovie,
      onSelect: () => handleMovie(movie, libraryMovie),
    });
  }

  // Actions
  for (const cmd of filteredActions) {
    items.push({
      id: cmd.id,
      category: "action",
      label: cmd.label,
      icon: cmd.icon,
      onSelect: () => handleAction(cmd),
    });
  }

  // ── Keyboard handling ──────────────────────────────────────────────────────

  function onKeyDown(e: React.KeyboardEvent) {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, items.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
        break;
      case "Enter":
        e.preventDefault();
        if (items[activeIndex]) {
          items[activeIndex].onSelect();
        }
        break;
      case "Escape":
        e.preventDefault();
        onClose();
        break;
    }
  }

  // Scroll active item into view
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const active = list.querySelector(`[data-index="${activeIndex}"]`);
    if (active) {
      active.scrollIntoView({ block: "nearest" });
    }
  }, [activeIndex]);

  // ── Grouped rendering ─────────────────────────────────────────────────────

  const navItems = items.filter((i) => i.category === "navigation");
  const movieItems = items.filter((i) => i.category === "movie");
  const actionItems = items.filter((i) => i.category === "action");

  // Track global index for each item
  let globalIndex = 0;
  function nextIndex() {
    return globalIndex++;
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.5)",
        backdropFilter: "blur(2px)",
        zIndex: 300,
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "center",
        paddingTop: "20vh",
      }}
      onClick={onClose}
      data-testid="command-palette-backdrop"
    >
      <div
        style={{
          background: "var(--color-bg-surface)",
          border: "1px solid var(--color-border-default)",
          borderRadius: 12,
          width: 560,
          maxWidth: "calc(100vw - 32px)",
          maxHeight: "min(480px, 60vh)",
          boxShadow: "var(--shadow-modal)",
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
      >
        {/* Input */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            borderBottom: "1px solid var(--color-border-subtle)",
          }}
        >
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Type a command or search..."
            autoFocus
            style={{
              flex: 1,
              padding: "14px 16px",
              background: "transparent",
              border: "none",
              outline: "none",
              fontSize: 15,
              color: "var(--color-text-primary)",
            }}
            data-testid="command-palette-input"
          />
          <kbd
            style={{
              marginRight: 12,
              fontSize: 10,
              padding: "2px 6px",
              borderRadius: 4,
              background: "var(--color-bg-elevated)",
              color: "var(--color-text-muted)",
              border: "1px solid var(--color-border-subtle)",
              flexShrink: 0,
            }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div
          ref={listRef}
          style={{
            flex: 1,
            overflowY: "auto",
            padding: "8px 0",
          }}
          data-testid="command-palette-list"
        >
          {items.length === 0 && (
            <div
              style={{
                textAlign: "center",
                padding: "32px 16px",
                color: "var(--color-text-muted)",
                fontSize: 13,
              }}
            >
              {query.length >= 2 && lookup.isPending
                ? "Searching..."
                : "No results"}
            </div>
          )}

          {navItems.length > 0 && (
            <PaletteGroup label="Pages">
              {navItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {movieItems.length > 0 && (
            <PaletteGroup label="Movies">
              {movieItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}

          {query.length >= 2 && lookup.isPending && movieItems.length === 0 && (
            <PaletteGroup label="Movies">
              {[1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="skeleton"
                  style={{ height: 36, margin: "0 8px 4px", borderRadius: 6 }}
                />
              ))}
            </PaletteGroup>
          )}

          {actionItems.length > 0 && (
            <PaletteGroup label="Actions">
              {actionItems.map((item) => {
                const idx = nextIndex();
                return (
                  <PaletteRow
                    key={item.id}
                    item={item}
                    index={idx}
                    isActive={idx === activeIndex}
                    onHover={setActiveIndex}
                  />
                );
              })}
            </PaletteGroup>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Sub-components ───────────────────────────────────────────────────────────

function PaletteGroup({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div
        style={{
          padding: "8px 16px 4px",
          fontSize: 11,
          fontWeight: 600,
          letterSpacing: "0.08em",
          textTransform: "uppercase",
          color: "var(--color-text-muted)",
        }}
      >
        {label}
      </div>
      {children}
    </div>
  );
}

function PaletteRow({
  item,
  index,
  isActive,
  onHover,
}: {
  item: PaletteItem;
  index: number;
  isActive: boolean;
  onHover: (index: number) => void;
}) {
  const Icon = item.icon ?? Film;

  return (
    <button
      data-index={index}
      data-testid={`palette-item-${item.id}`}
      aria-selected={isActive}
      onClick={item.onSelect}
      onMouseEnter={() => onHover(index)}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 16px",
        cursor: "pointer",
        background: isActive ? "var(--color-bg-elevated)" : "transparent",
        border: "none",
        width: "100%",
        textAlign: "left",
        fontSize: 13,
        color: isActive
          ? "var(--color-text-primary)"
          : "var(--color-text-secondary)",
        borderRadius: 0,
      }}
    >
      {item.posterUrl ? (
        <img
          src={item.posterUrl}
          alt=""
          style={{
            width: 24,
            height: 36,
            borderRadius: 3,
            objectFit: "cover",
            flexShrink: 0,
          }}
        />
      ) : (
        <Icon size={16} strokeWidth={1.5} style={{ flexShrink: 0 }} />
      )}
      <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {item.label}
      </span>
      {item.subtitle && (
        <span style={{ fontSize: 11, color: "var(--color-text-muted)", flexShrink: 0 }}>
          {item.subtitle}
        </span>
      )}
      {item.inLibrary && (
        <span
          style={{
            fontSize: 10,
            padding: "1px 6px",
            borderRadius: 4,
            background: "color-mix(in srgb, var(--color-success) 15%, transparent)",
            color: "var(--color-success)",
            fontWeight: 600,
            flexShrink: 0,
          }}
        >
          In Library
        </span>
      )}
    </button>
  );
}
