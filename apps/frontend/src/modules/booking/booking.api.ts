export interface TicketCategoryAvailability {
  category: string;
  price: number;
  available: number;
  total: number;
}

export interface ReservationResponse {
  ticketId: number;
  category: string;
  expiresAt: string;
}

export const bookingApi = {
  getAvailability: async (): Promise<TicketCategoryAvailability[]> => {
    // Placeholder API call
    return [];
  },

  reserveTicket: async (category: string): Promise<ReservationResponse> => {
    // Placeholder API call
    return { ticketId: 0, category, expiresAt: "" };
  },

  cancelHold: async (): Promise<void> => {
    // Placeholder API call
  },
};
