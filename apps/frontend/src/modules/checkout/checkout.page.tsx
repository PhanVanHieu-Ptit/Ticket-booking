import React, { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Clock, CreditCard, User, Mail, ShieldCheck, AlertTriangle } from "lucide-react";
import { CheckoutLayout } from "./checkout.layout";
import { bookingApi, ReservationDetails } from "../booking/booking.api";
import { useCheckoutState } from "./checkout.state";

interface ExpirationModalProps {
  onClose: () => void;
}

const ExpirationModal: React.FC<ExpirationModalProps> = ({ onClose }) => {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-md animate-fade-in">
      <div className="w-full max-w-md glass border border-red-500/20 p-8 rounded-3xl text-center space-y-6 shadow-2xl shadow-red-950/20 animate-scale-in">
        <div className="mx-auto w-16 h-16 bg-red-500/10 border border-red-500/30 rounded-2xl flex items-center justify-center">
          <AlertTriangle className="w-8 h-8 text-red-400 animate-bounce" />
        </div>
        <div className="space-y-2">
          <h3 className="text-2xl font-extrabold tracking-tight text-white">Hold Expired</h3>
          <p className="text-neutral-400 text-sm leading-relaxed">
            Your reservation has expired, and your ticket has been released back to the pool.
          </p>
        </div>
        <button
          onClick={onClose}
          className="w-full py-3 px-4 font-bold bg-primary hover:bg-primary/95 text-white rounded-xl transition-all shadow-lg shadow-primary/20 hover:scale-[1.02]"
        >
          Return to Home Page
        </button>
      </div>
    </div>
  );
};

// Simple, robust helper to generate a valid UUIDv4 string without external dependencies
function generateUUID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export const CheckoutPage: React.FC = () => {
  const navigate = useNavigate();
  const [holdDetails, setHoldDetails] = useState<ReservationDetails | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number>(300);
  const [loading, setLoading] = useState<boolean>(true);
  const [expired, setExpired] = useState<boolean>(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Countdown source of truth: the server's own seconds_remaining as of the
  // last sync. Ticking decrements this by exactly 1 on each setInterval
  // callback instead of computing elapsed time from any clock reading --
  // neither Date.now() nor performance.now() are read here, since on some
  // browser/OS combinations performance.now() has been observed to track
  // the OS wall clock too (despite the spec calling for it to be immune).
  // setInterval's own scheduling doesn't need a clock reading to know "a
  // second passed", so this stays correct regardless of clock changes.
  const remainingRef = useRef<number>(0);
  const verifyingExpiryRef = useRef<boolean>(false);
  const didFetchHoldRef = useRef<boolean>(false);
  const [email, setEmail] = useState<string>("");
  const [cardName, setCardName] = useState<string>("");
  const [cardNumber, setCardNumber] = useState<string>("");
  const [cardExpiry, setCardExpiry] = useState<string>("");
  const [cardCvv, setCardCvv] = useState<string>("");
  const [simulateStatus, setSimulateStatus] = useState<"success" | "fail">("success");
  
  const [validationError, setValidationError] = useState<string | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState<boolean>(false);

  const { isProcessing, processCheckout } = useCheckoutState();

  // Kept in refs so the countdown's interval callback always sees the latest
  // values without needing to be torn down/recreated on every change.
  const isProcessingRef = useRef(isProcessing);
  useEffect(() => {
    isProcessingRef.current = isProcessing;
  }, [isProcessing]);
  const cancellingRef = useRef(cancelling);
  useEffect(() => {
    cancellingRef.current = cancelling;
  }, [cancelling]);

  const handleCardNumberChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValidationError(null);
    setCheckoutError(null);
    const clean = e.target.value.replace(/\D/g, "");
    const formatted = clean.match(/.{1,4}/g)?.join(" ") || clean;
    setCardNumber(formatted.slice(0, 19));
  };

  const handleExpiryChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValidationError(null);
    setCheckoutError(null);
    const clean = e.target.value.replace(/\D/g, "");
    if (clean.length <= 2) {
      setCardExpiry(clean);
    } else {
      setCardExpiry(`${clean.slice(0, 2)}/${clean.slice(2, 4)}`);
    }
  };

  const handleCvvChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValidationError(null);
    setCheckoutError(null);
    const clean = e.target.value.replace(/\D/g, "");
    setCardCvv(clean.slice(0, 4));
  };

  const handleCancelReservation = async () => {
    setCancelling(true);
    try {
      await bookingApi.cancelHold();
      navigate("/", { state: { message: "Your reservation has been cancelled successfully." } });
    } catch (err: any) {
      alert(err.message || "Failed to cancel reservation.");
    } finally {
      setCancelling(false);
    }
  };

  // Applies a fresh hold response: trusts the server's own seconds_remaining
  // as ground truth, re-anchoring the countdown to it.
  const applyHoldDetails = (details: ReservationDetails) => {
    const baseline = Math.max(0, Math.round(details.seconds_remaining));
    remainingRef.current = baseline;

    setHoldDetails(details);
    setSecondsRemaining(baseline);
    setExpired(baseline <= 0);
  };

  const fetchHold = async () => {
    try {
      const details = await bookingApi.getActiveHold();
      setLoadError(null);
      applyHoldDetails(details);
    } catch (err: any) {
      if (err.code === "NO_ACTIVE_HOLD") {
        // Genuinely no reservation for this session — safe to send home.
        navigate("/");
      } else {
        // Any other failure (network blip, 5xx, etc.) — surface it instead of
        // silently bouncing the user back to the home page.
        setLoadError(err.message || "Failed to load your reservation. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Guard against React.StrictMode's mount -> cleanup -> re-mount cycle
    // firing this twice in development.
    if (!didFetchHoldRef.current) {
      didFetchHoldRef.current = true;
      fetchHold();
    }

    // Re-sync on window focus/visibility changes to keep the timer strictly aligned
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        fetchHold();
      }
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, []);

  useEffect(() => {
    if (loading || expired) return;

    const timer = setInterval(() => {
      remainingRef.current = Math.max(0, remainingRef.current - 1);
      const remaining = remainingRef.current;

      if (remaining > 0) {
        setSecondsRemaining(remaining);
        return;
      }

      // Countdown hit zero locally — don't trust it blindly (the tab may have
      // been backgrounded, or the hold could have just been extended/consumed
      // by a payment that's still in flight). Confirm with the server first.
      if (verifyingExpiryRef.current || isProcessingRef.current || cancellingRef.current) {
        return;
      }

      verifyingExpiryRef.current = true;
      bookingApi
        .getActiveHold()
        .then((details) => {
          // Server still reports an active hold (e.g. it was extended) — resync instead of expiring.
          applyHoldDetails(details);
        })
        .catch(() => {
          setSecondsRemaining(0);
          setExpired(true);
        })
        .finally(() => {
          verifyingExpiryRef.current = false;
        });
    }, 1000);

    return () => clearInterval(timer);
  }, [loading, expired]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  const handleReturnHome = () => {
    navigate("/");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!holdDetails) return;

    // Validate email
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      setValidationError("Please enter a valid email address.");
      return;
    }

    // Validate Cardholder Name
    if (!cardName.trim()) {
      setValidationError("Cardholder name is required.");
      return;
    }

    // Validate Card Number (16 digits)
    const cleanCard = cardNumber.replace(/\s/g, "");
    if (!/^\d{16}$/.test(cleanCard)) {
      setValidationError("Card number must be 16 digits.");
      return;
    }

    // Validate Expiration (MM/YY)
    if (!/^(0[1-9]|1[0-2])\/([0-9]{2})$/.test(cardExpiry)) {
      setValidationError("Expiration date must be in MM/YY format.");
      return;
    }

    // Validate CVV (3 or 4 digits)
    if (!/^\d{3,4}$/.test(cardCvv)) {
      setValidationError("CVV must be 3 or 4 digits.");
      return;
    }

    setValidationError(null);
    setCheckoutError(null);

    const idempotencyKey = generateUUID();

    try {
      const payload = {
        ticketId: holdDetails.ticket_id,
        email,
        cardHolderName: cardName,
        paymentMethod: "simulated",
        simulateStatus,
      };

      const result = await processCheckout(payload, idempotencyKey);
      navigate("/confirmation", { state: { orderResult: result, holdDetails } });
    } catch (err: any) {
      if (err.status === 410 || err.code === "RESERVATION_EXPIRED" || err.code === "HOLD_EXPIRED") {
        setExpired(true);
      } else if (err.code === "TICKET_SOLD_OUT" || err.code === "TICKET_UNAVAILABLE") {
        setCheckoutError("Sorry, this ticket just sold out.");
      } else if (err.code === "TIMEOUT") {
        setCheckoutError("Slow connection, please try again.");
      } else {
        setCheckoutError(err.message || "Payment failed. Please check your card details and try again.");
      }
    }
  };

  if (loading) {
    return (
      <CheckoutLayout>
        <div className="grid gap-6 lg:gap-8 lg:grid-cols-3 items-start animate-pulse" aria-busy="true" aria-label="Loading secure checkout">
          {/* Left Side: Form skeleton */}
          <div className="lg:col-span-2 order-2 lg:order-1">
            <div className="p-4 sm:p-6 md:p-8 rounded-3xl border border-white/5 glass-premium space-y-6">
              <div className="flex justify-between items-center pb-4 border-b border-white/5">
                <div className="h-6 w-40 bg-white/10 rounded" />
                <div className="h-6 w-28 bg-white/10 rounded-full" />
              </div>
              <div className="space-y-5">
                {[1, 2, 3, 4, 5].map((i) => (
                  <div key={i} className="space-y-2">
                    <div className="h-4 w-32 bg-white/10 rounded" />
                    <div className="h-12 w-full bg-white/5 rounded-xl" />
                  </div>
                ))}
                <div className="h-14 w-full bg-white/10 rounded-xl" />
                <div className="h-12 w-full bg-white/5 rounded-xl" />
              </div>
            </div>
          </div>

          {/* Right Side: Order summary skeleton */}
          <div className="space-y-6 order-1 lg:order-2">
            <div className="p-4 sm:p-6 rounded-3xl border border-white/5 glass space-y-4">
              <div className="h-5 w-32 bg-white/10 rounded" />
              <div className="h-4 w-full bg-white/5 rounded" />
              <div className="h-4 w-2/3 bg-white/5 rounded" />
              <div className="border-t border-white/5 pt-4">
                <div className="h-6 w-full bg-white/10 rounded" />
              </div>
            </div>
            <div className="h-14 w-full rounded-2xl border border-white/5 bg-white/5" />
          </div>
        </div>
      </CheckoutLayout>
    );
  }

  if (loadError) {
    return (
      <CheckoutLayout>
        <div className="flex flex-col items-center justify-center py-20 space-y-4 text-center">
          <AlertTriangle className="w-12 h-12 text-red-400" />
          <p className="text-red-200 font-semibold max-w-md">{loadError}</p>
          <div className="flex gap-3">
            <button
              onClick={() => {
                setLoading(true);
                fetchHold();
              }}
              className="px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-sm font-semibold rounded-lg transition"
            >
              Retry
            </button>
            <button
              onClick={() => navigate("/")}
              className="px-4 py-2 bg-primary hover:bg-primary/95 text-white text-sm font-semibold rounded-lg transition"
            >
              Return to Home
            </button>
          </div>
        </div>
      </CheckoutLayout>
    );
  }

  const isVip = holdDetails?.category.toUpperCase() === "VIP";
  const price = holdDetails?.price || 0;
  const ticketCategoryLabel = isVip ? "VIP Experience" : `${holdDetails?.category ?? ""} Pass`;

  return (
    <CheckoutLayout>
      <div className="grid gap-6 lg:gap-8 lg:grid-cols-3 items-start animate-fade-in">
        {/* Left Side: Form */}
        <div className="lg:col-span-2 space-y-6 order-2 lg:order-1">
          <div className="p-4 sm:p-6 md:p-8 rounded-3xl border border-white/5 glass-premium space-y-6">
            <div className="flex justify-between items-center pb-4 border-b border-white/5">
              <h3 className="text-xl font-bold">Billing Information</h3>
              <div className="flex items-center gap-1.5 text-xs text-green-400 bg-green-500/10 px-2.5 py-1 rounded-full border border-green-500/20 font-medium">
                <ShieldCheck className="w-3.5 h-3.5" /> Secure Checkout
              </div>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit}>
              {validationError && (
                <div className="p-4 rounded-xl border border-red-500/20 bg-red-950/20 text-red-400 text-sm flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 shrink-0" />
                  <span>{validationError}</span>
                </div>
              )}

              {checkoutError && (
                <div className="p-4 rounded-xl border border-red-500/20 bg-red-950/20 text-red-400 text-sm flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 shrink-0" />
                  <span>{checkoutError}</span>
                </div>
              )}

              <div className="space-y-2">
                <label htmlFor="checkout-email" className="block text-sm font-semibold text-neutral-300">Email Address</label>
                <div className="relative">
                  <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-neutral-500" />
                  <input
                    id="checkout-email"
                    type="email"
                    value={email}
                    onChange={(e) => { setEmail(e.target.value); setValidationError(null); setCheckoutError(null); }}
                    disabled={expired || isProcessing}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl pl-11 pr-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="you@example.com"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label htmlFor="checkout-card-name" className="block text-sm font-semibold text-neutral-300">Cardholder Name</label>
                <div className="relative">
                  <User className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-neutral-500" />
                  <input
                    id="checkout-card-name"
                    type="text"
                    value={cardName}
                    onChange={(e) => { setCardName(e.target.value); setValidationError(null); setCheckoutError(null); }}
                    disabled={expired || isProcessing}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl pl-11 pr-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="John Doe"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label htmlFor="checkout-card-number" className="block text-sm font-semibold text-neutral-300">Card Number</label>
                <div className="relative">
                  <CreditCard className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-neutral-500" />
                  <input
                    id="checkout-card-number"
                    type="text"
                    value={cardNumber}
                    onChange={handleCardNumberChange}
                    disabled={expired || isProcessing}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl pl-11 pr-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="4111 1111 1111 1111"
                    required
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label htmlFor="checkout-card-expiry" className="block text-sm font-semibold text-neutral-300">Expiration Date</label>
                  <input
                    id="checkout-card-expiry"
                    type="text"
                    value={cardExpiry}
                    onChange={handleExpiryChange}
                    disabled={expired || isProcessing}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl px-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="MM/YY"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="checkout-card-cvv" className="block text-sm font-semibold text-neutral-300">CVV</label>
                  <input
                    id="checkout-card-cvv"
                    type="password"
                    value={cardCvv}
                    onChange={handleCvvChange}
                    disabled={expired || isProcessing}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl px-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="123"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label htmlFor="checkout-simulate-status" className="block text-sm font-semibold text-neutral-300">Simulation Status (Testing)</label>
                <select
                  id="checkout-simulate-status"
                  value={simulateStatus}
                  onChange={(e) => setSimulateStatus(e.target.value as "success" | "fail")}
                  disabled={expired || isProcessing}
                  className="w-full bg-neutral-950/60 border border-white/5 rounded-xl px-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                >
                  <option value="success">Simulate Success</option>
                  <option value="fail">Simulate Failure (402 Payment Required)</option>
                </select>
              </div>

              <button
                type="submit"
                disabled={expired || cancelling || isProcessing}
                className="w-full py-4 bg-primary hover:bg-primary/95 text-white font-bold rounded-xl transition shadow-lg shadow-primary/20 hover:scale-[1.01] will-change-transform flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
              >
                {isProcessing ? (
                  <>
                    <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    Processing Payment...
                  </>
                ) : (
                  <>
                    <CreditCard className="w-5 h-5" />
                    Pay Now
                  </>
                )}
              </button>

              <button
                type="button"
                onClick={handleCancelReservation}
                disabled={expired || cancelling || isProcessing}
                className="w-full py-3 bg-neutral-900 hover:bg-neutral-800/80 text-neutral-300 font-semibold rounded-xl border border-white/5 transition hover:scale-[1.01] will-change-transform flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
              >
                {cancelling ? (
                  <div className="w-5 h-5 border-2 border-neutral-300 border-t-transparent rounded-full animate-spin" />
                ) : (
                  "Cancel Reservation"
                )}
              </button>
            </form>
          </div>
        </div>

        {/* Right Side: Order Summary & Countdown */}
        <div className="space-y-6 order-1 lg:order-2">
          {/* Order Summary Card */}
          <div className="p-4 sm:p-6 rounded-3xl border border-white/5 glass space-y-4">
            <h3 className="text-lg font-bold">Order Summary</h3>
            <div className="flex justify-between text-neutral-400 text-sm">
              <span>1x {ticketCategoryLabel}</span>
              <span className="font-semibold text-neutral-200">${price.toFixed(2)}</span>
            </div>

            {holdDetails?.ticket_code && (
              <div className="text-xs text-neutral-400 flex justify-between">
                <span>Ticket Code</span>
                <span className="font-mono text-neutral-400">{holdDetails.ticket_code}</span>
              </div>
            )}

            <div className="border-t border-white/5 pt-4 flex justify-between font-bold text-lg">
              <span>Total</span>
              <span className="text-purple-400">${price.toFixed(2)}</span>
            </div>
          </div>

          {/* Countdown Clock */}
          <div className={`p-4 rounded-2xl border text-center transition-all duration-300 ${
            expired
              ? "border-red-500/20 bg-red-950/20 text-red-400"
              : secondsRemaining < 60
              ? "border-amber-500/30 bg-amber-950/20 text-amber-300 animate-pulse"
              : "border-white/5 bg-white/5 text-neutral-300"
          }`}>
            {expired ? (
              <div className="flex items-center justify-center gap-2 text-sm font-semibold">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>Hold Expired</span>
              </div>
            ) : (
              <div className="flex items-center justify-center gap-2 text-sm">
                <Clock className={`w-4 h-4 shrink-0 ${secondsRemaining < 60 ? "text-amber-400" : "text-primary"}`} />
                <span>Hold expires in <span className="font-mono font-extrabold">{formatTime(secondsRemaining)}</span></span>
              </div>
            )}
          </div>
        </div>
      </div>

      {expired && <ExpirationModal onClose={handleReturnHome} />}
    </CheckoutLayout>
  );
};
