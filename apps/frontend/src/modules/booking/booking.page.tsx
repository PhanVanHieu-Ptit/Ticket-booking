import React, { useState, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Flame, Clock, Layers, ShieldAlert, AlertTriangle } from "lucide-react";
import { useTicketAvailability } from "../../hooks/useTicketAvailability";
import { TicketCategoryCard } from "../../components/TicketCategoryCard";
import { bookingApi } from "./booking.api";

export const BookingPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { categories, loading, error, isConnected, isDegraded } = useTicketAvailability();
  
  const [reservingCategory, setReservingCategory] = useState<string | null>(null);
  const [reserveError, setReserveError] = useState<string | null>(null);
  const [cancellationMessage, setCancellationMessage] = useState<string | null>(
    location.state?.message || null
  );

  useEffect(() => {
    if (location.state?.message) {
      navigate(location.pathname, { replace: true, state: {} });
    }
  }, [location, navigate]);

  const totalAvailable = categories.reduce((sum, cat) => sum + cat.available, 0);
  const isEventSoldOut = !loading && categories.length > 0 && totalAvailable === 0;

  const handleReserve = async (categoryName: string) => {
    setReservingCategory(categoryName);
    setReserveError(null);
    try {
      await bookingApi.reserveTicket(categoryName);
      navigate("/checkout");
    } catch (err: any) {
      // If the error indicates an active hold already exists, redirect directly to checkout page
      if (err.code === "ACTIVE_HOLD_EXISTS" || (err.message && err.message.includes("active reservation"))) {
        navigate("/checkout");
      } else if (err.code === "TICKET_SOLD_OUT" || err.code === "TICKET_UNAVAILABLE") {
        setReserveError("Sorry, this ticket just sold out. Please pick another category.");
      } else if (err.code === "TIMEOUT") {
        setReserveError("Slow connection, please try again.");
      } else {
        setReserveError(err.message || "Failed to reserve ticket. Please try again.");
      }
    } finally {
      setReservingCategory(null);
    }
  };

  return (
    <div className="space-y-8 animate-fade-in">
      {isEventSoldOut && (
        <div className="p-4 rounded-xl bg-red-950/30 border border-red-500/30 text-red-200 text-center font-bold flex items-center justify-center gap-2 animate-pulse shadow-lg shadow-red-950/20">
          <AlertTriangle className="w-5 h-5 text-red-400 animate-bounce" />
          <span>ALL TICKETS SOLD OUT: Neon Symphony 2026 is fully booked!</span>
        </div>
      )}
      
      {reserveError && (
        <div className="p-4 rounded-xl bg-red-950/30 border border-red-500/30 text-red-200 text-center font-semibold flex items-center justify-center gap-2">
          <AlertTriangle className="w-5 h-5 text-red-400" />
          <span>{reserveError}</span>
        </div>
      )}

      {cancellationMessage && (
        <div className="p-4 rounded-xl bg-purple-950/30 border border-purple-500/30 text-purple-200 text-center font-semibold flex items-center justify-between gap-2 animate-fade-in shadow-lg shadow-purple-950/20">
          <div className="flex items-center gap-2 mx-auto">
            <ShieldAlert className="w-5 h-5 text-primary" />
            <span>{cancellationMessage}</span>
          </div>
          <button 
            onClick={() => setCancellationMessage(null)}
            className="text-neutral-400 hover:text-white transition-colors text-lg font-bold px-1"
          >
            ×
          </button>
        </div>
      )}

      {/* Hero Section */}
      <div className="relative overflow-hidden rounded-3xl glass-premium p-8 md:p-12 flex flex-col md:flex-row gap-8 items-center justify-between">
        <div className="absolute top-0 right-0 w-96 h-96 bg-primary/10 rounded-full blur-3xl -z-10" />
        <div className="space-y-4 max-w-xl text-center md:text-left">
          <div className="flex flex-wrap items-center gap-2 justify-center md:justify-start">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-primary/20 border border-primary/30 text-primary uppercase tracking-wider">
              <Flame className="w-3.5 h-3.5" /> Live Concert Event
            </span>
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border ${
              isConnected
                ? 'bg-green-500/10 text-green-400 border-green-500/20'
                : isDegraded
                  ? 'bg-orange-500/10 text-orange-400 border-orange-500/20 animate-pulse'
                  : 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20 animate-pulse'
            }`}>
              <span className={`w-1.5 h-1.5 rounded-full ${
                isConnected ? 'bg-green-400' : isDegraded ? 'bg-orange-400' : 'bg-yellow-400'
              }`} />
              {isConnected ? 'Syncing Live' : isDegraded ? 'Slow Mode · Polling' : 'Reconnecting...'}
            </span>
          </div>
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight leading-tight">
            Neon Symphony: <br />
            <span className="bg-clip-text text-transparent bg-gradient-to-r from-primary to-purple-400">
              Hyperion Tour 2026
            </span>
          </h1>
          <p className="text-neutral-400 text-base md:text-lg">
            Experience the ultimate high-energy electronic music festival. 5,000 concurrent
            fans competing for only 500 exclusive seats. Speed is everything.
          </p>
          <div className="flex flex-wrap gap-4 pt-2 justify-center md:justify-start">
            <div className="flex items-center gap-2 text-sm text-neutral-300 bg-white/5 px-3.5 py-1.5 rounded-full border border-white/5">
              <Clock className="w-4 h-4 text-primary" />
              <span>June 30, 2026 • 20:00 UTC</span>
            </div>
            <div className="flex items-center gap-2 text-sm text-neutral-300 bg-white/5 px-3.5 py-1.5 rounded-full border border-white/5">
              <Layers className="w-4 h-4 text-purple-400" />
              <span>500 Total Tickets</span>
            </div>
          </div>
        </div>

        {/* Countdown / Timer Card Mockup */}
        <div className="w-full md:w-auto min-w-[280px] glass p-6 rounded-2xl border border-white/10 flex flex-col items-center justify-center space-y-4 text-center">
          <span className="text-xs font-bold text-neutral-400 uppercase tracking-wider">
            Ticket Sale Begins In
          </span>
          <div className="flex gap-3">
            <div className="flex flex-col">
              <span className="text-3xl font-extrabold font-mono bg-white/5 px-3 py-2 rounded-lg border border-white/5">
                00
              </span>
              <span className="text-[10px] font-semibold text-neutral-500 mt-1">HOURS</span>
            </div>
            <span className="text-2xl font-bold self-center text-primary">:</span>
            <div className="flex flex-col">
              <span className="text-3xl font-extrabold font-mono bg-white/5 px-3 py-2 rounded-lg border border-white/5">
                04
              </span>
              <span className="text-[10px] font-semibold text-neutral-500 mt-1">MINUTES</span>
            </div>
            <span className="text-2xl font-bold self-center text-primary">:</span>
            <div className="flex flex-col">
              <span className="text-3xl font-extrabold font-mono bg-white/5 px-3 py-2 rounded-lg border border-white/5">
                59
              </span>
              <span className="text-[10px] font-semibold text-neutral-500 mt-1">SECONDS</span>
            </div>
          </div>
          <div className="w-full pt-2">
            <div className="h-1.5 w-full bg-white/10 rounded-full overflow-hidden">
              <div className="h-full bg-primary rounded-full w-[98%] animate-pulse" />
            </div>
          </div>
        </div>
      </div>

      {/* Ticket Categories (Dynamic Grid) */}
      {loading && categories.length === 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {[1, 2].map((i) => (
            <div key={i} className="h-72 rounded-2xl glass border border-white/5 animate-pulse flex flex-col justify-between p-6">
              <div className="space-y-4">
                <div className="h-6 w-24 bg-white/10 rounded" />
                <div className="h-8 w-48 bg-white/10 rounded" />
                <div className="h-16 w-full bg-white/10 rounded" />
              </div>
              <div className="space-y-4">
                <div className="h-4 w-32 bg-white/10 rounded" />
                <div className="h-2 w-full bg-white/10 rounded" />
                <div className="h-12 w-full bg-white/10 rounded" />
              </div>
            </div>
          ))}
        </div>
      ) : error && categories.length === 0 ? (
        <div className="p-8 text-center rounded-2xl border border-red-500/20 bg-red-950/10 text-red-400">
          <AlertTriangle className="w-12 h-12 mx-auto mb-4 text-red-500 animate-bounce" />
          <h4 className="font-bold text-lg">Error loading availability</h4>
          <p className="text-sm text-neutral-400 mt-2">{error}</p>
          <button 
            onClick={() => window.location.reload()}
            className="mt-4 px-4 py-2 bg-white/10 hover:bg-white/15 text-white text-xs font-semibold rounded-lg transition"
          >
            Retry Connection
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {categories.map((category) => (
            <TicketCategoryCard
              key={category.name}
              category={category}
              onReserve={handleReserve}
              isReserving={reservingCategory === category.name}
            />
          ))}
        </div>
      )}

      {/* Note about concurrency shield */}
      <div className="flex items-start gap-4 p-4 rounded-xl bg-purple-950/20 border border-purple-500/20 max-w-3xl mx-auto">
        <ShieldAlert className="w-5 h-5 text-primary shrink-0 mt-0.5" />
        <div className="text-xs text-neutral-400 leading-relaxed">
          <span className="font-semibold text-neutral-200 block mb-0.5">
            High-Concurrency Shield Active
          </span>
          This app uses an in-memory Redis lock. When you click "Reserve", a temporary
          5-minute hold is acquired. If you do not complete payment before the timer expires,
          the ticket is immediately recycled back into the public pool.
        </div>
      </div>
    </div>
  );
};
