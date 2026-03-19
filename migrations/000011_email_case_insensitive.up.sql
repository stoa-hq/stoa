-- Normalize existing emails to lowercase.
UPDATE admin_users SET email = LOWER(TRIM(email)) WHERE email != LOWER(TRIM(email));
UPDATE customers SET email = LOWER(TRIM(email)) WHERE email != LOWER(TRIM(email));

-- Replace plain UNIQUE constraints with case-insensitive functional indexes.
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_email_key;
ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_email_key;

CREATE UNIQUE INDEX idx_admin_users_email_lower ON admin_users (LOWER(email));
CREATE UNIQUE INDEX idx_customers_email_lower ON customers (LOWER(email));
