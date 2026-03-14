DROP INDEX IF EXISTS idx_orders_guest_token;
ALTER TABLE orders DROP COLUMN IF EXISTS guest_token;
