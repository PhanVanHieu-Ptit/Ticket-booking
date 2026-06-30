export interface AdminMetrics {
  totalTicketsSold: number;
  totalRevenue: number;
  remainingInventory: { VIP: number; Standard: number };
  heldInventory: { VIP: number; Standard: number };
  availableInventory: { VIP: number; Standard: number };
}

export interface ActiveHoldDetail {
  ticketId: number;
  ticketCode: string;
  category: string;
  price: number;
  sessionId: string;
  expiresAt: string;
  secondsRemaining: number;
}

export const adminApi = {
  getMetrics: async (): Promise<AdminMetrics> => {
    // Placeholder API call
    return {
      totalTicketsSold: 0,
      totalRevenue: 0,
      remainingInventory: { VIP: 100, Standard: 400 },
      heldInventory: { VIP: 0, Standard: 0 },
      availableInventory: { VIP: 100, Standard: 400 },
    };
  },

  getActiveHolds: async (): Promise<ActiveHoldDetail[]> => {
    // Placeholder API call
    return [];
  },
};
