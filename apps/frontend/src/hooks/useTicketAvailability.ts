import { useState, useEffect, useRef, useCallback } from "react";
import { bookingApi, TicketCategoryAvailability } from "../modules/booking/booking.api";

export function useTicketAvailability() {
  const [categories, setCategories] = useState<TicketCategoryAvailability[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState<boolean>(false);

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<any>(null);
  const reconnectDelayRef = useRef<number>(1000); // Initial reconnect delay (1s)
  const isTabActiveRef = useRef<boolean>(true);
  const tabInactiveTimeoutRef = useRef<any>(null);

  // Fetch initial counts from REST
  const fetchInitialAvailability = useCallback(async () => {
    try {
      setLoading(true);
      const data = await bookingApi.getAvailability();
      setCategories(data);
      setError(null);
    } catch (err: any) {
      setError(err.message || "Failed to fetch ticket availability");
    } finally {
      setLoading(false);
    }
  }, []);

  // Connect to SSE stream
  const connectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Connect using EventSource. Browser includes cookies by default for same-origin (proxied) requests.
    const url = "/api/v1/tickets/availability/stream";
    const es = new EventSource(url, { withCredentials: true });
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
      reconnectDelayRef.current = 1000; // Reset delay on successful connection
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

      // Exponential backoff reconnection logic (cap at 30 seconds)
      const delay = reconnectDelayRef.current;
      reconnectDelayRef.current = Math.min(delay * 2, 30000);

      // Only attempt reconnect if tab is active
      if (isTabActiveRef.current) {
        if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = setTimeout(() => {
          connectSSE();
        }, delay);
      }
    };
  }, []);

  const disconnectSSE = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsConnected(false);
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

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

    // Initial load: Fetch then connect stream
    fetchInitialAvailability().then(() => {
      connectSSE();
    });

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
    refetch: fetchInitialAvailability,
  };
}
