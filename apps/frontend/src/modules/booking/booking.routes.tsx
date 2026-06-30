import { RouteObject } from "react-router-dom";
import { BookingPage } from "./booking.page";

export const bookingRoutes: RouteObject[] = [
  {
    path: "/",
    element: <BookingPage />,
  },
];
