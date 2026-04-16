CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS raw;

CREATE TABLE IF NOT EXISTS audit.reference_sync_runs (
    id BIGSERIAL PRIMARY KEY,
    source_code TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    items_discovered INTEGER NOT NULL DEFAULT 0,
    items_processed INTEGER NOT NULL DEFAULT 0,
    items_inserted INTEGER NOT NULL DEFAULT 0,
    items_updated INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS raw.reference_raw_items (
    id BIGSERIAL PRIMARY KEY,
    source_code TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT,
    content_type TEXT NOT NULL,
    raw_payload TEXT NOT NULL,
    raw_hash TEXT NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_code, external_id, raw_hash)
);

CREATE TABLE IF NOT EXISTS catalog.reference_records (
    id BIGSERIAL PRIMARY KEY,
    source_code TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    severity TEXT,
    published_at TIMESTAMPTZ,
    modified_at TIMESTAMPTZ,
    source_url TEXT,
    status TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_code, external_id)
);

CREATE TABLE IF NOT EXISTS catalog.reference_aliases (
    id BIGSERIAL PRIMARY KEY,
    reference_record_id BIGINT NOT NULL REFERENCES catalog.reference_records (id) ON DELETE CASCADE,
    alias_type TEXT NOT NULL,
    alias_value TEXT NOT NULL,
    UNIQUE (reference_record_id, alias_type, alias_value)
);

CREATE INDEX IF NOT EXISTS idx_reference_records_source_code ON catalog.reference_records (source_code);
CREATE INDEX IF NOT EXISTS idx_reference_aliases_alias_value ON catalog.reference_aliases (alias_value);
