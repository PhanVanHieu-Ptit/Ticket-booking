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

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
}

const getAuthHeaders = (): Record<string, string> => {
  const token = sessionStorage.getItem("admin_token");
  return token ? { "Authorization": `Bearer ${token}` } : {};
};

export const adminApi = {
  login: async (passcode: string): Promise<string> => {
    const response = await fetch("/api/v1/admin/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ passcode }),
    });

    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "Login failed");
      (errorObj as any).code = errResult.error?.code;
      (errorObj as any).status = response.status;
      throw errorObj;
    }

    const result: ApiResponse<{ token: string; expires_at: string }> = await response.json();
    return result.data.token;
  },

  getMetrics: async (): Promise<AdminMetrics> => {
    const response = await fetch("/api/v1/admin/metrics", {
      headers: {
        ...getAuthHeaders(),
      },
    });

    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "Failed to fetch metrics");
      (errorObj as any).code = errResult.error?.code;
      (errorObj as any).status = response.status;
      throw errorObj;
    }

    const result: ApiResponse<any> = await response.json();
    const data = result.data;
    return {
      totalTicketsSold: data.total_tickets_sold,
      totalRevenue: data.total_revenue,
      remainingInventory: {
        VIP: data.remaining_inventory.VIP,
        Standard: data.remaining_inventory.Standard,
      },
      heldInventory: {
        VIP: data.held_inventory.VIP,
        Standard: data.held_inventory.Standard,
      },
      availableInventory: {
        VIP: data.available_inventory.VIP,
        Standard: data.available_inventory.Standard,
      },
    };
  },

  getActiveHolds: async (): Promise<ActiveHoldDetail[]> => {
    const response = await fetch("/api/v1/admin/holds", {
      headers: {
        ...getAuthHeaders(),
      },
    });

    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "Failed to fetch holds");
      (errorObj as any).code = errResult.error?.code;
      (errorObj as any).status = response.status;
      throw errorObj;
    }

    const result: ApiResponse<any[]> = await response.json();
    return result.data.map((item: any) => ({
      ticketId: item.ticket_id,
      ticketCode: item.ticket_code,
      category: item.category,
      price: item.price,
      sessionId: item.session_id,
      expiresAt: item.expires_at,
      secondsRemaining: item.seconds_remaining,
    }));
  },
};

