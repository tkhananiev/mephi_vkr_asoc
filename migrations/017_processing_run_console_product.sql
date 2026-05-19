-- Прогон processing → продукт консоли (фильтр отчёта / групп по проекту).

ALTER TABLE core.processing_runs
    ADD COLUMN IF NOT EXISTS console_product_id BIGINT REFERENCES core.console_products (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_processing_runs_console_product
    ON core.processing_runs (console_product_id)
    WHERE console_product_id IS NOT NULL;
