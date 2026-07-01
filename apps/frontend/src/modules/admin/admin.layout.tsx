import React from "react";

export interface AdminLayoutProps {
  children?: React.ReactNode;
}

export const AdminLayout: React.FC<AdminLayoutProps> = ({ children }) => {
  return (
    <div className="max-w-7xl mx-auto px-4">
      {children}
    </div>
  );
};
