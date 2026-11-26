import { Link } from "react-router-dom";
import { Building2, Layers, Tag, User } from "lucide-react";
import { pivotLink, type DiscoverRef, type DiscoverSuggestions } from "@/api/discover";

// What a search matched besides titles: people, collections, keywords and
// studios, each a link to that list of movies.

const chip: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: 6,
  padding: "4px 10px",
  borderRadius: 6,
  border: "1px solid var(--color-border-subtle)",
  background: "var(--color-bg-elevated)",
  color: "var(--color-text-primary)",
  fontSize: 12,
  fontWeight: 500,
  textDecoration: "none",
};

function Group({ label, icon, refs, to }: { label: string; icon: React.ReactNode; refs: DiscoverRef[]; to: (r: DiscoverRef) => string }) {
  if (refs.length === 0) return null;
  return (
    <div role="group" aria-label={label} style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
      <span style={{ display: "inline-flex", alignItems: "center", gap: 5, width: 96, flexShrink: 0, fontSize: 11, fontWeight: 600, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: "0.04em" }}>
        {icon}
        {label}
      </span>
      {refs.map((r) => (
        <Link key={r.id} to={to(r)} style={chip}>
          {r.name}
          {r.detail && <span style={{ color: "var(--color-text-muted)", fontWeight: 400 }}>{r.detail}</span>}
        </Link>
      ))}
    </div>
  );
}

export default function SearchSuggestions({ suggestions }: { suggestions: DiscoverSuggestions }) {
  const { people, collections, keywords, companies } = suggestions;
  if (people.length + collections.length + keywords.length + companies.length === 0) return null;
  return (
    <div data-testid="search-suggestions" style={{ display: "flex", flexDirection: "column", gap: 8, marginBottom: 24 }}>
      <Group label="People" icon={<User size={12} />} refs={people} to={(r) => `/people/${r.id}`} />
      <Group label="Collections" icon={<Layers size={12} />} refs={collections} to={(r) => pivotLink("collection", r)} />
      <Group label="Keywords" icon={<Tag size={12} />} refs={keywords} to={(r) => pivotLink("keyword", r)} />
      <Group label="Studios" icon={<Building2 size={12} />} refs={companies} to={(r) => pivotLink("company", r)} />
    </div>
  );
}
