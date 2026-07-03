import React, { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { CheckCircle2, Home, Ticket, DollarSign, Calendar, ShieldCheck } from "lucide-react";
import { CheckoutLayout } from "./checkout.layout";

export const ConfirmationPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // Retrieve checkout results passed via router navigation state
  const orderResult = location.state?.orderResult;
  const holdDetails = location.state?.holdDetails;

  useEffect(() => {
    // If no order result exists in navigation state, prevent direct access
    if (!orderResult) {
      navigate("/", { replace: true });
    }
  }, [orderResult, navigate]);

  if (!orderResult) {
    return null;
  }

  const isVip = holdDetails?.category?.toUpperCase() === "VIP";
  const categoryLabel = isVip ? "VIP Experience" : `${holdDetails?.category ?? ""} Pass`;
  const paidAtDate = new Date(orderResult.paidAt).toLocaleString();

  return (
    <CheckoutLayout>
      <div className="max-w-xl mx-auto space-y-8 py-8 animate-fade-in">
        {/* Success Banner */}
        <div className="text-center space-y-4">
          <div className="mx-auto w-20 h-20 bg-green-500/10 border border-green-500/30 rounded-3xl flex items-center justify-center shadow-2xl shadow-green-950/20">
            <CheckCircle2 className="w-10 h-10 text-green-400 animate-scale-in" />
          </div>
          <div className="space-y-2">
            <span className="text-xs font-semibold tracking-wider text-green-400 uppercase bg-green-500/10 px-3.5 py-1.5 rounded-full border border-green-500/20">
              Payment Successful
            </span>
            <h2 className="text-3xl font-extrabold tracking-tight text-white pt-2">
              Your Ticket is Secured!
            </h2>
            <p className="text-neutral-400 text-sm leading-relaxed max-w-sm mx-auto">
              A confirmation email has been sent. Your digital ticket is ready for the event.
            </p>
          </div>
        </div>

        {/* Order Receipt Details */}
        <div className="p-4 sm:p-6 md:p-8 rounded-3xl border border-white/5 glass-premium space-y-6 shadow-2xl">
          <div className="flex justify-between items-center pb-4 border-b border-white/5">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <ShieldCheck className="w-5 h-5 text-primary" /> Order Receipt
            </h3>
            <span className="text-xs text-neutral-400 font-mono bg-neutral-950/60 px-2.5 py-1 rounded-md border border-white/5">
              Ref: {orderResult.paymentReference}
            </span>
          </div>

          <div className="space-y-4 text-sm">
            {/* Ticket Details */}
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-neutral-950/60 rounded-xl border border-white/5">
                  <Ticket className="w-4 h-4 text-neutral-400" />
                </div>
                <div>
                  <p className="text-xs text-neutral-400">Ticket Category</p>
                  <p className="font-semibold text-white">{categoryLabel}</p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-xs text-neutral-400">Amount Paid</p>
                <p className="font-bold text-purple-400 flex items-center justify-end">
                  <DollarSign className="w-3.5 h-3.5" />
                  {orderResult.amount.toFixed(2)}
                </p>
              </div>
            </div>

            {/* Ticket Code */}
            {holdDetails?.ticket_code && (
              <div className="flex items-center justify-between p-3.5 bg-neutral-950/40 rounded-2xl border border-white/5">
                <div>
                  <p className="text-xs text-neutral-400">Ticket Reference Code</p>
                  <p className="font-mono text-base font-bold text-white tracking-wide mt-0.5">
                    {holdDetails.ticket_code}
                  </p>
                </div>
                <div className="px-3 py-1 bg-white/5 rounded-lg border border-white/5 text-[10px] uppercase font-bold text-neutral-300">
                  Active
                </div>
              </div>
            )}

            {/* Date Details */}
            <div className="flex items-center gap-3 pt-2">
              <div className="p-2.5 bg-neutral-950/60 rounded-xl border border-white/5">
                <Calendar className="w-4 h-4 text-neutral-400" />
              </div>
              <div>
                <p className="text-xs text-neutral-400">Date & Time</p>
                <p className="font-semibold text-white">{paidAtDate}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Return Button */}
        <div className="flex flex-col sm:flex-row gap-4 items-center justify-center">
          <button
            onClick={() => navigate("/")}
            className="w-full sm:w-auto px-8 py-3.5 bg-primary hover:bg-primary/95 text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/20 hover:scale-[1.02] flex items-center justify-center gap-2"
          >
            <Home className="w-4 h-4" />
            Return to Home Page
          </button>
        </div>
      </div>
    </CheckoutLayout>
  );
};
