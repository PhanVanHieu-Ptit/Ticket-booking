# Administration & Analytics Module

The **Administration & Analytics Module** provides event organizers with secure access to real-time sales metrics, revenue data, and a live monitor of active ticket holds.

---

## 1. Purpose

To enable event administrators to monitor the financial performance and inventory status of the concert ticket sales. It offers visibility into active ticket holds, allowing organizers to see how much inventory is currently locked in checkout carts and how much has been permanently sold.

---

## 2. Responsibilities

- **Administrative Authorization**: Protect all admin endpoints by verifying administrative JWT tokens.
- **Sales Metrics Calculation**: Calculate and expose real-time metrics, including:
  - Total tickets sold.
  - Total revenue generated (calculated as `(VIP_Sold * 100) + (Standard_Sold * 50)`).
  - Remaining inventory breakdown (VIP vs. Standard).
- **Live Lock Monitoring**: Query and expose a list of all tickets currently in the `Holding` state, including their associated session IDs and remaining hold durations (seconds remaining until expiration).

---

## 3. Dependencies

- **[Session Management Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/session_management.md)**: Relies on this module to validate the administrative Bearer token.
- **[Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)**: Relies on this module's data to query ticket availability status and retrieve details of active holds.
- **[Checkout & Payment Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/checkout_payment.md)**: Relies on this module's data to calculate total revenue and count completed orders.

---

## 4. APIs

All endpoints require the `Authorization: Bearer <admin_token>` header.

- **`GET /api/v1/admin/metrics`**: Retrieves overall sales, revenue, and inventory metrics.
  - _Response_:
    ```json
    {
      "total_tickets_sold": 120,
      "total_revenue": 8500.0,
      "remaining_inventory": { "VIP": 45, "Standard": 335 },
      "held_inventory": { "VIP": 10, "Standard": 15 },
      "available_inventory": { "VIP": 45, "Standard": 50 }
    }
    ```
- **`GET /api/v1/admin/holds`**: Retrieves a list of all tickets currently in the `Holding` state along with remaining hold times.
  - _Response_: Returns an array of active hold details.

---

## 5. Data Ownership

This module does not own any private operational tables or Redis keys. It functions as a read-only reporting layer that queries data owned by other modules:

- Queries the `tickets` table (owned by [Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)).
- Queries the `orders` table (owned by [Checkout & Payment Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/checkout_payment.md)).
