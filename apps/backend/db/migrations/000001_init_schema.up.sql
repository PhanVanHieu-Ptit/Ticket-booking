CREATE TABLE tickets (
    id SERIAL PRIMARY KEY,
    ticket_code VARCHAR(64) UNIQUE NOT NULL,
    category VARCHAR(20) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Available',
    session_id VARCHAR(255) NULL,
    held_at TIMESTAMP WITH TIME ZONE NULL,
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_ticket_price CHECK (price >= 0),
    CONSTRAINT chk_ticket_status CHECK (status IN ('Available', 'Holding', 'Sold')),
    CONSTRAINT chk_hold_dates CHECK (
        (status = 'Available' AND session_id IS NULL AND held_at IS NULL AND expires_at IS NULL) OR
        (status = 'Holding' AND session_id IS NOT NULL AND held_at IS NOT NULL AND expires_at IS NOT NULL) OR
        (status = 'Sold' AND session_id IS NOT NULL)
    ),
    CONSTRAINT chk_expiration_after_hold CHECK (expires_at > held_at)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id INT NOT NULL,
    session_id VARCHAR(255) UNIQUE NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'Paid',
    email VARCHAR(255) NOT NULL,
    card_holder_name VARCHAR(255) NOT NULL,
    payment_reference VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE RESTRICT,
    CONSTRAINT chk_order_status CHECK (status IN ('Paid', 'Refunded', 'Failed')),
    CONSTRAINT chk_order_amount CHECK (amount >= 0)
);

CREATE TABLE admin_configs (
    key VARCHAR(50) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE UNIQUE INDEX idx_tickets_session_id_unique ON tickets(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_tickets_expires_at_holding ON tickets(expires_at) WHERE status = 'Holding';
CREATE INDEX idx_tickets_status_category ON tickets(status, category);
CREATE INDEX idx_orders_email ON orders(email);
