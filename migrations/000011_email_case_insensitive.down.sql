-- Drop case-insensitive indexes.
DROP INDEX IF EXISTS idx_admin_users_email_lower;
DROP INDEX IF EXISTS idx_customers_email_lower;

-- Restore original UNIQUE constraints.
ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
ALTER TABLE customers ADD CONSTRAINT customers_email_key UNIQUE (email);
