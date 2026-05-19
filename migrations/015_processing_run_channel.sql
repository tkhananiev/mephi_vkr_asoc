-- Канал прогона: ручной запуск из консоли vs приём из CI/CD (ingest).

ALTER TABLE core.processing_runs
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'manual';

ALTER TABLE core.processing_runs
    DROP CONSTRAINT IF EXISTS chk_processing_runs_channel;

ALTER TABLE core.processing_runs
    ADD CONSTRAINT chk_processing_runs_channel CHECK (channel IN ('manual', 'ci'));
