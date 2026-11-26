import { useRef } from "react";
import { Link } from "react-router-dom";
import { ChevronLeft, ChevronRight, Shuffle } from "lucide-react";
import type { DiscoverResult, DiscoverShelf } from "@/api/discover";
import { DiscoverEmpty } from "@beacon-shared/Discover";
import DiscoverCard from "./DiscoverCard";

// Discover's home view: rows of movies the library doesn't have, most of
// them picked from what it does have.

const CARD_WIDTH = 150;

const arrow: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  width: 26,
  height: 26,
  borderRadius: 6,
  border: "1px solid var(--color-border-default)",
  background: "transparent",
  color: "var(--color-text-secondary)",
  cursor: "pointer",
};

function Shelf({ shelf, onAdd }: { shelf: DiscoverShelf; onAdd: (m: DiscoverResult) => void }) {
  const row = useRef<HTMLDivElement>(null);

  function scroll(direction: 1 | -1) {
    const el = row.current;
    el?.scrollBy?.({ left: direction * el.clientWidth * 0.9, behavior: "smooth" });
  }

  return (
    <section aria-label={shelf.title} data-testid={`shelf-${shelf.id}`} style={{ marginBottom: 28 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 10 }}>
        <h2 style={{ margin: 0, fontSize: 15, fontWeight: 600, color: "var(--color-text-primary)", minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {shelf.title}
        </h2>
        {shelf.see_all && (
          <Link to={shelf.see_all} style={{ fontSize: 12, color: "var(--color-accent)", textDecoration: "none", flexShrink: 0 }}>
            See all
          </Link>
        )}
        <div style={{ marginLeft: "auto", display: "flex", gap: 6, flexShrink: 0 }}>
          <button type="button" aria-label={`Scroll ${shelf.title} back`} onClick={() => scroll(-1)} style={arrow}>
            <ChevronLeft size={15} />
          </button>
          <button type="button" aria-label={`Scroll ${shelf.title} forward`} onClick={() => scroll(1)} style={arrow}>
            <ChevronRight size={15} />
          </button>
        </div>
      </div>
      <div ref={row} style={{ display: "flex", gap: 14, overflowX: "auto", scrollSnapType: "x proximity", paddingBottom: 8 }}>
        {shelf.results.map((movie) => (
          <div key={movie.tmdb_id} style={{ flex: `0 0 ${CARD_WIDTH}px`, minWidth: 0, scrollSnapAlign: "start" }}>
            <DiscoverCard movie={movie} onAdd={onAdd} />
          </div>
        ))}
      </div>
    </section>
  );
}

function ShelvesLoading() {
  return (
    <div data-testid="shelves-loading">
      {Array.from({ length: 3 }).map((_, row) => (
        <div key={row} style={{ marginBottom: 28 }}>
          <div className="skeleton" style={{ height: 18, width: 240, borderRadius: 4, marginBottom: 12 }} />
          <div style={{ display: "flex", gap: 14, overflow: "hidden" }}>
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="skeleton" style={{ flex: `0 0 ${CARD_WIDTH}px`, height: CARD_WIDTH * 1.5, borderRadius: 8 }} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

interface DiscoverShelvesProps {
  shelves?: DiscoverShelf[];
  loading: boolean;
  failed: boolean;
  /** New rows are on their way; the old ones are still showing. */
  shuffling: boolean;
  onShuffle: () => void;
  onAdd: (m: DiscoverResult) => void;
  /** Leave the home view for a plain list. */
  onBrowse: () => void;
}

export default function DiscoverShelves({ shelves, loading, failed, shuffling, onShuffle, onAdd, onBrowse }: DiscoverShelvesProps) {
  const browse = (
    <button type="button" onClick={onBrowse} style={{ background: "none", border: "none", color: "var(--color-accent)", fontSize: 13, cursor: "pointer" }}>
      Browse Trending
    </button>
  );

  if (loading) return <ShelvesLoading />;
  if (failed) return <DiscoverEmpty title="Couldn't load your picks" hint="TMDB didn't answer. Try again in a moment." action={browse} />;
  if (!shelves || shelves.length === 0) return <DiscoverEmpty title="Nothing to suggest right now" action={browse} />;

  const personal = shelves.some((s) => s.kind !== "list");

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 18 }}>
        <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
          {personal ? "Picked from what's in your library. Nothing here is one you already have." : "Add a few movies and this page starts picking from them."}
        </span>
        {personal && (
          <button
            type="button"
            onClick={onShuffle}
            disabled={shuffling}
            style={{
              marginLeft: "auto",
              display: "inline-flex",
              alignItems: "center",
              gap: 6,
              padding: "5px 12px",
              borderRadius: 6,
              border: "1px solid var(--color-border-default)",
              background: "transparent",
              color: "var(--color-text-secondary)",
              fontSize: 12,
              fontWeight: 500,
              cursor: shuffling ? "wait" : "pointer",
              flexShrink: 0,
            }}
          >
            <Shuffle size={13} />
            Shuffle
          </button>
        )}
      </div>
      <div style={{ opacity: shuffling ? 0.6 : 1, transition: "opacity 120ms" }}>
        {shelves.map((shelf) => (
          <Shelf key={shelf.id} shelf={shelf} onAdd={onAdd} />
        ))}
      </div>
    </div>
  );
}
