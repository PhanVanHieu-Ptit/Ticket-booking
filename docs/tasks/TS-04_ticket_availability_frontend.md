# Task: Real-Time Ticket Availability - Frontend

## Task ID

`TS-04`

## Goal

Initialize the React frontend application using Vite, TypeScript, and Tailwind CSS. Implement the Event Home Page UI displaying the ticket categories, their prices, and remaining availability. Connect to the backend SSE stream to update the counts reactively.

## Files Expected to Change

- [NEW] `frontend/package.json`
- [NEW] `frontend/vite.config.ts`
- [NEW] `frontend/src/main.tsx`
- [NEW] `frontend/src/index.css` (Tailwind CSS configuration and custom global styles)
- [NEW] `frontend/src/App.tsx` (main application shell)
- [NEW] `frontend/src/components/TicketCategoryCard.tsx` (card displaying category name, price, available count, and a "Reserve" button)
- [NEW] `frontend/src/hooks/useTicketAvailability.ts` (custom React hook that handles fetching initial counts and subscribing to the SSE stream)

## Dependencies

- `TS-03` (Requires the backend REST availability endpoint and SSE stream to be functional)

## Acceptance Criteria

1. **Modern Premium UI**: The page is styled using Tailwind CSS, featuring a dark/glassmorphism design, custom typography, and hover micro-animations on the cards and buttons.
2. **Initial Render**: The page successfully fetches and displays the initial ticket counts (100 VIP, 400 Standard) and prices ($100, $50) from `GET /api/v1/tickets/availability`.
3. **Reactive Updates**: The UI updates the remaining ticket counts in real-time (within 2 seconds) when a message is received from the SSE stream `/api/v1/tickets/availability/stream` without requiring a page refresh.
4. **Sold Out States**:
   - If a category's count reaches `0`, its "Reserve" button is disabled and its text changes to "Sold Out".
   - If the total inventory (500 tickets) is sold out, a prominent "Sold Out" banner is displayed at the top of the page and all buttons are disabled.
5. **Resilience & Optimization**:
   - The SSE hook implements automatic reconnection with exponential backoff if the connection drops.
   - Page Visibility API is used to close the SSE connection if the tab is inactive for more than 3 minutes, and re-establish it immediately when the tab becomes active.

## Estimated Complexity

Medium (3 - 4 hours)
