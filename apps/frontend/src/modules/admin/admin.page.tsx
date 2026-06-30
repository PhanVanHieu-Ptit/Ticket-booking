import React from "react";
import { AdminLayout } from "./admin.layout";

export const AdminPage: React.FC = () => {
  return (
    <AdminLayout>
      <div className="space-y-8">
        <section className="flex justify-between items-center">
          <div>
            <h2 className="text-3xl font-extrabold tracking-tight">Real-time Metrics</h2>
            <p className="text-zinc-400 text-sm">Monitor sales progress and active reservation holds.</p>
          </div>
        </section>

        {/* Analytics Grid */}
        <section className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/40">
            <div className="text-sm text-zinc-400">Total Tickets Sold</div>
            <div className="text-3xl font-black mt-2">0</div>
          </div>
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/40">
            <div className="text-sm text-zinc-400">Total Revenue</div>
            <div className="text-3xl font-black mt-2 text-emerald-400">$0.00</div>
          </div>
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/40">
            <div className="text-sm text-zinc-400">Active Holds</div>
            <div className="text-3xl font-black mt-2 text-amber-400">0</div>
          </div>
          <div className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/40">
            <div className="text-sm text-zinc-400">Available Inventory</div>
            <div className="text-3xl font-black mt-2 text-blue-400">500</div>
          </div>
        </section>

        {/* Holds List Table */}
        <section className="p-6 rounded-xl border border-zinc-800 bg-zinc-900/20 space-y-4">
          <h3 className="text-lg font-bold">Active Reservations Queue</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 text-xs uppercase">
                  <th className="py-3 px-4">Ticket ID</th>
                  <th className="py-3 px-4">Category</th>
                  <th className="py-3 px-4">Session ID</th>
                  <th className="py-3 px-4">Time Remaining</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-zinc-900 text-zinc-500 text-sm">
                  <td className="py-4 px-4" colSpan={4}>
                    No active holds found.
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </AdminLayout>
  );
};
