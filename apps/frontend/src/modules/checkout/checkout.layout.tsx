import React from "react";

export interface CheckoutLayoutProps {
  children?: React.ReactNode;
}

export const CheckoutLayout: React.FC<CheckoutLayoutProps> = ({ children }) => {
  return (
    <div className="max-w-5xl mx-auto px-4">
      {children}
    </div>
  );
};
