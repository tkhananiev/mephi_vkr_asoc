CREATE TABLE IF NOT EXISTS authn.pending_registrations (
    email             TEXT PRIMARY KEY,
    username          TEXT NOT NULL,
    first_name        TEXT NOT NULL,
    last_name         TEXT NOT NULL,
    patronymic        TEXT NOT NULL DEFAULT '',
    password_hash     TEXT NOT NULL,
    verification_code TEXT NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pending_registrations_expires
    ON authn.pending_registrations (expires_at);
