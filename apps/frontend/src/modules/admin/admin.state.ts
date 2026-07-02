import { useState, useCallback } from "react";
import { AdminMetrics, ActiveHoldDetail, adminApi } from "./admin.api";

export function useAdminState() {
  const [token, setTokenState] = useState<string | null>(() => sessionStorage.getItem("admin_token"));
  const [metrics, setMetrics] = useState<AdminMetrics | null>(null);
  const [holds, setHolds] = useState<ActiveHoldDetail[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const setToken = (newToken: string | null) => {
    if (newToken) {
      sessionStorage.setItem("admin_token", newToken);
    } else {
      sessionStorage.removeItem("admin_token");
    }
    setTokenState(newToken);
  };

  const login = async (passcode: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const jwt = await adminApi.login(passcode);
      setToken(jwt);
    } catch (err: any) {
      setError(err.message || "Invalid admin passcode.");
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    setToken(null);
    setMetrics(null);
    setHolds([]);
    setError(null);
  };

  const fetchMetrics = useCallback(async () => {
    if (!sessionStorage.getItem("admin_token")) return;
    try {
      const data = await adminApi.getMetrics();
      setMetrics(data);
    } catch (err: any) {
      if (err.status === 401) {
        logout();
      }
    }
  }, []);

  const fetchHolds = useCallback(async () => {
    if (!sessionStorage.getItem("admin_token")) return;
    try {
      const data = await adminApi.getActiveHolds();
      setHolds(data);
    } catch (err: any) {
      if (err.status === 401) {
        logout();
      }
    }
  }, []);

  return {
    token,
    metrics,
    holds,
    setHolds,
    isLoading,
    error,
    login,
    logout,
    fetchMetrics,
    fetchHolds,
  };
}

