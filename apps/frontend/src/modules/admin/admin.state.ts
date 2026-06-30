import { useState } from "react";
import { AdminMetrics, ActiveHoldDetail } from "./admin.api";

export function useAdminState() {
  const [metrics] = useState<AdminMetrics | null>(null);
  const [holds] = useState<ActiveHoldDetail[]>([]);
  const [isLoading] = useState(false);

  const fetchMetrics = async () => {
    // Placeholder metrics loader
  };

  const fetchHolds = async () => {
    // Placeholder holds loader
  };

  return {
    metrics,
    holds,
    isLoading,
    fetchMetrics,
    fetchHolds,
  };
}
