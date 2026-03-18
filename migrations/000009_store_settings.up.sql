CREATE TABLE IF NOT EXISTS store_settings (
    singleton         BOOLEAN NOT NULL DEFAULT TRUE UNIQUE CHECK (singleton = TRUE),
    store_name        TEXT NOT NULL DEFAULT 'Stoa',
    store_description TEXT NOT NULL DEFAULT '',
    logo_url          TEXT,
    favicon_url       TEXT,
    contact_email     TEXT,
    currency          TEXT NOT NULL DEFAULT 'EUR',
    country           TEXT,
    timezone          TEXT NOT NULL DEFAULT 'UTC',
    copyright_text    TEXT NOT NULL DEFAULT '',
    maintenance_mode  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default row
INSERT INTO store_settings (singleton) VALUES (TRUE) ON CONFLICT DO NOTHING;
