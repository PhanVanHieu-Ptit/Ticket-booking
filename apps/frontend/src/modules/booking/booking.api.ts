export interface TicketCategoryAvailability {
  name: string;
  price: number;
  available: number;
  total: number;
  status: string;
}

export interface ReservationResponse {
  ticketId: number;
  category: string;
  expiresAt: string;
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

  reserveTicket: async (category: string): Promise<ReservationResponse> => {
    // Placeholder API call
    return { ticketId: 0, category, expiresAt: "" };
  },

  cancelHold: async (): Promise<void> => {
    // Placeholder API call
  },
};
