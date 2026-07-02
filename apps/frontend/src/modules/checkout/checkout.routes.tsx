import { RouteObject } from "react-router-dom";
import { CheckoutPage } from "./checkout.page";
import { ConfirmationPage } from "./confirmation.page";

export const checkoutRoutes: RouteObject[] = [
  {
    path: "/checkout",
    element: <CheckoutPage />,
  },
  {
    path: "/confirmation",
    element: <ConfirmationPage />,
  },
];
