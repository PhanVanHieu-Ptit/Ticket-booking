import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Clock, CreditCard, User, Mail, ShieldCheck, AlertTriangle } from "lucide-react";
import { CheckoutLayout } from "./checkout.layout";
import { bookingApi, ReservationDetails } from "../booking/booking.api";

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

export const CheckoutPage: React.FC = () => {
  const navigate = useNavigate();
  const [holdDetails, setHoldDetails] = useState<ReservationDetails | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number>(300);
  const [loading, setLoading] = useState<boolean>(true);
  const [expired, setExpired] = useState<boolean>(false);
  const [email, setEmail] = useState<string>("");
  const [cardName, setCardName] = useState<string>("");
  const [cancelling, setCancelling] = useState<boolean>(false);

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

  const fetchHold = async () => {
    try {
      const details = await bookingApi.getActiveHold();
      setHoldDetails(details);
      setSecondsRemaining(details.seconds_remaining);
      if (details.seconds_remaining <= 0) {
        setExpired(true);
      } else {
        setExpired(false);
      }
    } catch (err: any) {
      // Redirect to home if there is no active hold found for this session
      navigate("/");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHold();

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
    if (loading || expired || secondsRemaining <= 0) return;

    const timer = setInterval(() => {
      setSecondsRemaining((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          setExpired(true);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [loading, expired, secondsRemaining]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  const handleReturnHome = () => {
    navigate("/");
  };

  if (loading) {
    return (
      <CheckoutLayout>
        <div className="flex flex-col items-center justify-center py-20 space-y-4">
          <div className="w-12 h-12 border-4 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="text-neutral-400 font-medium animate-pulse">Loading secure checkout...</p>
        </div>
      </CheckoutLayout>
    );
  }

  const isVip = holdDetails?.category.toUpperCase() === "VIP";
  const price = holdDetails?.price || 0;
  const ticketCategoryLabel = isVip ? "VIP Experience" : "Standard Pass";

  return (
    <CheckoutLayout>
      <div className="grid gap-8 lg:grid-cols-3 items-start animate-fade-in">
        {/* Left Side: Form */}
        <div className="lg:col-span-2 space-y-6">
          <div className="p-6 md:p-8 rounded-3xl border border-white/5 glass-premium space-y-6">
            <div className="flex justify-between items-center pb-4 border-b border-white/5">
              <h3 className="text-xl font-bold">Billing Information</h3>
              <div className="flex items-center gap-1.5 text-xs text-green-400 bg-green-500/10 px-2.5 py-1 rounded-full border border-green-500/20 font-medium">
                <ShieldCheck className="w-3.5 h-3.5" /> Secure Checkout
              </div>
            </div>

            <form className="space-y-5" onSubmit={(e) => e.preventDefault()}>
              <div className="space-y-2">
                <label className="block text-sm font-semibold text-neutral-300">Email Address</label>
                <div className="relative">
                  <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-neutral-500" />
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    disabled={expired}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl pl-11 pr-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="you@example.com"
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-semibold text-neutral-300">Cardholder Name</label>
                <div className="relative">
                  <User className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-neutral-500" />
                  <input
                    type="text"
                    value={cardName}
                    onChange={(e) => setCardName(e.target.value)}
                    disabled={expired}
                    className="w-full bg-neutral-950/60 border border-white/5 rounded-xl pl-11 pr-4 py-3 text-neutral-200 focus:outline-none focus:border-primary transition disabled:opacity-50 disabled:bg-neutral-900/40"
                    placeholder="John Doe"
                    required
                  />
                </div>
              </div>

              <button 
                type="submit"
                disabled={expired || cancelling}
                className="w-full py-4 bg-primary hover:bg-primary/95 text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/20 hover:scale-[1.01] flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
              >
                <CreditCard className="w-5 h-5" />
                Pay Now
              </button>

              <button
                type="button"
                onClick={handleCancelReservation}
                disabled={expired || cancelling}
                className="w-full py-3 bg-neutral-900 hover:bg-neutral-800/80 text-neutral-300 font-semibold rounded-xl border border-white/5 transition-all hover:scale-[1.01] flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
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
        <div className="space-y-6">
          {/* Order Summary Card */}
          <div className="p-6 rounded-3xl border border-white/5 glass space-y-4">
            <h3 className="text-lg font-bold">Order Summary</h3>
            <div className="flex justify-between text-neutral-400 text-sm">
              <span>1x {ticketCategoryLabel}</span>
              <span className="font-semibold text-neutral-200">${price.toFixed(2)}</span>
            </div>
            
            {holdDetails?.ticket_code && (
              <div className="text-xs text-neutral-500 flex justify-between">
                <span>Ticket Code</span>
                <span className="font-mono text-neutral-400">{holdDetails.ticket_code}</span>
              </div>
            )}
            
            <div className="border-t border-white/5 pt-4 flex justify-between font-bold text-lg">
              <span>Total</span>
              <span className="text-primary">${price.toFixed(2)}</span>
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
