-- Курсоры синхронизации внешних каталогов (NVD: полный прогон vs инкремент по lastMod).
CREATE TABLE IF NOT EXISTS audit.reference_sync_cursor (
    source_code TEXT PRIMARY KEY,
    -- Конец последнего успешного окна lastMod для NVD (UTC). Следующий инкремент: от (nvd_last_mod_end - overlap) до now.
    nvd_last_mod_end TIMESTAMPTZ,
    -- После первого полного прохода NVD без курсора — true; дальше только инкремент (если включено в сервисе).
    nvd_full_sync_completed BOOLEAN NOT NULL DEFAULT FALSE
);
