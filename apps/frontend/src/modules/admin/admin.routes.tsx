import { RouteObject } from "react-router-dom";
import { AdminPage } from "./admin.page";

export const adminRoutes: RouteObject[] = [
  {
    path: "/admin",
    element: <AdminPage />,
  },
];
