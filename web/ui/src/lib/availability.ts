// Where a movie stands, from the viewer's side. The server works this out
// from the media server, the download queue and the disk; this file gives
// each state its words and colour.

export type Availability =
  | "available"
  | "downloaded"
  | "downloading"
  | "wanted"
  | "upcoming"
  | "missing"
  | "unmonitored";

interface AvailabilityStyle {
  label: string;
  hint: string;
  color: string; // a CSS colour, from the theme
}

export const AVAILABILITY: Record<Availability, AvailabilityStyle> = {
  available: { label: "Available", hint: "On your media server, ready to watch", color: "var(--color-success)" },
  downloaded: { label: "Downloaded", hint: "On disk, but not on your media server yet", color: "var(--color-accent)" },
  downloading: { label: "Downloading", hint: "A download is in progress", color: "var(--color-accent)" },
  wanted: { label: "Wanted", hint: "Released; Prism is looking for it", color: "var(--color-warning)" },
  upcoming: { label: "Upcoming", hint: "Not released yet", color: "var(--color-text-muted)" },
  missing: { label: "Missing", hint: "It was downloaded, but the file is gone", color: "var(--color-danger)" },
  unmonitored: { label: "Not monitored", hint: "Prism isn't looking for it", color: "var(--color-text-muted)" },
};

// The order states are listed in, from done to not started.
export const AVAILABILITY_ORDER = Object.keys(AVAILABILITY) as Availability[];

// availabilityOf returns a movie's state. Servers older than this field
// send only `status`, which is mapped as closely as it can be.
export function availabilityOf(movie: { availability?: string; status: string; monitored: boolean }): Availability {
  if (movie.availability && movie.availability in AVAILABILITY) return movie.availability as Availability;
  if (movie.status === "downloaded") return "downloaded";
  if (movie.status === "missing") return "missing";
  if (movie.status === "announced") return "upcoming";
  return movie.monitored ? "wanted" : "unmonitored";
}

// availabilityLabel is the badge text: the state, with the percentage
// while downloading.
export function availabilityLabel(state: Availability, progress?: number): string {
  if (state === "downloading" && progress !== undefined && progress > 0) {
    return `Downloading ${Math.floor(progress * 100)}%`;
  }
  return AVAILABILITY[state].label;
}
