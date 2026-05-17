-- Продукты консоли (per user) и привязка прогона processing к authn.console_users.

ALTER TABLE core.processing_runs
    ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES authn.console_users (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS core.console_products (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES authn.console_users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    repository_url TEXT NOT NULL DEFAULT '',
    repository_ref TEXT NOT NULL DEFAULT 'main',
    repository_subdirectory TEXT NOT NULL DEFAULT '',
    scan_target_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_console_products_name_nonempty CHECK (length(trim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_console_products_owner_created
    ON core.console_products (owner_user_id, created_at DESC);
