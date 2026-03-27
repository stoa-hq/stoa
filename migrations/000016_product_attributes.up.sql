-- Product Attribute System: generic, admin-defined attributes for products and variants.
-- Separate from property_groups (which define variant combinations like Size × Color).

-- Attribute definitions
CREATE TABLE attributes (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(255) NOT NULL,
    type       VARCHAR(20)  NOT NULL CHECK (type IN ('text', 'number', 'select', 'multi_select', 'boolean')),
    unit       VARCHAR(20),
    position   INTEGER      NOT NULL DEFAULT 0,
    filterable BOOLEAN      NOT NULL DEFAULT false,
    required   BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX attributes_identifier_key ON attributes (identifier);

-- Attribute translations (i18n: name + description per locale)
CREATE TABLE attribute_translations (
    attribute_id UUID        NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
    locale       VARCHAR(10) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    PRIMARY KEY (attribute_id, locale)
);

-- Attribute options (predefined values for select / multi_select types)
CREATE TABLE attribute_options (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    attribute_id UUID        NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
    position     INTEGER     NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Attribute option translations
CREATE TABLE attribute_option_translations (
    option_id UUID        NOT NULL REFERENCES attribute_options(id) ON DELETE CASCADE,
    locale    VARCHAR(10) NOT NULL,
    name      VARCHAR(255) NOT NULL,
    PRIMARY KEY (option_id, locale)
);

-- Product attribute values (one value per product + attribute)
CREATE TABLE product_attribute_values (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id    UUID          NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_id  UUID          NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
    value_text    TEXT,
    value_numeric NUMERIC(15,4),
    value_boolean BOOLEAN,
    option_id     UUID          REFERENCES attribute_options(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, attribute_id)
);

-- Product attribute multi-select junction
CREATE TABLE product_attribute_value_options (
    value_id  UUID NOT NULL REFERENCES product_attribute_values(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES attribute_options(id) ON DELETE CASCADE,
    PRIMARY KEY (value_id, option_id)
);

-- Variant attribute values (optional override per variant + attribute)
CREATE TABLE variant_attribute_values (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    variant_id    UUID          NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    attribute_id  UUID          NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
    value_text    TEXT,
    value_numeric NUMERIC(15,4),
    value_boolean BOOLEAN,
    option_id     UUID          REFERENCES attribute_options(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (variant_id, attribute_id)
);

-- Variant attribute multi-select junction
CREATE TABLE variant_attribute_value_options (
    value_id  UUID NOT NULL REFERENCES variant_attribute_values(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES attribute_options(id) ON DELETE CASCADE,
    PRIMARY KEY (value_id, option_id)
);

-- Indexes for filtering and lookup
CREATE INDEX idx_product_attribute_values_product   ON product_attribute_values(product_id);
CREATE INDEX idx_product_attribute_values_attribute  ON product_attribute_values(attribute_id);
CREATE INDEX idx_product_attribute_values_option     ON product_attribute_values(option_id) WHERE option_id IS NOT NULL;
CREATE INDEX idx_variant_attribute_values_variant    ON variant_attribute_values(variant_id);
CREATE INDEX idx_variant_attribute_values_attribute  ON variant_attribute_values(attribute_id);
CREATE INDEX idx_attributes_filterable              ON attributes(filterable) WHERE filterable = true;
