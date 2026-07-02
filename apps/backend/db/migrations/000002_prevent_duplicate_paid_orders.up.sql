-- Final safety net against double-selling a ticket: even if application-level
-- locking (SELECT ... FOR UPDATE in Checkout) were ever bypassed, the database
-- itself refuses to store a second 'Paid' order for the same ticket.
CREATE UNIQUE INDEX idx_orders_ticket_id_paid
    ON orders(ticket_id)
    WHERE status = 'Paid';
