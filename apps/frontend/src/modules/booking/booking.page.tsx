import React from "react";
import { BookingLayout } from "./booking.layout";

export const BookingPage: React.FC = () => {
  // Placeholder page state and elements
  return (
    <BookingLayout>
      <div className="max-w-2xl mx-auto space-y-8">
        <section className="text-center space-y-4">
          <h2 className="text-4xl font-extrabold tracking-tight">Reserve Your Tickets</h2>
          <p className="text-zinc-400 text-lg">
            Experience the concert of the year. Select your ticket category below.
          </p>
        </section>

        <section className="grid gap-6 md:grid-cols-2">
          {/* Ticket categories list placeholders */}
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/50 space-y-4">
            <h3 className="text-2xl font-bold">VIP Ticket</h3>
            <p className="text-zinc-400">Premium seating, exclusive lounge access, and merchandise pack.</p>
            <div className="flex justify-between items-center pt-4">
              <span className="text-2xl font-black text-blue-400">$100.00</span>
              <button className="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-6 rounded-lg transition">
                Reserve VIP
              </button>
            </div>
          </div>

          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/50 space-y-4">
            <h3 className="text-2xl font-bold">Standard Ticket</h3>
            <p className="text-zinc-400">General admission entry with great views of the main stage.</p>
            <div className="flex justify-between items-center pt-4">
              <span className="text-2xl font-black text-blue-400">$50.00</span>
              <button className="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-6 rounded-lg transition">
                Reserve Standard
              </button>
            </div>
          </div>
        </section>
      </div>
    </BookingLayout>
  );
};
