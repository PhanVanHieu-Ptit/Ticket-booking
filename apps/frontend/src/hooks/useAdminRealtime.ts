import { useCallback, useEffect, useRef, useState } from "react";
import { apiUrl } from "../lib/apiBase";

// Safety-net poll interval while the SSE stream is down.
const POLL_FALLBACK_MS = 20000;
// Coalesces bursts of events (e.g. several holds reclaimed at once) into a single refetch.
const DEBOUNCE_MS = 300;

/**
 * Subscribes to the admin SSE stream (`/api/v1/admin/stream`) and calls
 * `onUpdate` whenever a ticket/hold mutation is broadcast, so the caller can
 * refetch metrics/holds without polling. Falls back to calling `onUpdate` on
 * a slow interval while the stream is disconnected.
 */
export function useAdminRealtime(token: string | null, onUpdate: () => void) {
  const [isConnected, setIsConnected] = useState(false);
  const [isDegraded, setIsDegraded] = useState(false);

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<any>(null);
  const reconnectDelayRef = useRef<number>(1000);
  const debounceTimeoutRef = useRef<any>(null);
  const isTabActiveRef = useRef<boolean>(true);
  const tabInactiveTimeoutRef = useRef<any>(null);
  const pollIntervalRef = useRef<any>(null);

  // Keeps the callback fresh inside closures without re-triggering effects.
  const onUpdateRef = useRef(onUpdate);
  useEffect(() => {
    onUpdateRef.current = onUpdate;
  }, [onUpdate]);

  const triggerUpdate = useCallback(() => {
    if (debounceTimeoutRef.current) clearTimeout(debounceTimeoutRef.current);
    debounceTimeoutRef.current = setTimeout(() => {
      onUpdateRef.current();
    }, DEBOUNCE_MS);
  }, []);

  const startPolling = useCallback(() => {
    if (pollIntervalRef.current) return;
    setIsDegraded(true);
    pollIntervalRef.current = setInterval(() => {
      onUpdateRef.current();
    }, POLL_FALLBACK_MS);
  }, []);

  const stopPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
    setIsDegraded(false);
  }, []);

  const connectSSE = useCallback(
    (currentToken: string) => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }

      const url = apiUrl(`/api/v1/admin/stream?token=${encodeURIComponent(currentToken)}`);
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onopen = () => {
        setIsConnected(true);
        reconnectDelayRef.current = 1000;
        stopPolling();
      };

      es.addEventListener("inventory_update", triggerUpdate);
      es.addEventListener("event_sold_out", triggerUpdate);

      es.onerror = () => {
        setIsConnected(false);
        es.close();

        if (isTabActiveRef.current) {
          startPolling();

          const delay = reconnectDelayRef.current;
          reconnectDelayRef.current = Math.min(delay * 2, 30000);

          if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
          reconnectTimeoutRef.current = setTimeout(() => {
            connectSSE(currentToken);
          }, delay);
        }
      };
    },
    [startPolling, stopPolling, triggerUpdate]
  );

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
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
      debounceTimeoutRef.current = null;
    }
  }, [stopPolling]);

  useEffect(() => {
    if (!token) {
      disconnectSSE();
      return;
    }

    const handleVisibilityChange = () => {
      if (document.hidden) {
        isTabActiveRef.current = false;
        tabInactiveTimeoutRef.current = setTimeout(() => {
          disconnectSSE();
        }, 3 * 60 * 1000);
      } else {
        isTabActiveRef.current = true;
        if (tabInactiveTimeoutRef.current) {
          clearTimeout(tabInactiveTimeoutRef.current);
          tabInactiveTimeoutRef.current = null;
        }
        if (!eventSourceRef.current || eventSourceRef.current.readyState === EventSource.CLOSED) {
          onUpdateRef.current();
          connectSSE(token);
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    connectSSE(token);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      disconnectSSE();
      if (tabInactiveTimeoutRef.current) {
        clearTimeout(tabInactiveTimeoutRef.current);
      }
    };
  }, [token, connectSSE, disconnectSSE]);

  return { isConnected, isDegraded };
}
