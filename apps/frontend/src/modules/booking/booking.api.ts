import { apiUrl } from "../../lib/apiBase";
import { ensureSessionToken } from "../../lib/sessionToken";

export interface TicketCategoryAvailability {
  name: string;
  price: number;
  available: number;
  total: number;
  status: string;
}

export interface ReservationDetails {
  ticket_id: number;
  ticket_code: string;
  category: string;
  price: number;
  status: string;
  held_at: string;
  expires_at: string;
  seconds_remaining: number;
  server_time?: string;
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

export interface AvailabilityResponse {
  success: boolean;
  data: {
    event_name: string;
    total_capacity: number;
    categories: TicketCategoryAvailability[];
  };
}

export const bookingApi = {
  // Fetches the raw signed session token so it can be attached as
  // ?session_token= on the EventSource URL. Cookies alone aren't reliable
  // for that connection: private/incognito tabs block the cross-site
  // session_token cookie (frontend on Vercel, backend on Render are
  // different origins), so the token also needs to travel in the JSON body.
  getSessionToken: async (): Promise<string> => {
    const response = await fetch(apiUrl("/api/v1/sessions"), {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) {
      throw new Error("Failed to initialize session");
    }
    const result: ApiResponse<{ session_token: string }> = await response.json();
    return result.data.session_token;
  },

  getAvailability: async (): Promise<TicketCategoryAvailability[]> => {
    const response = await fetch(apiUrl("/api/v1/tickets/availability"), {
      credentials: "include",
    });
    if (!response.ok) {
      throw new Error("Failed to fetch ticket availability");
    }
    const result: AvailabilityResponse = await response.json();
    return result.data.categories;
  },

  reserveTicket: async (category: string): Promise<ReservationDetails> => {
    // Reserve, the follow-up hold lookup on /checkout, and the final
    // checkout call must all resolve to the same session. Relying on the
    // cookie alone breaks that in private/incognito tabs (see
    // lib/sessionToken.ts), so the cached token also travels as a header.
    const sessionToken = await ensureSessionToken();

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 8000);

    let response: Response;
    try {
      response = await fetch(apiUrl("/api/v1/tickets/reserve"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Session-Token": sessionToken,
        },
        body: JSON.stringify({ category }),
        signal: controller.signal,
        credentials: "include",
      });
    } catch (err: any) {
      if (err.name === "AbortError") {
        const timeoutErr = new Error("Slow connection, please try again.");
        (timeoutErr as any).code = "TIMEOUT";
        throw timeoutErr;
      }
      throw err;
    } finally {
      clearTimeout(timeoutId);
    }

    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "Failed to reserve ticket");
      (errorObj as any).code = errResult.error?.code;
      throw errorObj;
    }

    const result: ApiResponse<ReservationDetails> = await response.json();
    return result.data;
  },

  getActiveHold: async (): Promise<ReservationDetails> => {
    const sessionToken = await ensureSessionToken();
    const response = await fetch(apiUrl("/api/v1/tickets/hold"), {
      headers: {
        "X-Session-Token": sessionToken,
      },
      credentials: "include",
    });
    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "No active hold found");
      (errorObj as any).code = errResult.error?.code;
      throw errorObj;
    }
    const result: ApiResponse<ReservationDetails> = await response.json();
    return result.data;
  },

  cancelHold: async (): Promise<void> => {
    const sessionToken = await ensureSessionToken();
    const response = await fetch(apiUrl("/api/v1/tickets/hold/cancel"), {
      method: "POST",
      headers: {
        "X-Session-Token": sessionToken,
      },
      credentials: "include",
    });
    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      throw new Error(errResult.error?.message || "Failed to cancel hold");
    }
  },
};
