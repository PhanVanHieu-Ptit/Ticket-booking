import React, { useState, useEffect } from "react";
import { AdminLayout } from "./admin.layout";
import { useAdminState } from "./admin.state";
import { LogOut, RefreshCw, Lock, ShieldAlert } from "lucide-react";

export const AdminPage: React.FC = () => {
  const {
    token,
    metrics,
    holds,
    setHolds,
    isLoading,
    login,
    logout,
    fetchMetrics,
    fetchHolds,
  } = useAdminState();

  const [passcode, setPasscode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Poll metrics & holds every 5 seconds when authenticated
  useEffect(() => {
    if (!token) return;

    fetchMetrics();
    fetchHolds();

    const interval = setInterval(() => {
      fetchMetrics();
      fetchHolds();
    }, 5000);

    return () => clearInterval(interval);
  }, [token, fetchMetrics, fetchHolds]);

  // Decrement secondsRemaining client-side every second for smooth countdown ticking
  useEffect(() => {
    if (!token || holds.length === 0) return;

    const interval = setInterval(() => {
      setHolds((prevHolds) =>
        prevHolds.map((hold) => ({
          ...hold,
          secondsRemaining: Math.max(0, hold.secondsRemaining - 1),
        }))
      );
    }, 1000);

    return () => clearInterval(interval);
  }, [token, holds.length, setHolds]);

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!passcode.trim()) return;

    setSubmitting(true);
    setAuthError(null);
    try {
      await login(passcode);
    } catch (err: any) {
      setAuthError(err.message || "Failed to sign in. Please verify your passcode.");
    } finally {
      setSubmitting(false);
    }
  };

  const formatDuration = (seconds: number) => {
    if (seconds <= 0) return "00:00";
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  if (!token) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[60vh]">
          <div className="w-full max-w-md p-8 rounded-2xl border border-white/10 bg-zinc-900/60 backdrop-blur-xl space-y-6 shadow-2xl">
            <div className="flex flex-col items-center text-center space-y-2">
              <div className="p-3 rounded-2xl bg-primary/10 border border-primary/20 text-primary">
                <Lock className="w-6 h-6" />
              </div>
              <h2 className="text-2xl font-extrabold tracking-tight text-white mt-3">Admin Access Required</h2>
              <p className="text-sm text-zinc-400">Enter passcode to view sales metrics and monitor holds.</p>
            </div>

            <form onSubmit={handleLoginSubmit} className="space-y-4">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Passcode</label>
                <input
                  type="password"
                  placeholder="••••••••"
                  value={passcode}
                  onChange={(e) => setPasscode(e.target.value)}
                  className="w-full bg-black/40 border border-zinc-800 rounded-xl px-4 py-3 text-white placeholder-zinc-700 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  disabled={submitting}
                />
              </div>

              {authError && (
                <div className="flex items-center gap-2 p-3 rounded-lg border border-red-500/20 bg-red-500/5 text-red-400 text-xs">
                  <ShieldAlert className="w-4 h-4 flex-shrink-0" />
                  <span>{authError}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={submitting || !passcode}
                className="w-full py-3 bg-primary hover:bg-primary/90 disabled:opacity-50 disabled:hover:bg-primary text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/25 active:scale-[0.98]"
              >
                {submitting ? "Verifying..." : "Access Dashboard"}
              </button>
            </form>
          </div>
        </div>
      </AdminLayout>
    );
  }

  const activeHoldsCount = holds.length;

  return (
    <AdminLayout>
      <div className="space-y-8 animate-in fade-in duration-300">
        {/* Header */}
        <section className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-800/80 pb-6">
          <div>
            <h2 className="text-3xl font-extrabold tracking-tight text-white bg-clip-text text-transparent bg-gradient-to-r from-white to-zinc-400">
              Admin Dashboard
            </h2>
            <p className="text-zinc-400 text-sm mt-1">Real-time metrics and active ticket hold monitor.</p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                fetchMetrics();
                fetchHolds();
              }}
              className="flex items-center gap-2 px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-white hover:bg-zinc-805 text-xs font-semibold transition-all"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? "animate-spin" : ""}`} />
              Refresh
            </button>
            <button
              onClick={logout}
              className="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-950/20 border border-red-900/30 text-red-400 hover:text-red-300 hover:bg-red-950/40 text-xs font-semibold transition-all"
            >
              <LogOut className="w-3.5 h-3.5" />
              Sign Out
            </button>
          </div>
        </section>

        {/* Analytics Grid */}
        <section className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 backdrop-blur-md relative overflow-hidden group hover:border-zinc-700/80 transition-all">
            <div className="text-xs font-bold text-zinc-400 uppercase tracking-wider">Total Tickets Sold</div>
            <div className="text-4xl font-extrabold mt-2 text-white">
              {metrics ? metrics.totalTicketsSold : 0}
            </div>
            <div className="text-xs text-zinc-500 mt-2">
              All sold tickets in system
            </div>
          </div>

          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 backdrop-blur-md relative overflow-hidden group hover:border-zinc-700/80 transition-all">
            <div className="text-xs font-bold text-zinc-400 uppercase tracking-wider">Total Revenue</div>
            <div className="text-4xl font-extrabold mt-2 text-emerald-400">
              ${metrics ? metrics.totalRevenue.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : "0.00"}
            </div>
            <div className="text-xs text-zinc-500 mt-2">
              Sum value of all purchases
            </div>
          </div>

          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 backdrop-blur-md relative overflow-hidden group hover:border-zinc-700/80 transition-all">
            <div className="text-xs font-bold text-zinc-400 uppercase tracking-wider">Active Holds</div>
            <div className="text-4xl font-extrabold mt-2 text-amber-400">
              {activeHoldsCount}
            </div>
            <div className="text-xs text-zinc-500 mt-2">
              Holds active in last 5 mins
            </div>
          </div>

          <div className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/30 backdrop-blur-md relative overflow-hidden group hover:border-zinc-700/80 transition-all">
            <div className="text-xs font-bold text-zinc-400 uppercase tracking-wider">Available Inventory</div>
            <div className="text-4xl font-extrabold mt-2 text-blue-400">
              {metrics ? Object.values(metrics.availableInventory).reduce((sum, n) => sum + n, 0) : 0}
            </div>
            <div className="text-xs text-zinc-500 mt-2 flex justify-between gap-2">
              {metrics &&
                Object.entries(metrics.availableInventory).map(([category, count]) => (
                  <span key={category}>{category}: {count}</span>
                ))}
            </div>
          </div>
        </section>

        {/* Detailed Breakdown section */}
        <section className="grid gap-6 md:grid-cols-2">
          {metrics &&
            Object.keys(metrics.availableInventory).map((category) => (
              <div key={category} className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/10 space-y-4">
                <h3 className="text-base font-bold text-white flex items-center justify-between">
                  <span>{category} Category Inventory</span>
                </h3>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-zinc-400">Available</span>
                    <span className="text-white font-semibold">{metrics.availableInventory[category] ?? 0}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-zinc-400">Held (Pending)</span>
                    <span className="text-amber-400 font-semibold">{metrics.heldInventory[category] ?? 0}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-zinc-400">Remaining (Not Sold)</span>
                    <span className="text-blue-400 font-semibold">{metrics.remainingInventory[category] ?? 0}</span>
                  </div>
                </div>
              </div>
            ))}
        </section>

        {/* Holds List Table */}
        <section className="p-6 rounded-2xl border border-zinc-800/80 bg-zinc-900/10 space-y-4">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-bold text-white">Live Reservations Queue</h3>
            <span className="px-2.5 py-0.5 bg-amber-500/10 border border-amber-500/20 text-amber-500 rounded-full text-xs font-medium">
              {activeHoldsCount} Active Holds
            </span>
          </div>

          <div className="overflow-x-auto rounded-xl border border-zinc-800/80 bg-zinc-950/20">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-zinc-800/80 text-zinc-400 text-xs uppercase tracking-wider">
                  <th className="py-4 px-6 font-semibold">Ticket ID</th>
                  <th className="py-4 px-6 font-semibold">Code</th>
                  <th className="py-4 px-6 font-semibold">Category</th>
                  <th className="py-4 px-6 font-semibold">Session ID</th>
                  <th className="py-4 px-6 font-semibold text-right">Time Remaining</th>
                </tr>
              </thead>
              <tbody>
                {holds.length === 0 ? (
                  <tr className="border-b border-zinc-900/40 text-zinc-500 text-sm">
                    <td className="py-8 px-6 text-center" colSpan={5}>
                      No active reservation holds found.
                    </td>
                  </tr>
                ) : (
                  holds.map((hold) => (
                    <tr
                      key={hold.ticketId}
                      className="border-b border-zinc-900/60 text-zinc-300 text-sm hover:bg-white/5 transition-all"
                    >
                      <td className="py-4 px-6 font-mono text-zinc-400">#{hold.ticketId}</td>
                      <td className="py-4 px-6 font-semibold text-white">{hold.ticketCode}</td>
                      <td className="py-4 px-6">
                        <span
                          className={`px-2 py-0.5 rounded-md text-xs font-semibold ${
                            hold.category === "VIP"
                              ? "bg-purple-500/10 border border-purple-500/20 text-purple-400"
                              : "bg-blue-500/10 border border-blue-500/20 text-blue-400"
                          }`}
                        >
                          {hold.category}
                        </span>
                      </td>
                      <td className="py-4 px-6 font-mono text-xs text-zinc-500 truncate max-w-[180px]" title={hold.sessionId}>
                        {hold.sessionId}
                      </td>
                      <td className="py-4 px-6 text-right font-mono font-bold text-amber-500">
                        {formatDuration(hold.secondsRemaining)}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </AdminLayout>
  );
};
