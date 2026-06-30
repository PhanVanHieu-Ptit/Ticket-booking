# Task: Administration & Analytics

## Task ID

`TS-11`

## Goal

Implement the Administration & Analytics features. Create backend endpoints to retrieve sales metrics (tickets sold, revenue, availability) and the list of active holds. Implement a secure, passcode-protected Admin Dashboard (`/admin`) on the frontend to display these metrics and the live hold monitor in real-time.

## Files Expected to Change

- [NEW] `internal/handlers/admin_metrics_handler.go` (implements `GET /api/v1/admin/metrics` and `GET /api/v1/admin/holds`)
- [NEW] `frontend/src/pages/AdminDashboardPage.tsx` (the Admin Dashboard UI)
- [NEW] `frontend/src/components/AdminLoginModal.tsx` (passcode prompt modal for admin access)
- [MODIFY] `frontend/src/App.tsx` (add routing for `/admin`)
- [MODIFY] `main.go` (register the admin metrics routes with the admin authentication middleware)

## Dependencies

- `TS-02` (Requires admin authentication middleware)
- `TS-09` (Requires orders data to calculate revenue)

## Acceptance Criteria

1. **Admin Endpoints**:
   - `GET /api/v1/admin/metrics` returns:
     - `total_tickets_sold` (count of tickets with status `Sold`).
     - `total_revenue` (sum of prices of all `Sold` tickets).
     - `remaining_inventory` and `held_inventory` breakdowns.
   - `GET /api/v1/admin/holds` returns an array of tickets currently in the `Holding` state, including their ticket ID, category, session ID, and remaining hold duration in seconds.
   - Both endpoints require a valid admin JWT in the `Authorization: Bearer <token>` header. If invalid or missing, return `401 Unauthorized`.
2. **Admin Login Prompt**:
   - Accessing `/admin` on the frontend displays a login prompt requiring a passcode.
   - Submitting the passcode calls `POST /api/v1/admin/login`. On success, the received JWT token is stored (e.g., in `sessionStorage`), and the dashboard is revealed.
3. **Admin Dashboard UI**:
   - The dashboard features a clean, professional admin interface.
   - It displays cards for key metrics: **Total Sold**, **Total Revenue**, and **Inventory Status** (VIP vs. Standard).
   - It displays a **Live Hold Monitor** table listing all active holds (Ticket ID, Category, Session ID, and a live countdown timer for each hold).
4. **Real-time Synchronization**: The dashboard metrics and active holds table refresh every 5 seconds to provide the organizer with up-to-date sales data.
5. **Logout**: A "Logout" button is available on the dashboard, which clears the admin token and redirects the user to the login prompt.

## Estimated Complexity

Medium (4 - 5 hours)
