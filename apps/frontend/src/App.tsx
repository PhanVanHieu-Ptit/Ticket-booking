import { Ticket, BarChart3, Flame } from 'lucide-react';
import { createBrowserRouter, RouterProvider, Outlet, useNavigate, useLocation } from 'react-router-dom';

function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  
  // Decide which tab is active based on path
  const activeTab = location.pathname.startsWith('/admin') ? 'admin' : 'home';

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col selection:bg-primary selection:text-white">
      {/* Navigation Header */}
      <header className="sticky top-0 z-50 glass border-b border-white/5 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3 cursor-pointer" onClick={() => navigate('/')}>
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
            onClick={() => navigate('/')}
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
            onClick={() => navigate('/admin')}
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
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="glass border-t border-white/5 py-6 px-8 text-center text-xs text-neutral-400 flex flex-col sm:flex-row gap-4 items-center justify-between">
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

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        path: '',
        lazy: async () => {
          const { BookingPage } = await import('./modules/booking/booking.page');
          return { Component: BookingPage };
        },
      },
      {
        path: 'checkout',
        lazy: async () => {
          const { CheckoutPage } = await import('./modules/checkout/checkout.page');
          return { Component: CheckoutPage };
        },
      },
      {
        path: 'confirmation',
        lazy: async () => {
          const { ConfirmationPage } = await import('./modules/checkout/confirmation.page');
          return { Component: ConfirmationPage };
        },
      },
      {
        path: 'admin',
        lazy: async () => {
          const { AdminPage } = await import('./modules/admin/admin.page');
          return { Component: AdminPage };
        },
      },
    ],
  },
]);

export default function App() {
  return <RouterProvider router={router} />;
}
