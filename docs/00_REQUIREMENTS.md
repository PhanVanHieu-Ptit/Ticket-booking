# Product Requirements: Concert Ticket Booking System

This document outlines the product requirements, technical focus areas, architectural assumptions, and open questions for the high-concurrency ticket booking system.

---

## 1. Confirmed Requirements

### 1.1. Scenario & Scale

- **Event scale**: A music concert with a limited supply of **500 tickets**.
- **Traffic profile**: Approximately **5,000 concurrent users** accessing the system at the moment of opening, performing rapid page refreshes (F5) and attempting to book tickets simultaneously.
- **Scope**: A fullstack application comprising a **Backend API** and a **Frontend Web Application**.

### 1.2. Backend API Requirements

- **Ticket Inventory Management**:
  - The ticket data structure must include:
    - `Ticket Type` (e.g., VIP, Standard)
    - `Price`
    - `Inventory / Remaining Quantity`
    - `Status` (with at least three states: `Available` / `Holding` / `Sold`)
- **Hold & Reserve Flow**:
  - When a user selects a ticket, the system must temporarily hold/reserve that ticket for **5 minutes**.
  - During this 5-minute window, the held ticket must be locked; no other user can select or reserve it.
  - If the user completes the payment within 5 minutes, the ticket status transitions to `Sold`.
  - If the user fails to complete the payment within 5 minutes, the ticket must be automatically **released** back into the available inventory.
- **Simulated Payment**:
  - An API endpoint to receive and process simulated payment success notifications.
  - Upon successful payment, the status of the corresponding ticket must transition to `Sold`.

### 1.3. Frontend Web Requirements

- **Event Home Page**:
  - Must display the number of remaining tickets in **real-time**.
- **Booking Page**:
  - Screen to select the ticket type.
  - Displays a **5-minute countdown timer** indicating the remaining time for the ticket hold.
- **Admin Dashboard**:
  - A simple administrative interface to view:
    - Total tickets sold.
    - Total revenue generated.
    - A list of tickets currently temporarily locked (held).

### 1.4. Technical Evaluation Focus

- **Concurrency & Race Conditions (Backend)**:
  - Implement an anti-overselling mechanism. The system must prevent selling more than the 500 available tickets even when thousands of requests arrive at the exact same millisecond.
- **High-Load UX (Frontend)**:
  - Gracefully handle network latency or slow server responses.
  - Prevent spam clicks (double-submits).
  - Provide clear loading states.
  - Keep the countdown timer synchronized.
- **Code Quality (Clean Code)**:
  - Structured project directory layout.
  - Centralized global error handling.
  - Input data validation at the API gateway/entry points.
  - Unit tests covering core business logic.

---

## 2. Assumptions

To establish a clear technical direction without altering the core scope, the following assumptions are made:

1. **Ticket Granularity**: Tickets are treated as individual, uniquely identifiable units (e.g., each ticket has a unique ID, and optionally a seat/serial number) rather than just a simple numeric counter. This allows precise tracking of which specific ticket is `Holding` or `Sold`.
2. **User Identification**: To associate a ticket hold with a specific client, the system will identify users using a temporary session identifier (e.g., a UUID or JWT stored in the browser) without requiring a full, permanent user registration flow.
3. **Simulated Payment Scope**: The payment process is entirely simulated. No actual integration with third-party payment gateways (e.g., Stripe, PayPal, VNPay) is required. A mock API call from the frontend representing a successful transaction is sufficient.
4. **Real-time Communication**: The real-time updates for ticket counts on the Event Home Page will be implemented using a lightweight, efficient push mechanism (e.g., WebSockets or Server-Sent Events) rather than aggressive HTTP short-polling, to reduce server load under high traffic.
5. **Admin Access**: The Admin Dashboard is intended for demonstration purposes and will be accessible via a dedicated route without requiring complex multi-role authorization systems, though basic protection can be discussed.

---

## 3. Open Questions

The following questions will help refine the implementation details:

> [!IMPORTANT]
> These questions should be aligned upon before final design lock.

1. **Ticket Categories & Quantities**:
   - Are the 500 tickets divided into multiple categories (e.g., 100 VIP tickets at $100, 400 Standard tickets at $50), or is it a single ticket type?
2. **Purchase Limits**:
   - Is a user allowed to hold and purchase multiple tickets at once, or is there a strict limit of **1 ticket per user** to prevent scalping and ensure fair distribution?
3. **Admin Dashboard Security**:
   - Should the Admin Dashboard be publicly accessible for testing convenience, or do we need basic authentication (e.g., a simple password or hardcoded admin token)?
4. **Countdown Expiry Behavior**:
   - When the 5-minute countdown expires on the frontend, should the user be redirected back to the event home page automatically with an expiration message, or should the booking page update inline?
5. **Data Persistence**:
   - Do we need the ticket states to persist across application restarts (using a database like PostgreSQL/MySQL), or is an in-memory database/cache (like Redis or in-memory data structures with file persistence) acceptable for the scope of this project?
