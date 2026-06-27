-- Create a simple items table to demonstrate the migrator library.
-- This is a neutral demo migration with no auth or domain-specific logic.
CREATE TABLE IF NOT EXISTS items (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
