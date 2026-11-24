import { useState, useRef, useEffect } from "react";
import { useCreatePlexPin, checkPlexPin } from "@/api/importlists";
import { useSearchPeople, useSearchAll } from "@/api/collections";
import { inputStyle, labelStyle, fieldStyle } from "@/lib/styles";
import type { FormState } from "./ImportListList";

interface SubFormProps {
  form: FormState;
  set: <K extends keyof FormState>(field: K, value: FormState[K]) => void;
  editing: boolean;
  focusBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
  blurBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
}

function TMDbListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>TMDb List ID *</label>
      <input
        style={inputStyle}
        value={form.tmdb_list_id}
        onChange={(e) => set("tmdb_list_id", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
        placeholder="e.g. 8136823"
      />
      <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
        The numeric ID from a TMDb list URL.
      </p>
    </div>
  );
}

function TMDbPersonSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const { data: results } = useSearchPeople(query);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const hasSelection = form.tmdb_person_id && form.tmdb_person_id !== "0";

  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Person *</label>
        {hasSelection ? (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              padding: "8px 12px",
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
            }}
          >
            <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
              {form.tmdb_person_name || `TMDb #${form.tmdb_person_id}`}
              <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: 8 }}>
                ID: {form.tmdb_person_id}
              </span>
            </span>
            <button
              type="button"
              style={{
                background: "none",
                border: "none",
                color: "var(--color-accent)",
                cursor: "pointer",
                fontSize: 12,
                padding: 0,
              }}
              onClick={() => {
                set("tmdb_person_id", "");
                set("tmdb_person_name", "");
                setQuery("");
              }}
            >
              Change
            </button>
          </div>
        ) : (
          <div ref={wrapRef} style={{ position: "relative" }}>
            <input
              style={inputStyle}
              value={query}
              onChange={(e) => {
                setQuery(e.currentTarget.value);
                setOpen(true);
              }}
              onFocus={(e) => {
                focusBorder(e);
                if (query.trim().length >= 2) setOpen(true);
              }}
              onBlur={blurBorder}
              placeholder="Search for a person..."
            />
            {open && results && results.length > 0 && (
              <div
                style={{
                  position: "absolute",
                  top: "100%",
                  left: 0,
                  right: 0,
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  marginTop: 4,
                  maxHeight: 220,
                  overflowY: "auto",
                  zIndex: 10,
                  boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
                }}
              >
                {results.map((p) => (
                  <PersonDropdownItem
                    key={p.person_id}
                    name={p.name}
                    dept={p.known_for_department}
                    imgPath={p.profile_path}
                    onSelect={() => {
                      set("tmdb_person_id", String(p.person_id));
                      set("tmdb_person_name", p.name);
                      setOpen(false);
                      setQuery("");
                    }}
                  />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>Credit Type</label>
        <select
          style={{ ...inputStyle, cursor: "pointer" }}
          value={form.tmdb_person_type}
          onChange={(e) => set("tmdb_person_type", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
        >
          <option value="actor">Actor (Cast)</option>
          <option value="director">Director</option>
          <option value="writer">Writer</option>
          <option value="producer">Producer</option>
        </select>
      </div>
    </>
  );
}

function PersonDropdownItem({
  name,
  dept,
  imgPath,
  onSelect,
}: {
  name: string;
  dept: string;
  imgPath: string;
  onSelect: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const img = imgPath ? `https://image.tmdb.org/t/p/w45${imgPath}` : "";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 12px",
        cursor: "pointer",
        background: hovered ? "var(--color-bg-surface)" : "transparent",
        transition: "background 0.1s",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onSelect}
    >
      {img ? (
        <img
          src={img}
          alt=""
          style={{ width: 30, height: 30, borderRadius: "50%", objectFit: "cover", flexShrink: 0 }}
        />
      ) : (
        <div
          style={{
            width: 30,
            height: 30,
            borderRadius: "50%",
            background: "var(--color-bg-surface)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>{name}</div>
        {dept && (
          <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{dept}</div>
        )}
      </div>
    </div>
  );
}

function TMDbCollectionSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const { data: allResults } = useSearchAll(query);
  const results = allResults?.filter((r) => r.result_type === "franchise");
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const hasSelection = form.tmdb_collection_id && form.tmdb_collection_id !== "0";

  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Collection *</label>
      {hasSelection ? (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "8px 12px",
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
          }}
        >
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            {form.tmdb_collection_name || `TMDb #${form.tmdb_collection_id}`}
            <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: 8 }}>
              ID: {form.tmdb_collection_id}
            </span>
          </span>
          <button
            type="button"
            style={{
              background: "none",
              border: "none",
              color: "var(--color-accent)",
              cursor: "pointer",
              fontSize: 12,
              padding: 0,
            }}
            onClick={() => {
              set("tmdb_collection_id", "");
              set("tmdb_collection_name", "");
              setQuery("");
            }}
          >
            Change
          </button>
        </div>
      ) : (
        <div ref={wrapRef} style={{ position: "relative" }}>
          <input
            style={inputStyle}
            value={query}
            onChange={(e) => {
              setQuery(e.currentTarget.value);
              setOpen(true);
            }}
            onFocus={(e) => {
              focusBorder(e);
              if (query.trim().length >= 2) setOpen(true);
            }}
            onBlur={blurBorder}
            placeholder="Search for a collection..."
          />
          {open && results && results.length > 0 && (
            <div
              style={{
                position: "absolute",
                top: "100%",
                left: 0,
                right: 0,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                marginTop: 4,
                maxHeight: 220,
                overflowY: "auto",
                zIndex: 10,
                boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
              }}
            >
              {results.map((c) => (
                <CollectionDropdownItem
                  key={c.id}
                  name={c.name}
                  subtitle={c.subtitle}
                  imgPath={c.image_path}
                  onSelect={() => {
                    set("tmdb_collection_id", String(c.id));
                    set("tmdb_collection_name", c.name);
                    setOpen(false);
                    setQuery("");
                  }}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function CollectionDropdownItem({
  name,
  subtitle,
  imgPath,
  onSelect,
}: {
  name: string;
  subtitle: string;
  imgPath: string;
  onSelect: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const img = imgPath ? `https://image.tmdb.org/t/p/w92${imgPath}` : "";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 12px",
        cursor: "pointer",
        background: hovered ? "var(--color-bg-surface)" : "transparent",
        transition: "background 0.1s",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onSelect}
    >
      {img ? (
        <img
          src={img}
          alt=""
          style={{ width: 36, height: 36, borderRadius: 4, objectFit: "cover", flexShrink: 0 }}
        />
      ) : (
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: 4,
            background: "var(--color-bg-surface)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>{name}</div>
        {subtitle && (
          <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{subtitle}</div>
        )}
      </div>
    </div>
  );
}

function TraktListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Username *</label>
        <input
          style={inputStyle}
          value={form.trakt_username}
          onChange={(e) => set("trakt_username", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="Trakt username"
        />
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>List Type</label>
        <select
          style={{ ...inputStyle, cursor: "pointer" }}
          value={form.trakt_list_type}
          onChange={(e) => set("trakt_list_type", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
        >
          <option value="watchlist">Watchlist</option>
          <option value="custom">Custom List</option>
        </select>
      </div>
      {form.trakt_list_type === "custom" && (
        <div style={fieldStyle}>
          <label style={labelStyle}>List Slug *</label>
          <input
            style={inputStyle}
            value={form.trakt_list_slug}
            onChange={(e) => set("trakt_list_slug", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="e.g. my-favorite-movies"
          />
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            The slug from the Trakt list URL.
          </p>
        </div>
      )}
    </>
  );
}

function PlexWatchlistSettings({ form, set, editing }: SubFormProps) {
  const createPin = useCreatePlexPin();
  const [polling, setPolling] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const hasToken = form.plex_access_token.length > 0;

  function startPlexAuth() {
    createPin.mutate(undefined, {
      onSuccess: (pin) => {
        window.open(pin.auth_url, "_blank", "noopener");
        setPolling(true);
        let attempts = 0;
        pollRef.current = setInterval(async () => {
          attempts++;
          try {
            const status = await checkPlexPin(pin.id);
            if (status.claimed && status.token) {
              set("plex_access_token", status.token);
              setPolling(false);
              if (pollRef.current) clearInterval(pollRef.current);
            } else if (attempts >= 60) {
              setPolling(false);
              if (pollRef.current) clearInterval(pollRef.current);
            }
          } catch {
            setPolling(false);
            if (pollRef.current) clearInterval(pollRef.current);
          }
        }, 2000);
      },
    });
  }

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  if (hasToken && !polling) {
    return (
      <div style={fieldStyle}>
        <label style={labelStyle}>Plex Account</label>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "8px 12px",
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
          }}
        >
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            {editing ? "Plex account linked (token saved)" : "Plex account linked"}
          </span>
          <button
            type="button"
            style={{
              background: "none",
              border: "none",
              color: "var(--color-accent)",
              cursor: "pointer",
              fontSize: 12,
              padding: 0,
            }}
            onClick={() => set("plex_access_token", "")}
          >
            Re-authenticate
          </button>
        </div>
      </div>
    );
  }

  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Plex Account *</label>
      <button
        type="button"
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 8,
          width: "100%",
          padding: "10px 16px",
          background: polling ? "var(--color-bg-elevated)" : "#e5a00d",
          color: polling ? "var(--color-text-secondary)" : "#1a1a1a",
          border: "1px solid var(--color-border-default)",
          borderRadius: 6,
          fontSize: 13,
          fontWeight: 600,
          cursor: polling || createPin.isPending ? "default" : "pointer",
          opacity: createPin.isPending ? 0.6 : 1,
        }}
        onClick={startPlexAuth}
        disabled={polling || createPin.isPending}
      >
        {polling
          ? "Waiting for Plex authorization..."
          : createPin.isPending
            ? "Creating link..."
            : "Sign in with Plex"}
      </button>
      {polling && (
        <p style={{ margin: "6px 0 0", fontSize: 11, color: "var(--color-text-muted)", textAlign: "center" }}>
          A Plex sign-in window has opened. Complete the sign-in there, then return here.
        </p>
      )}
    </div>
  );
}

function MDBListSettings({ form, set, editing, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>API Key *</label>
        <input
          style={inputStyle}
          type="password"
          value={form.mdblist_api_key}
          onChange={(e) => set("mdblist_api_key", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder={editing ? "enter to change" : "Your MDBList API key"}
          autoComplete="new-password"
        />
        {editing && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            API key is masked. Enter a new value to update.
          </p>
        )}
        <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          Get your API key from mdblist.com/preferences
        </p>
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>List ID *</label>
        <input
          style={inputStyle}
          value={form.mdblist_list_id}
          onChange={(e) => set("mdblist_list_id", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="e.g. 12345"
        />
        <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          The numeric ID from the MDBList list URL.
        </p>
      </div>
    </>
  );
}

function CustomListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>URL *</label>
      <input
        style={inputStyle}
        value={form.custom_list_url}
        onChange={(e) => set("custom_list_url", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
        placeholder='https://example.com/movies.json'
      />
      <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
        URL returning a JSON array. Each object needs a <code style={{ fontSize: 11 }}>tmdb</code> or <code style={{ fontSize: 11 }}>tmdb_id</code> field.
      </p>
    </div>
  );
}

function TMDbTrendingSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Time Window</label>
      <select
        style={{ ...inputStyle, cursor: "pointer" }}
        value={form.tmdb_trending_window}
        onChange={(e) => set("tmdb_trending_window", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
      >
        <option value="day">Today</option>
        <option value="week">This Week</option>
      </select>
    </div>
  );
}

export default function KindSettings(props: SubFormProps) {
  switch (props.form.kind) {
    case "tmdb_popular":
    case "tmdb_upcoming":
    case "tmdb_now_playing":
    case "tmdb_top_rated":
      return null;
    case "tmdb_trending":
      return <TMDbTrendingSettings {...props} />;
    case "tmdb_list":
      return <TMDbListSettings {...props} />;
    case "tmdb_person":
      return <TMDbPersonSettings {...props} />;
    case "tmdb_collection":
      return <TMDbCollectionSettings {...props} />;
    case "trakt_popular":
    case "trakt_trending":
    case "trakt_anticipated":
    case "trakt_box_office":
    case "stevenlu":
      return null;
    case "trakt_list":
      return <TraktListSettings {...props} />;
    case "plex_watchlist":
      return <PlexWatchlistSettings {...props} />;
    case "mdblist":
      return <MDBListSettings {...props} />;
    case "custom_list":
      return <CustomListSettings {...props} />;
    default:
      return null;
  }
}
