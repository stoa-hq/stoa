ALTER TABLE shipping_methods ADD COLUMN tax_rule_id UUID REFERENCES tax_rules(id) ON DELETE SET NULL;
