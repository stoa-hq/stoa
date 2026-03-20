ALTER TABLE api_keys ADD COLUMN created_by UUID REFERENCES admin_users(id) ON DELETE SET NULL;
CREATE INDEX idx_api_keys_created_by ON api_keys (created_by);
