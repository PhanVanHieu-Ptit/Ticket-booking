export interface CheckoutPayload {
  ticketId: number;
  email: string;
  cardHolderName: string;
  paymentMethod: string;
  simulateStatus: "success" | "fail";
}

export interface CheckoutResult {
  orderId: string;
  ticketId: number;
  amount: number;
  paymentReference: string;
  paidAt: string;
}

export const checkoutApi = {
  checkout: async (payload: CheckoutPayload, _idempotencyKey: string): Promise<CheckoutResult> => {
    // Placeholder API call
    return {
      orderId: "",
      ticketId: payload.ticketId,
      amount: 0,
      paymentReference: "",
      paidAt: "",
    };
  },
};
