-- Список веток SCM для продукта консоли (сканирование по выбранной ветке из UI).

ALTER TABLE core.console_products
    ADD COLUMN IF NOT EXISTS repository_branch_refs JSONB NOT NULL DEFAULT '["main"]'::jsonb;

UPDATE core.console_products p
SET repository_branch_refs = to_jsonb(ARRAY[CASE WHEN length(trim(COALESCE(p.repository_ref, ''))) = 0
                                                THEN 'main'
                                                ELSE trim(p.repository_ref) END]::text[]);
