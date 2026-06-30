import { useState } from 'react';
import { Ticket, ShieldAlert, BarChart3, Clock, Flame, CreditCard, Layers } from 'lucide-react';

export default function App() {
  const [activeTab, setActiveTab] = useState<'home' | 'admin'>('home');

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
            {/* Hero Section */}
            <div className="relative overflow-hidden rounded-3xl glass-premium p-8 md:p-12 flex flex-col md:flex-row gap-8 items-center justify-between">
              <div className="absolute top-0 right-0 w-96 h-96 bg-primary/10 rounded-full blur-3xl -z-10" />
              <div className="space-y-4 max-w-xl text-center md:text-left">
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-primary/20 border border-primary/30 text-primary uppercase tracking-wider">
                  <Flame className="w-3.5 h-3.5" /> Live Concert Event
                </span>
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

            {/* Ticket Categories (Mockup Grid) */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* VIP Category */}
              <div className="relative overflow-hidden rounded-2xl glass p-6 border border-primary/20 hover:border-primary/40 transition-all group flex flex-col justify-between space-y-6">
                <div className="absolute top-0 right-0 w-24 h-24 bg-primary/5 rounded-full blur-2xl" />
                <div className="space-y-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-bold text-primary uppercase tracking-widest px-2 py-0.5 bg-primary/10 rounded border border-primary/20">
                        Premium
                      </span>
                      <h3 className="text-2xl font-bold mt-2">VIP Experience</h3>
                    </div>
                    <span className="text-3xl font-extrabold text-primary">$150</span>
                  </div>
                  <p className="text-sm text-neutral-400">
                    Front row access, private lounge entry, complimentary merchandise, and
                    meet-and-greet options.
                  </p>
                </div>

                <div className="space-y-4">
                  <div>
                    <div className="flex justify-between text-xs font-semibold mb-1 text-neutral-300">
                      <span>Inventory Remaining</span>
                      <span>100 / 100 Available</span>
                    </div>
                    <div className="h-2 w-full bg-white/5 rounded-full overflow-hidden">
                      <div className="h-full bg-gradient-to-r from-primary to-purple-500 rounded-full w-full" />
                    </div>
                  </div>
                  <button className="w-full py-3 px-4 bg-primary hover:bg-primary/90 text-white font-bold rounded-xl transition-all shadow-lg shadow-primary/20 flex items-center justify-center gap-2 group-hover:scale-[1.02]">
                    <Ticket className="w-5 h-5" />
                    Reserve VIP Ticket
                  </button>
                </div>
              </div>

              {/* Standard Category */}
              <div className="relative overflow-hidden rounded-2xl glass p-6 border border-white/5 hover:border-white/10 transition-all group flex flex-col justify-between space-y-6">
                <div className="space-y-3">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="text-xs font-bold text-neutral-400 uppercase tracking-widest px-2 py-0.5 bg-white/5 rounded border border-white/5">
                        General Admission
                      </span>
                      <h3 className="text-2xl font-bold mt-2">Standard Pass</h3>
                    </div>
                    <span className="text-3xl font-extrabold text-neutral-200">$75</span>
                  </div>
                  <p className="text-sm text-neutral-400">
                    Access to main standing arena, standard food and beverage stalls, and
                    high-fidelity sound zones.
                  </p>
                </div>

                <div className="space-y-4">
                  <div>
                    <div className="flex justify-between text-xs font-semibold mb-1 text-neutral-300">
                      <span>Inventory Remaining</span>
                      <span>400 / 400 Available</span>
                    </div>
                    <div className="h-2 w-full bg-white/5 rounded-full overflow-hidden">
                      <div className="h-full bg-neutral-600 rounded-full w-full" />
                    </div>
                  </div>
                  <button className="w-full py-3 px-4 bg-white/10 hover:bg-white/15 text-white font-bold rounded-xl transition-all flex items-center justify-center gap-2 group-hover:scale-[1.02]">
                    <Ticket className="w-5 h-5" />
                    Reserve Standard Ticket
                  </button>
                </div>
              </div>
            </div>

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
