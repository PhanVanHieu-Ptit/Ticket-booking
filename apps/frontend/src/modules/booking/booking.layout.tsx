import React from "react";

export interface BookingLayoutProps {
  children?: React.ReactNode;
}

export const BookingLayout: React.FC<BookingLayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-50 font-sans">
      <header className="border-b border-zinc-800 bg-zinc-900/50 backdrop-blur sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-blue-400 to-indigo-500 bg-clip-text text-transparent">
            Concert Tickets
          </h1>
          <div className="text-sm text-zinc-400">Live Booking Session</div>
        </div>
      </header>
      <main className="max-w-6xl mx-auto px-4 py-8">{children}</main>
    </div>
  );
};
