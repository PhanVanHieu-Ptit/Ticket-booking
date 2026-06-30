# Project Context: Concert Ticket Booking System

This document establishes the strategic, business, and user context for the Concert Ticket Booking System. It translates the raw product requirements into a structured framework that guides architectural decisions, user experience design, and quality assurance.

---

## 1. Product Goal

The **Concert Ticket Booking System** aims to deliver a **highly reliable, fair, and high-performance transaction platform** for high-demand event ticketing.

The primary goal is to successfully sell a limited inventory of **500 tickets** to a pool of approximately **5,000 highly active, concurrent users** at the exact moment of opening. The system must guarantee that:

1. **No overselling occurs** (exactly 500 tickets are sold, or fewer if demand is lower).
2. **Users experience a fair, transparent, and responsive interface** even under heavy load.
3. **Inventory is optimized** by temporarily holding tickets during checkout and immediately reclaiming abandoned carts.

---

## 2. User Personas

### 2.1. The Ticket Buyer (General User)

- **Description**: A passionate music fan eager to secure a ticket to a highly anticipated concert.
- **Behaviors**:
  - Arrives at the website minutes before the sale starts.
  - Rapidly refreshes the page (F5) at the exact second the sale opens.
  - Wants to select their ticket type, secure a hold, and complete their payment as quickly as possible.
- **Pain Points**:
  - Website crashes, slow loading states, or unresponsive buttons under load.
  - "Cart sniping" where a ticket is snatched away _after_ they have already clicked "buy" and started typing payment info.
  - Lack of clarity on whether tickets are actually sold out or just temporarily held.
- **Needs**:
  - Immediate visual feedback on ticket availability.
  - A guaranteed, stress-free reservation window (5 minutes) to complete payment.
  - Protection against accidental double-clicks or double-charges.

### 2.2. The Concert Organizer / Administrator

- **Description**: The business owner or event organizer responsible for the financial and operational success of the concert.
- **Behaviors**:
  - Monitors the progress of the ticket sale in real-time.
  - Tracks revenue and velocity of sales.
  - Reviews inventory states to ensure the system is operating correctly.
- **Pain Points**:
  - Lack of visibility into how many tickets are currently locked in carts versus actually sold.
  - Suspicion of system abuse (e.g., scalpers holding tickets to block genuine buyers).
  - Inability to verify real-time revenue and sales stats.
- **Needs**:
  - A clear, real-time dashboard displaying total tickets sold, total revenue, and currently locked tickets.
  - High system reliability to avoid reputational damage and lost revenue.

---

## 3. Business Problems

| Problem                            | Business Impact                                                                                                                                                    | System Mitigation                                                                                                                      |
| :--------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------- |
| **Overselling (Double Booking)**   | Customers arrive at the venue with valid receipts but no seats, leading to severe reputational damage, customer support overload, and potential legal liabilities. | Strict concurrency control and atomic inventory updates on the backend.                                                                |
| **Inventory Lockup (Hoarding)**    | Users add tickets to their cart but never check out, preventing other genuine buyers from purchasing them and leaving the organizer with unsold inventory.         | A strict 5-minute hold-and-release mechanism that automatically reclaims abandoned tickets.                                            |
| **System Outages under Peak Load** | The server crashes under the weight of 5,000 concurrent users refreshing at the same second, resulting in lost sales, frustrated customers, and negative press.    | Lightweight frontend pages, efficient backend request processing, and real-time communication protocols that minimize server overhead. |
| **Frustrating User Experience**    | Slow API responses cause users to click "Submit" multiple times, resulting in duplicate reservations, redundant processing, and user confusion.                    | Frontend debouncing/disable states and backend idempotency.                                                                            |

---

## 4. Functional Requirements

### 4.1. Customer-Facing Features

- **Real-time Inventory Display**: The Event Home Page must display the remaining count of available tickets. This count must update in real-time as tickets are held or sold, without requiring the user to manually refresh the page.
- **Ticket Selection**: A dedicated Booking Page allows users to select their desired ticket category.
- **Temporary Ticket Hold**:
  - Selecting a ticket type triggers a temporary **5-minute reservation**.
  - The ticket is locked to that specific user session; it cannot be selected or purchased by anyone else during this window.
- **Hold Countdown**: The Booking Page must display a live, synchronized **5-minute countdown timer** showing the remaining time to complete the transaction.
- **Simulated Payment Checkout**:
  - An interface for users to trigger a simulated payment.
  - A backend endpoint to receive payment confirmations and permanently transition the held ticket to `Sold`.
- **Auto-Release**: If the 5-minute timer expires before payment is confirmed, the ticket must immediately transition back to `Available` and become purchasable by other users.

### 4.2. Administrative Features

- **Sales Analytics**: A dashboard showing:
  - **Total Tickets Sold** (cumulative count).
  - **Total Revenue Generated** (monetary sum based on sold ticket prices).
- **Live Lock Monitor**: A real-time list of all tickets currently in the `Holding` state, including their hold expiration times.

---

## 5. Non-Functional Requirements

### 5.1. Performance & Concurrency

- **High-Concurrency Handling**: The system must handle a sudden spike of **5,000 concurrent users** at the moment of launch, processing rapid read and write requests without dropping transactions or degrading response times.
- **Race Condition Prevention**: The inventory deduction and hold mechanisms must be atomic. Under no circumstances can two concurrent requests successfully reserve or buy the same ticket.

### 5.2. User Experience (UX) Under Load

- **Spam Prevention**: Interactive elements (like the "Reserve" or "Pay" buttons) must immediately disable upon click to prevent double-submits.
- **Graceful Degradation**: The frontend must handle slow API responses or network latency with clear loading indicators, preventing the user from thinking the application has frozen.
- **State Synchronization**: The countdown timer must remain synchronized with the server's lock duration, preventing discrepancies where a user pays for a ticket that the server has already released.

### 5.3. Reliability & Security

- **Session-Based Isolation**: Ticket holds must be securely tied to a unique client session (e.g., via temporary token or session identifier) to prevent users from hijacking or paying for other users' holds.
- **Input Validation**: All API entry points must validate incoming payloads to reject malformed data, negative quantities, or invalid ticket types before they reach the core business logic.
- **Centralized Error Handling**: The system must catch and handle errors gracefully, returning clean, standard error messages to the client instead of raw system stack traces.

---

## 6. Hidden / Implicit Requirements

- **Idempotency**: The reservation and payment endpoints must be idempotent. If a network retry occurs, the system should not create duplicate holds or process duplicate payments for the same session.
- **Active vs. Passive Ticket Release**: To keep the Admin Dashboard and inventory counts accurate in real-time, the system cannot rely solely on "lazy" expiration (checking if a ticket is expired only when someone tries to buy it). It requires an active background mechanism or time-to-live (TTL) trigger to release expired holds and push the updated counts immediately.
- **State Transition Integrity**: The ticket state machine must strictly enforce valid transitions:
  - `Available` $\rightarrow$ `Holding` (Valid)
  - `Holding` $\rightarrow$ `Sold` (Valid, via payment)
  - `Holding` $\rightarrow$ `Available` (Valid, via timeout or cancellation)
  - `Available` $\rightarrow$ `Sold` (Invalid, must go through `Holding` first to ensure fairness)
  - `Sold` $\rightarrow$ any state (Invalid, once sold, a ticket cannot be modified or re-held)
- **Real-time Push Efficiency**: To support 5,000 concurrent connections receiving real-time ticket updates, the server must use an event-driven broadcast model rather than individual database queries per connected client.

---

## 7. Key Risks & Mitigations

| Risk                                      | Impact | Mitigation Strategy                                                                                                                                                                                                                                                               |
| :---------------------------------------- | :----- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Database Bottlenecks & Deadlocks**      | High   | Avoid heavy, long-lived database transactions or row-level locks on the main database under peak write load. Use memory-optimized structures, atomic check-and-set operations, or distributed locks to manage the hot-path inventory.                                             |
| **Timer Desynchronization**               | Medium | Do not rely on client-side clocks for the 5-minute window. The server must dictate the exact expiration timestamp, and the frontend countdown must synchronize with the server's time.                                                                                            |
| **Network Latency during Payment**        | High   | If a user initiates payment at 4 minutes and 59 seconds, and the network request takes 2 seconds, the ticket must not be released mid-transit. The system should either grant a short grace period during active payment processing or lock the state during the payment attempt. |
| **Memory Exhaustion / Connection Limits** | High   | Maintaining 5,000 active real-time connections (WebSockets/SSE) can exhaust server file descriptors or memory. Implement connection limits, lightweight heartbeat messages, and resource cleanups.                                                                                |

---

## 8. Success Metrics

- **Zero Overselling**: 100% compliance. The total number of tickets in the `Sold` state must never exceed the initial inventory limit (500 tickets).
- **Zero Orphaned Holds**: All tickets in the `Holding` state must either transition to `Sold` or return to `Available` within exactly 5 minutes of their lock creation. No tickets should remain permanently locked in a holding state.
- **Peak Load Stability**: The application maintains $\ge 99.9\%$ uptime during the initial 15-minute rush, with average API response times under **200ms** for the reservation and payment endpoints.
- **Data Consistency**: The real-time ticket count displayed on the Event Home Page matches the actual database inventory state at all times.
