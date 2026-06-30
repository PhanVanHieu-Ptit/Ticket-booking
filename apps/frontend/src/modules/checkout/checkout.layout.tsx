import React from "react";

export interface CheckoutLayoutProps {
  children?: React.ReactNode;
}

export const CheckoutLayout: React.FC<CheckoutLayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-50 font-sans">
      <header className="border-b border-zinc-800 bg-zinc-900/50 backdrop-blur sticky top-0 z-50">
        <div className="max-w-4xl mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-emerald-400 to-teal-500 bg-clip-text text-transparent">
            Secure Checkout
          </h1>
          <div className="text-sm text-zinc-400 font-medium">SSL Encrypted</div>
        </div>
      </header>
      <main className="max-w-4xl mx-auto px-4 py-8">{children}</main>
    </div>
  );
};
