# Hold Reclamation Module

The **Hold Reclamation Module** is an internal, event-driven background service. It is responsible for releasing expired ticket holds and returning them to the available inventory pool, ensuring that tickets are not locked indefinitely if a user abandons their checkout session.

---

## 1. Purpose

To guarantee inventory accuracy and fairness by reclaiming held tickets that were not paid for within the 5-minute (300 seconds) reservation window. It ensures that abandoned tickets are immediately made available to other buyers, keeping the real-time inventory count up to date.

---

## 2. Responsibilities

- **Keyspace Notification Listening**: Listen to Redis `expired` keyspace events for `hold:{session_id}` keys.
- **Immediate Reclamation**: Upon receiving an expiration event, execute an atomic database transaction to:
  1. Lock the ticket row using pessimistic locking (`FOR UPDATE`).
  2. Verify the ticket is still in the `Holding` state.
  3. Reset the ticket status to `Available`, and clear `session_id`, `held_at`, and `expires_at`.
- **Redis Inventory Restoration**: Add the reclaimed ticket ID back to the corresponding Redis category pool (`tickets:available:VIP` or `tickets:available:Standard`).
- **Fail-Safe Polling**: Run a periodic database query (every 10 seconds) to find and reclaim any tickets in the `Holding` state where `expires_at < NOW()`. This acts as a backup in case of Redis keyspace event delivery failure, network partition, or server restart.
- **Inventory Update Notification**: Publish an inventory update event to Redis Pub/Sub, triggering a real-time SSE update to all connected clients.

---

## 3. Dependencies

- **[Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)**: Directly modifies the data owned by the Inventory & Reservation Module (the `tickets` table in PostgreSQL and the availability sets/hold keys in Redis) to release the hold and restore availability.

---

## 4. APIs

This module does not expose any public or administrative HTTP endpoints. It operates entirely as an **internal, event-driven background worker** and a scheduled cron job.

---

## 5. Data Ownership

This module does not own any private data models. Instead, it operates on data owned by other modules:

- Modifies the `tickets` table (owned by [Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)).
- Modifies the Redis available ticket sets and hold keys (owned by [Inventory & Reservation Module](file:///Users/phanvanhieu/Documents/CaNhan/MyProject/Ticket-booking/docs/modules/inventory_reservation.md)).
