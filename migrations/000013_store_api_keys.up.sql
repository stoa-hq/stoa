ALTER TABLE api_keys ADD COLUMN key_type VARCHAR(10) NOT NULL DEFAULT 'admin';
ALTER TABLE api_keys ADD COLUMN customer_id UUID REFERENCES customers(id) ON DELETE CASCADE;
CREATE INDEX idx_api_keys_customer_id ON api_keys (customer_id);
CREATE INDEX idx_api_keys_key_type ON api_keys (key_type);
