import { apiUrl } from "../../lib/apiBase";

export interface AdminMetrics {
  totalTicketsSold: number;
  totalRevenue: number;
  remainingInventory: Record<string, number>;
  heldInventory: Record<string, number>;
  availableInventory: Record<string, number>;
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
    const response = await fetch(apiUrl("/api/v1/admin/login"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ passcode }),
      credentials: "include",
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
    const response = await fetch(apiUrl("/api/v1/admin/metrics"), {
      headers: {
        ...getAuthHeaders(),
      },
      credentials: "include",
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
      remainingInventory: data.remaining_inventory ?? {},
      heldInventory: data.held_inventory ?? {},
      availableInventory: data.available_inventory ?? {},
    };
  },

  getActiveHolds: async (): Promise<ActiveHoldDetail[]> => {
    const response = await fetch(apiUrl("/api/v1/admin/holds"), {
      headers: {
        ...getAuthHeaders(),
      },
      credentials: "include",
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

