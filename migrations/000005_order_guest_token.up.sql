ALTER TABLE orders ADD COLUMN guest_token VARCHAR(64);

CREATE UNIQUE INDEX idx_orders_guest_token
    ON orders (guest_token) WHERE guest_token IS NOT NULL;
