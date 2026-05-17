-- Пользователи веб-консоли: регистрация и подтверждение e-mail
CREATE SCHEMA IF NOT EXISTS app;

CREATE TABLE app.users (
    id                 BIGSERIAL PRIMARY KEY,
    email              TEXT NOT NULL,
    password_hash      TEXT NOT NULL,
    email_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    verification_token TEXT,
    verification_sent_at TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_lower_unique ON app.users ((lower(email)));

CREATE INDEX idx_app_users_verification_token
    ON app.users (verification_token)
    WHERE verification_token IS NOT NULL;
