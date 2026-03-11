import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

interface ServerEvent {
  type: string;
  timestamp: string;
  movie_id?: string;
  data?: Record<string, unknown>;
}

function buildWsUrl(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/api/v1/ws`;
}

// useWebSocket connects to the server event stream and keeps the React Query
// cache in sync. It reconnects automatically with exponential backoff (1s →
// 2s → 4s → … capped at 30s) so a server restart is handled transparently.
export function useWebSocket() {
  const qc = useQueryClient();
  const retryDelay = useRef(1000);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let stopped = false;

    function connect() {
      if (stopped) return;

      const ws = new WebSocket(buildWsUrl());
      wsRef.current = ws;

      ws.onopen = () => {
        retryDelay.current = 1000; // reset backoff on successful connect
      };

      ws.onmessage = (ev) => {
        let event: ServerEvent;
        try {
          event = JSON.parse(ev.data as string) as ServerEvent;
        } catch {
          return;
        }
        handleEvent(event);
      };

      ws.onclose = () => {
        wsRef.current = null;
        if (stopped) return;
        const delay = retryDelay.current;
        retryDelay.current = Math.min(delay * 2, 30_000);
        timerRef.current = setTimeout(connect, delay);
      };

      ws.onerror = () => {
        // onclose fires after onerror — reconnect is handled there
        ws.close();
      };
    }

    function handleEvent(e: ServerEvent) {
      switch (e.type) {
        case "movie_added":
        case "movie_updated":
        case "movie_deleted":
          qc.invalidateQueries({ queryKey: ["movies"] });
          break;

        case "grab_started":
          qc.invalidateQueries({ queryKey: ["queue"] });
          toast.info("Grab started");
          break;

        case "grab_failed":
          toast.error("Grab failed");
          break;

        case "download_done":
          qc.invalidateQueries({ queryKey: ["queue"] });
          break;

        case "import_complete":
          qc.invalidateQueries({ queryKey: ["movies"] });
          qc.invalidateQueries({ queryKey: ["queue"] });
          toast.success("Import complete");
          break;

        case "import_failed":
          toast.error("Import failed");
          break;

        case "health_issue":
        case "health_ok":
          qc.invalidateQueries({ queryKey: ["system", "health"] });
          break;

        case "task_started":
        case "task_finished":
          qc.invalidateQueries({ queryKey: ["tasks"] });
          break;

        case "bulk_search_complete": {
          const searched = (e.data?.searched as number) ?? 0;
          const grabbed = (e.data?.grabbed as number) ?? 0;
          qc.invalidateQueries({ queryKey: ["queue"] });
          qc.invalidateQueries({ queryKey: ["movies"] });
          qc.invalidateQueries({ queryKey: ["wanted"] });
          toast.success(`Grabbed ${grabbed} of ${searched} movie${searched !== 1 ? "s" : ""}`);
          break;
        }
      }
    }

    connect();

    return () => {
      stopped = true;
      if (timerRef.current !== null) clearTimeout(timerRef.current);
      wsRef.current?.close();
    };
  }, [qc]);
}
