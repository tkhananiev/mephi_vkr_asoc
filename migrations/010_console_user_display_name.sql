ALTER TABLE authn.console_users ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';

UPDATE authn.console_users
SET display_name = trim(split_part(lower(email), '@', 1))
WHERE trim(display_name) = '';
