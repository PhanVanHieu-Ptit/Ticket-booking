import { useState } from 'react';
import { Ticket, ShieldAlert, BarChart3, Clock, Flame, CreditCard, Layers, AlertTriangle } from 'lucide-react';
import { useTicketAvailability } from './hooks/useTicketAvailability';
import { TicketCategoryCard } from './components/TicketCategoryCard';

export default function App() {
  const [activeTab, setActiveTab] = useState<'home' | 'admin'>('home');
  const { categories, loading, error, isConnected } = useTicketAvailability();

  const totalAvailable = categories.reduce((sum, cat) => sum + cat.available, 0);
  const isEventSoldOut = !loading && categories.length > 0 && totalAvailable === 0;

  const handleReserve = (categoryName: string) => {
    console.log(`Reserving ticket for ${categoryName}`);
  };

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col selection:bg-primary selection:text-white">
      {/* Navigation Header */}
      <header className="sticky top-0 z-50 glass border-b border-white/5 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/20 rounded-xl border border-primary/30 glow-active">
            <Flame className="w-6 h-6 text-primary" />
          </div>
          <div>
            <span className="text-xl font-extrabold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-white via-neutral-200 to-primary">
              TICKET RUSH
            </span>
            <span className="ml-2 text-xs font-semibold px-2 py-0.5 bg-white/10 rounded-full text-neutral-300">
              v1.0.0-foundation
            </span>
          </div>
        </div>

        <nav className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab('home')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'home'
                ? 'bg-primary text-white shadow-lg shadow-primary/20'
                : 'text-neutral-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <Ticket className="w-4 h-4" />
            Concert & Booking
          </button>
          <button
            onClick={() => setActiveTab('admin')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'admin'
                ? 'bg-primary text-white shadow-lg shadow-primary/20'
                : 'text-neutral-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <BarChart3 className="w-4 h-4" />
            Admin Dashboard
          </button>
        </nav>
      </header>

      {/* Main Content Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-6 md:p-8">
        {activeTab === 'home' ? (
          <div className="space-y-8 animate-fade-in">
            {isEventSoldOut && (
              <div className="p-4 rounded-xl bg-red-950/30 border border-red-500/30 text-red-200 text-center font-bold flex items-center justify-center gap-2 animate-pulse shadow-lg shadow-red-950/20">
                <AlertTriangle className="w-5 h-5 text-red-400 animate-bounce" />
                <span>ALL TICKETS SOLD OUT: Neon Symphony 2026 is fully booked!</span>
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
                      : 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20 animate-pulse'
                  }`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-green-400' : 'bg-yellow-400'}`} />
                    {isConnected ? 'Syncing Live' : 'Reconnecting...'}
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
        ) : (
          <div className="space-y-8 animate-fade-in">
            {/* Dashboard Stats */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
              <div className="glass p-6 rounded-2xl border border-white/5 flex items-center justify-between">
                <div>
                  <span className="text-xs font-bold text-neutral-400 uppercase tracking-wider">
                    Total Sales
                  </span>
                  <p className="text-3xl font-extrabold mt-1">
                    0 <span className="text-sm font-semibold text-neutral-500">/ 500</span>
                  </p>
                </div>
                <div className="p-3 bg-primary/10 rounded-xl border border-primary/20">
                  <Ticket className="w-6 h-6 text-primary" />
                </div>
              </div>

              <div className="glass p-6 rounded-2xl border border-white/5 flex items-center justify-between">
                <div>
                  <span className="text-xs font-bold text-neutral-400 uppercase tracking-wider">
                    Total Revenue
                  </span>
                  <p className="text-3xl font-extrabold mt-1">$0.00</p>
                </div>
                <div className="p-3 bg-green-500/10 rounded-xl border border-green-500/20">
                  <CreditCard className="w-6 h-6 text-green-400" />
                </div>
              </div>

              <div className="glass p-6 rounded-2xl border border-white/5 flex items-center justify-between">
                <div>
                  <span className="text-xs font-bold text-neutral-400 uppercase tracking-wider">
                    Active Holds
                  </span>
                  <p className="text-3xl font-extrabold mt-1">0</p>
                </div>
                <div className="p-3 bg-yellow-500/10 rounded-xl border border-yellow-500/20">
                  <Clock className="w-6 h-6 text-yellow-400" />
                </div>
              </div>

              <div className="glass p-6 rounded-2xl border border-white/5 flex items-center justify-between">
                <div>
                  <span className="text-xs font-bold text-neutral-400 uppercase tracking-wider">
                    System Load
                  </span>
                  <p className="text-3xl font-extrabold mt-1 text-green-400">Idle</p>
                </div>
                <div className="p-3 bg-purple-500/10 rounded-xl border border-purple-500/20">
                  <ShieldAlert className="w-6 h-6 text-purple-400" />
                </div>
              </div>
            </div>

            {/* Active Holds Table Mockup */}
            <div className="glass rounded-2xl border border-white/5 overflow-hidden">
              <div className="px-6 py-5 border-b border-white/5 flex justify-between items-center">
                <div>
                  <h3 className="text-lg font-bold">Active Ticket Holds</h3>
                  <p className="text-xs text-neutral-400 mt-0.5">
                    Tickets currently locked in Redis memory (5-minute TTL)
                  </p>
                </div>
                <span className="text-xs font-semibold px-2.5 py-1 bg-white/5 rounded-full border border-white/5 text-neutral-300">
                  Real-time updates
                </span>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-white/5 bg-white/[0.02] text-xs font-bold text-neutral-400 uppercase tracking-wider">
                      <th className="px-6 py-4">Ticket ID</th>
                      <th className="px-6 py-4">Category</th>
                      <th className="px-6 py-4">Session ID</th>
                      <th className="px-6 py-4">Held At</th>
                      <th className="px-6 py-4">Time Remaining</th>
                      <th className="px-6 py-4">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-white/5 text-sm text-neutral-300">
                    <tr>
                      <td
                        className="px-6 py-8 text-center text-neutral-500 font-medium"
                        colSpan={6}
                      >
                        No active ticket holds at this time.
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="glass border-t border-white/5 py-6 px-8 text-center text-xs text-neutral-500 flex flex-col sm:flex-row gap-4 items-center justify-between">
        <p>© 2026 Ticket Rush Inc. All rights reserved.</p>
        <div className="flex gap-4">
          <a href="#" className="hover:text-neutral-300 transition-colors">
            API Docs
          </a>
          <a href="#" className="hover:text-neutral-300 transition-colors">
            System Status
          </a>
          <a href="#" className="hover:text-neutral-300 transition-colors">
            Terms of Service
          </a>
        </div>
      </footer>
    </div>
  );
}
