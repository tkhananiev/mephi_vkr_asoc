-- Микросервис авторизации: отдельная схема authn (пользователи консоли + администраторы Asoc-admin).
CREATE SCHEMA IF NOT EXISTS authn;

CREATE TABLE authn.console_users (
    id                 BIGSERIAL PRIMARY KEY,
    email              TEXT NOT NULL,
    password_hash      TEXT NOT NULL,
    disabled           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX console_users_email_lower ON authn.console_users ((lower(email)));

CREATE INDEX idx_console_users_created ON authn.console_users (created_at DESC);

CREATE TABLE authn.admins (
    id            BIGSERIAL PRIMARY KEY,
    login         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX admins_login_unique ON authn.admins ((lower(login)));

-- Однократный перенос из старой app.users (если схема/таблица есть).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'app' AND table_name = 'users'
    ) THEN
        INSERT INTO authn.console_users (email, password_hash, disabled, created_at)
        SELECT u.email, u.password_hash, NOT COALESCE(u.email_verified, false), u.created_at
        FROM app.users AS u
        WHERE NOT EXISTS (
            SELECT 1 FROM authn.console_users c WHERE lower(c.email) = lower(u.email)
        );
    END IF;
END $$;
