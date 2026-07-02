import { useState } from "react";
import { CheckoutPayload, CheckoutResult, checkoutApi } from "./checkout.api";

export function useCheckoutState() {
  const [isProcessing, setIsProcessing] = useState(false);
  const [result, setResult] = useState<CheckoutResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const processCheckout = async (payload: CheckoutPayload, idempotencyKey: string) => {
    setIsProcessing(true);
    setError(null);
    try {
      const res = await checkoutApi.checkout(payload, idempotencyKey);
      setResult(res);
      return res;
    } catch (err: any) {
      setError(err.message || "An unexpected error occurred during checkout.");
      throw err;
    } finally {
      setIsProcessing(false);
    }
  };

  const resetState = () => {
    setIsProcessing(false);
    setResult(null);
    setError(null);
  };

  return {
    isProcessing,
    result,
    error,
    processCheckout,
    resetState,
  };
}
