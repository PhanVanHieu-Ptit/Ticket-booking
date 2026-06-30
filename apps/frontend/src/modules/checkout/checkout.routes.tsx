import { RouteObject } from "react-router-dom";
import { CheckoutPage } from "./checkout.page";

export const checkoutRoutes: RouteObject[] = [
  {
    path: "/checkout",
    element: <CheckoutPage />,
  },
];
