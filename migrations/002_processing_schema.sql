CREATE SCHEMA IF NOT EXISTS core;

CREATE TABLE IF NOT EXISTS core.processing_runs (
    id BIGSERIAL PRIMARY KEY,
    source_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    findings_received INTEGER NOT NULL DEFAULT 0,
    findings_processed INTEGER NOT NULL DEFAULT 0,
    vulnerabilities_created INTEGER NOT NULL DEFAULT 0,
    groups_updated INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS core.findings (
    id BIGSERIAL PRIMARY KEY,
    processing_run_id BIGINT REFERENCES core.processing_runs (id) ON DELETE SET NULL,
    scanner_name TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    raw_identifier TEXT,
    normalized_identifier TEXT,
    severity TEXT,
    component TEXT,
    version TEXT,
    payload_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS core.vulnerabilities (
    id BIGSERIAL PRIMARY KEY,
    cve_id TEXT,
    product TEXT,
    version TEXT,
    cwe TEXT,
    normalized_severity TEXT,
    correlation_status TEXT NOT NULL DEFAULT 'pending',
    reference_record_id BIGINT REFERENCES catalog.reference_records (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_vulnerabilities_signature
    ON core.vulnerabilities (
        COALESCE(cve_id, ''),
        COALESCE(product, ''),
        COALESCE(version, ''),
        COALESCE(cwe, '')
    );

CREATE TABLE IF NOT EXISTS core.finding_vulnerabilities (
    finding_id BIGINT NOT NULL REFERENCES core.findings (id) ON DELETE CASCADE,
    vulnerability_id BIGINT NOT NULL REFERENCES core.vulnerabilities (id) ON DELETE CASCADE,
    PRIMARY KEY (finding_id, vulnerability_id)
);

CREATE TABLE IF NOT EXISTS core.vulnerability_groups (
    id BIGSERIAL PRIMARY KEY,
    group_key TEXT NOT NULL UNIQUE,
    grouping_rule TEXT NOT NULL,
    severity_max TEXT,
    assets_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS core.group_vulnerabilities (
    group_id BIGINT NOT NULL REFERENCES core.vulnerability_groups (id) ON DELETE CASCADE,
    vulnerability_id BIGINT NOT NULL REFERENCES core.vulnerabilities (id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, vulnerability_id)
);

CREATE INDEX IF NOT EXISTS idx_findings_normalized_identifier ON core.findings (normalized_identifier);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_cve_id ON core.vulnerabilities (cve_id);
