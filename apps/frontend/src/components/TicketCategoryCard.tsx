import React from "react";
import { Ticket, Loader2 } from "lucide-react";
import { TicketCategoryAvailability } from "../modules/booking/booking.api";

interface TicketCategoryCardProps {
  category: TicketCategoryAvailability;
  onReserve: (categoryName: string) => void;
  isReserving?: boolean;
}

export const TicketCategoryCard: React.FC<TicketCategoryCardProps> = ({
  category,
  onReserve,
  isReserving = false,
}) => {
  const { name, price, available, total, status } = category;
  const isSoldOut = available <= 0 || status === "Sold Out";
  const percentage = total > 0 ? (available / total) * 100 : 0;

  // Premium themes based on VIP vs Standard
  const isVip = name.toUpperCase() === "VIP";
  const cardBorderClass = isVip
    ? "border-primary/20 hover:border-primary/40"
    : "border-white/5 hover:border-white/10";
  const priceColorClass = isVip ? "text-primary" : "text-neutral-200";
  const badgeClass = isVip
    ? "text-primary bg-primary/10 border-primary/20"
    : "text-neutral-400 bg-white/5 border-white/5";
  const progressColorClass = isVip
    ? "bg-gradient-to-r from-primary to-purple-500"
    : "bg-neutral-600";
  
  const isDisabled = isSoldOut || isReserving;
  const buttonClass = isDisabled
    ? "bg-neutral-800 text-neutral-500 cursor-not-allowed border border-white/5"
    : isVip
    ? "bg-primary hover:bg-primary/90 text-white shadow-lg shadow-primary/20 hover:scale-[1.02]"
    : "bg-white/10 hover:bg-white/15 text-white hover:scale-[1.02]";

  return (
    <div
      className={`relative overflow-hidden rounded-2xl glass p-6 border transition-all group flex flex-col justify-between space-y-6 ${cardBorderClass}`}
    >
      {isVip && (
        <div className="absolute top-0 right-0 w-24 h-24 bg-primary/5 rounded-full blur-2xl pointer-events-none" />
      )}
      <div className="space-y-3">
        <div className="flex justify-between items-start">
          <div>
            <span
              className={`text-xs font-bold uppercase tracking-widest px-2 py-0.5 rounded border ${badgeClass}`}
            >
              {isVip ? "Premium" : "General Admission"}
            </span>
            <h3 className="text-2xl font-bold mt-2">
              {isVip ? "VIP Experience" : "Standard Pass"}
            </h3>
          </div>
          <span className={`text-3xl font-extrabold ${priceColorClass}`}>
            ${price}
          </span>
        </div>
        <p className="text-sm text-neutral-400">
          {isVip
            ? "Front row access, private lounge entry, complimentary merchandise, and meet-and-greet options."
            : "Access to main standing arena, standard food and beverage stalls, and high-fidelity sound zones."}
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <div className="flex justify-between text-xs font-semibold mb-1 text-neutral-300">
            <span>Inventory Remaining</span>
            <span>
              {isSoldOut ? "0" : available} / {total} Available
            </span>
          </div>
          <div className="h-2 w-full bg-white/5 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-500 ${progressColorClass}`}
              style={{ width: `${percentage}%` }}
            />
          </div>
        </div>
        <button
          onClick={() => !isDisabled && onReserve(name)}
          disabled={isDisabled}
          className={`w-full py-3 px-4 font-bold rounded-xl transition-all flex items-center justify-center gap-2 ${buttonClass}`}
        >
          {isReserving ? (
            <>
              <Loader2 className="w-5 h-5 animate-spin" />
              <span>Reserving...</span>
            </>
          ) : (
            <>
              <Ticket className="w-5 h-5" />
              <span>{isSoldOut ? "Sold Out" : `Reserve ${name} Ticket`}</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
};
