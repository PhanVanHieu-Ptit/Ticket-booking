import React from "react";

export interface AdminLayoutProps {
  children?: React.ReactNode;
}

export const AdminLayout: React.FC<AdminLayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-50 font-sans flex flex-col">
      <header className="border-b border-zinc-800 bg-zinc-900/80 backdrop-blur sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-red-400 to-orange-500 bg-clip-text text-transparent">
            Admin Console
          </h1>
          <div className="flex items-center gap-4">
            <span className="text-xs bg-red-950 text-red-400 px-2.5 py-1 rounded-full border border-red-900/50">
              Administrator Mode
            </span>
          </div>
        </div>
      </header>
      <div className="flex-1 max-w-7xl w-full mx-auto px-6 py-8">{children}</div>
    </div>
  );
};
