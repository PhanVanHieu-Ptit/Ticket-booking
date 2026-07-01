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
  getAvailability: async (): Promise<TicketCategoryAvailability[]> => {
    const response = await fetch("/api/v1/tickets/availability");
    if (!response.ok) {
      throw new Error("Failed to fetch ticket availability");
    }
    const result: AvailabilityResponse = await response.json();
    return result.data.categories;
  },

  reserveTicket: async (category: string): Promise<ReservationDetails> => {
    const response = await fetch("/api/v1/tickets/reserve", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ category }),
    });
    
    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      throw new Error(errResult.error?.message || "Failed to reserve ticket");
    }
    
    const result: ApiResponse<ReservationDetails> = await response.json();
    return result.data;
  },

  getActiveHold: async (): Promise<ReservationDetails> => {
    const response = await fetch("/api/v1/tickets/hold");
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
    const response = await fetch("/api/v1/tickets/hold/cancel", {
      method: "POST",
    });
    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      throw new Error(errResult.error?.message || "Failed to cancel hold");
    }
  },
};
