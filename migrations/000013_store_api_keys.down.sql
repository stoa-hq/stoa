DROP INDEX IF EXISTS idx_api_keys_key_type;
DROP INDEX IF EXISTS idx_api_keys_customer_id;
ALTER TABLE api_keys DROP COLUMN IF EXISTS customer_id;
ALTER TABLE api_keys DROP COLUMN IF EXISTS key_type;
