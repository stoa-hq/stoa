DROP INDEX IF EXISTS idx_api_keys_created_by;
ALTER TABLE api_keys DROP COLUMN IF EXISTS created_by;
