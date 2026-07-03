import { useState, useEffect, useRef, useCallback } from "react";
import { bookingApi, TicketCategoryAvailability } from "../modules/booking/booking.api";
import { apiUrl } from "../lib/apiBase";

const POLL_INTERVAL_MS = 8000;

export function useTicketAvailability() {
  const [categories, setCategories] = useState<TicketCategoryAvailability[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState<boolean>(false);
  // True while we've fallen back to REST polling because the SSE stream is down.
  const [isDegraded, setIsDegraded] = useState<boolean>(false);

  const eventSourceRef = useRef<EventSource | null>(null);
  // Cached session token used to authenticate the SSE connection via a
  // ?session_token= query param, since the session_token cookie alone isn't
  // reliable cross-origin in private/incognito tabs (see connectSSE below).
  const sessionTokenRef = useRef<string | null>(null);
  // Holds the latest connectSSE closure so scheduleReconnect (defined before
  // connectSSE, to break their mutual dependency) always retries with it.
  const connectSSERef = useRef<() => void>(() => {});
  const reconnectTimeoutRef = useRef<any>(null);
  const reconnectDelayRef = useRef<number>(1000); // Initial reconnect delay (1s)
  const isTabActiveRef = useRef<boolean>(true);
  const tabInactiveTimeoutRef = useRef<any>(null);
  const pollIntervalRef = useRef<any>(null);
  const didInitRef = useRef<boolean>(false);
  // Mirrors `categories` for synchronous reads inside the SSE onopen handler
  // (see connectSSE below), which closes over stale state otherwise.
  const categoriesRef = useRef<TicketCategoryAvailability[]>([]);
  useEffect(() => {
    categoriesRef.current = categories;
  }, [categories]);

  // Fetch initial counts from REST. Returns whether it actually populated data.
  const fetchInitialAvailability = useCallback(async (): Promise<boolean> => {
    try {
      setLoading(true);
      const data = await bookingApi.getAvailability();
      setCategories(data);
      setError(null);
      return true;
    } catch (err: any) {
      setError(err.message || "Failed to fetch ticket availability");
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  // Fall back to polling the REST endpoint while the SSE stream is unavailable.
  const startPolling = useCallback(() => {
    if (pollIntervalRef.current) return;
    setIsDegraded(true);
    pollIntervalRef.current = setInterval(() => {
      fetchInitialAvailability();
    }, POLL_INTERVAL_MS);
  }, [fetchInitialAvailability]);

  const stopPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
    setIsDegraded(false);
  }, []);

  // Retry connectSSE with exponential backoff (cap at 30s), same policy used
  // for a dropped EventSource connection and for a failed token fetch.
  const scheduleReconnect = useCallback(() => {
    if (!isTabActiveRef.current) return;

    // Keep counts fresh via REST polling while the live stream is down
    startPolling();

    const delay = reconnectDelayRef.current;
    reconnectDelayRef.current = Math.min(delay * 2, 30000);

    if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
    reconnectTimeoutRef.current = setTimeout(() => {
      connectSSERef.current();
    }, delay);
  }, [startPolling]);

  // Connect to SSE stream
  const connectSSE = useCallback(async () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Fetch (and cache) a session token to pass explicitly as a query param.
    // withCredentials still sends the session_token cookie when the browser
    // allows it, but private/incognito tabs block that cookie outright since
    // frontend (Vercel) and backend (Render) are different origins — the
    // query param is what keeps the stream authenticated in that case.
    if (!sessionTokenRef.current) {
      try {
        sessionTokenRef.current = await bookingApi.getSessionToken();
      } catch (err) {
        console.error("Failed to fetch session token for SSE stream:", err);
        scheduleReconnect();
        return;
      }
    }

    const url = apiUrl(
      `/api/v1/tickets/availability/stream?session_token=${encodeURIComponent(sessionTokenRef.current)}`
    );
    const es = new EventSource(url, { withCredentials: true });
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
      reconnectDelayRef.current = 1000; // Reset delay on successful connection

      if (categoriesRef.current.length > 0) {
        stopPolling(); // Live stream is back, no need to keep polling
      } else {
        // We've never had a successful initial REST fetch (e.g. it failed
        // while the network was down, before this connection came up).
        // initial_state/inventory_update below only patch existing entries
        // (see prev.map() in each handler), so there's nothing for them to
        // patch onto. Re-fetch now that the network is back, and only stop
        // polling once that fetch actually populates data.
        fetchInitialAvailability().then((ok) => {
          if (ok) stopPolling();
        });
      }
    };

    es.addEventListener("initial_state", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        // data format: {"VIP": {"available": X, "status": Y}, "Standard": {"available": A, "status": B}}
        setCategories((prev) =>
          prev.map((cat) => {
            const update = data[cat.name];
            if (update) {
              return {
                ...cat,
                available: update.available,
                status: update.status,
              };
            }
            return cat;
          })
        );
      } catch (err) {
        console.error("Failed to parse initial_state event data", err);
      }
    });

    es.addEventListener("inventory_update", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        // data format: {"category": "VIP", "available": X, "status": Y}
        setCategories((prev) =>
          prev.map((cat) => {
            if (cat.name === data.category) {
              return {
                ...cat,
                available: data.available,
                status: data.status,
              };
            }
            return cat;
          })
        );
      } catch (err) {
        console.error("Failed to parse inventory_update event data", err);
      }
    });

    es.addEventListener("event_sold_out", () => {
      // Set all availability to 0 and status to Sold Out
      setCategories((prev) =>
        prev.map((cat) => ({
          ...cat,
          available: 0,
          status: "Sold Out",
        }))
      );
    });

    es.onerror = (e) => {
      console.error("SSE connection error:", e);
      setIsConnected(false);
      es.close();
      // Drop the cached token: the error may be an expired/rejected session
      // token (EventSource exposes no status code), so the next reconnect
      // attempt should fetch a fresh one rather than retry the same one forever.
      sessionTokenRef.current = null;
      scheduleReconnect();
    };
  }, [scheduleReconnect, stopPolling, fetchInitialAvailability]);
  connectSSERef.current = connectSSE;

  const disconnectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsConnected(false);
    stopPolling();
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, [stopPolling]);

  // Handle visibility changes (Page Visibility API)
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        isTabActiveRef.current = false;
        // Start 3-minute (180,000ms) inactive window before closing connection
        tabInactiveTimeoutRef.current = setTimeout(() => {
          disconnectSSE();
        }, 3 * 60 * 1000);
      } else {
        isTabActiveRef.current = true;
        if (tabInactiveTimeoutRef.current) {
          clearTimeout(tabInactiveTimeoutRef.current);
          tabInactiveTimeoutRef.current = null;
        }
        // If connection is closed or not initialized, reconnect and refresh counts immediately
        if (!eventSourceRef.current || eventSourceRef.current.readyState === EventSource.CLOSED) {
          fetchInitialAvailability().then(() => {
            connectSSE();
          });
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    if (!didInitRef.current) {
      // Initial load: Fetch then connect stream
      didInitRef.current = true;
      fetchInitialAvailability().then(() => {
        connectSSE();
      });
    } else {
      // React.StrictMode re-invoked this effect after its phantom mount's
      // cleanup already closed the SSE connection below — reconnect without
      // re-issuing the REST fetch a second time.
      connectSSE();
    }

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      disconnectSSE();
      if (tabInactiveTimeoutRef.current) {
        clearTimeout(tabInactiveTimeoutRef.current);
      }
    };
  }, [fetchInitialAvailability, connectSSE, disconnectSSE]);

  return {
    categories,
    loading,
    error,
    isConnected,
    isDegraded,
    refetch: fetchInitialAvailability,
  };
}
