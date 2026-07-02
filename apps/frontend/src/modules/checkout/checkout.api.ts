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

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
}

export const checkoutApi = {
  checkout: async (payload: CheckoutPayload, idempotencyKey: string): Promise<CheckoutResult> => {
    const response = await fetch("/api/v1/payments/checkout", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": idempotencyKey,
      },
      body: JSON.stringify({
        ticket_id: payload.ticketId,
        email: payload.email,
        card_holder_name: payload.cardHolderName,
        payment_method: payload.paymentMethod,
        simulate_status: payload.simulateStatus,
      }),
    });

    if (!response.ok) {
      const errResult: ApiResponse<any> = await response.json().catch(() => ({ success: false }));
      const errorObj = new Error(errResult.error?.message || "Checkout failed");
      (errorObj as any).code = errResult.error?.code;
      (errorObj as any).status = response.status;
      throw errorObj;
    }

    const result: ApiResponse<any> = await response.json();
    return {
      orderId: result.data.order_id,
      ticketId: result.data.ticket_id,
      amount: result.data.amount,
      paymentReference: result.data.payment_reference,
      paidAt: result.data.paid_at,
    };
  },
};
