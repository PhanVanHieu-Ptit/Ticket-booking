# Task: Atomic Ticket Reservation - Frontend

## Task ID

`TS-06`

## Goal

Implement the user journey from the Event Home Page to the Booking/Checkout Page. Connect the "Reserve" button to the backend reservation API, handle redirection, and implement a synchronized 5-minute countdown timer on the Checkout Page that disables the page and shows an expiration modal when the timer reaches zero.

## Files Expected to Change

- [NEW] `frontend/src/pages/CheckoutPage.tsx` (the Booking/Checkout Page shell)
- [NEW] `frontend/src/components/CountdownTimer.tsx` (the countdown timer component)
- [NEW] `frontend/src/components/ExpirationModal.tsx` (modal overlay shown when the reservation expires)
- [MODIFY] `frontend/src/App.tsx` (add routing for `/checkout`)
- [MODIFY] `frontend/src/components/TicketCategoryCard.tsx` (wire up the "Reserve" button action)

## Dependencies

- `TS-04` (Requires the frontend project setup and Home Page UI)
- `TS-05` (Requires the backend reservation API and active hold query endpoint)

## Acceptance Criteria

1. **Button Debouncing**: Clicking the "Reserve" button immediately disables the button and displays a loading state to prevent double-clicks.
2. **Redirection**: On successful reservation (`200 OK` from `/api/v1/tickets/reserve`), the user is redirected to `/checkout`.
3. **Timer Initialization**: The Checkout Page displays a countdown timer initialized to `05:00` (or the remaining seconds returned by the server).
4. **Server-Side Synchronization**:
   - The countdown timer ticks down second-by-second in `MM:SS` format.
   - On page refresh, the page calls `GET /api/v1/tickets/hold` to retrieve the actual remaining seconds from the server. The timer resumes from this count, preventing client-side clock manipulation.
5. **Expiration Handling**:
   - When the timer reaches `00:00`, the checkout form inputs are set to read-only, and the payment button is disabled.
   - An overlay modal (`ExpirationModal`) is displayed with the message: _"Your reservation has expired, and your ticket has been released back to the pool."_
   - The modal contains a single button: _"Return to Home Page"_, which redirects the user back to the Home Page.

## Estimated Complexity

Medium (4 - 5 hours)
