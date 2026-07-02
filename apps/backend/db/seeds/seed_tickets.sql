-- 1. Seed VIP Tickets (100 tickets, IDs 1-100, price $100.00)
INSERT INTO tickets (id, ticket_code, category, price, status)
SELECT
    i,
    'TKT-VIP-' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 8)),
    'VIP',
    100.00,
    'Available'
FROM generate_series(1, 100) AS i;

-- 2. Seed Standard Tickets (400 tickets, IDs 101-500, price $50.00)
INSERT INTO tickets (id, ticket_code, category, price, status)
SELECT
    i,
    'TKT-STD-' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 8)),
    'Standard',
    50.00,
    'Available'
FROM generate_series(101, 500) AS i;

-- Adjust the sequence to prevent ID collision in future inserts
SELECT setval('tickets_id_seq', 500);

-- 3. Seed Default Admin Config (passcode: 'admin123' bcrypt-hashed)
INSERT INTO admin_configs (key, value, description)
VALUES ('admin_passcode', '$2a$12$LDPGLECY88uBIEuTm4nvQe3bHsE4HwyXX8VQCRj7hYpUAJEUVLdE6', 'Bcrypt hash of the admin dashboard access passcode');
