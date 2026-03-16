-- warehouses: multi-warehouse inventory management
CREATE TABLE warehouses (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(255) NOT NULL,
    code          VARCHAR(50)  NOT NULL UNIQUE,
    active        BOOLEAN NOT NULL DEFAULT true,
    priority      INTEGER NOT NULL DEFAULT 0,
    address_line1 VARCHAR(255) DEFAULT '',
    address_line2 VARCHAR(255) DEFAULT '',
    city          VARCHAR(255) DEFAULT '',
    state         VARCHAR(255) DEFAULT '',
    postal_code   VARCHAR(50)  DEFAULT '',
    country       VARCHAR(2)   DEFAULT '',
    custom_fields JSONB DEFAULT '{}',
    metadata      JSONB DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- warehouse_stock: inventory per product/variant per warehouse
CREATE TABLE warehouse_stock (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id   UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (warehouse_id, product_id, variant_id)
);

-- Partial unique index for rows without a variant
CREATE UNIQUE INDEX uq_warehouse_stock_product_only
    ON warehouse_stock (warehouse_id, product_id) WHERE variant_id IS NULL;

-- stock_movements: audit trail for inventory changes
CREATE TABLE stock_movements (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    warehouse_id   UUID NOT NULL REFERENCES warehouses(id),
    product_id     UUID NOT NULL REFERENCES products(id),
    variant_id     UUID REFERENCES product_variants(id),
    order_id       UUID REFERENCES orders(id),
    movement_type  VARCHAR(50) NOT NULL,
    quantity       INTEGER NOT NULL,
    reference      VARCHAR(255) DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_warehouse_stock_product ON warehouse_stock (product_id);
CREATE INDEX idx_warehouse_stock_variant ON warehouse_stock (variant_id) WHERE variant_id IS NOT NULL;
CREATE INDEX idx_stock_movements_order   ON stock_movements (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_stock_movements_product ON stock_movements (product_id);

-- Data migration: create default warehouse and migrate existing stock values
INSERT INTO warehouses (id, name, code, active, priority)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Warehouse', 'DEFAULT', true, 0);

-- Migrate product-level stock to warehouse_stock
INSERT INTO warehouse_stock (warehouse_id, product_id, variant_id, quantity)
SELECT '00000000-0000-0000-0000-000000000001', p.id, NULL, p.stock
FROM products p
WHERE p.stock > 0;

-- Migrate variant-level stock to warehouse_stock
INSERT INTO warehouse_stock (warehouse_id, product_id, variant_id, quantity)
SELECT '00000000-0000-0000-0000-000000000001', pv.product_id, pv.id, pv.stock
FROM product_variants pv
WHERE pv.stock > 0;
