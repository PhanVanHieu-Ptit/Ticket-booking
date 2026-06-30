import React from "react";
import { CheckoutLayout } from "./checkout.layout";

export const CheckoutPage: React.FC = () => {
  return (
    <CheckoutLayout>
      <div className="grid gap-8 md:grid-cols-3">
        <div className="md:col-span-2 space-y-6">
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/30 space-y-4">
            <h3 className="text-xl font-bold">Billing Information</h3>
            <form className="space-y-4" onSubmit={(e) => e.preventDefault()}>
              <div>
                <label className="block text-sm text-zinc-400 mb-1">Email Address</label>
                <input
                  type="email"
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 text-zinc-200 focus:outline-none focus:border-blue-500"
                  placeholder="you@example.com"
                />
              </div>
              <div>
                <label className="block text-sm text-zinc-400 mb-1">Cardholder Name</label>
                <input
                  type="text"
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 text-zinc-200 focus:outline-none focus:border-blue-500"
                  placeholder="John Doe"
                />
              </div>
              <button className="w-full bg-emerald-600 hover:bg-emerald-700 text-white font-semibold py-3 rounded-lg transition">
                Pay Now
              </button>
            </form>
          </div>
        </div>

        <div className="space-y-6">
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/50 space-y-4">
            <h3 className="text-lg font-bold">Order Summary</h3>
            <div className="flex justify-between text-zinc-400 text-sm">
              <span>1x VIP Ticket</span>
              <span>$100.00</span>
            </div>
            <div className="border-t border-zinc-800 pt-4 flex justify-between font-bold">
              <span>Total</span>
              <span className="text-blue-400">$100.00</span>
            </div>
          </div>

          <div className="p-4 rounded-xl border border-red-900/30 bg-red-950/20 text-center text-sm text-red-300">
            Hold expires in <span className="font-mono font-bold">04:59</span>
          </div>
        </div>
      </div>
    </CheckoutLayout>
  );
};
