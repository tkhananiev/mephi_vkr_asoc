-- Расширение профиля пользователя консоли: ФИО + логин (username).
ALTER TABLE authn.console_users
  ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS patronymic TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS console_users_username_lower_unique
  ON authn.console_users ((lower(trim(username))))
  WHERE length(trim(username)) > 0;
