# Demo Guide

`reference-data-service` — порт **8081** (`deploy/docker-compose.yml`). Вместо `localhost` подставь хост стенда, если вызываешь удалённо.

---

## 1. Синхронизация со справочниками

Инкрементально по RSS (быстро):

```bash
curl -X POST "http://localhost:8081/api/v1/sync/bdu"
```

Полный импорт БДУ из `vulxml.zip` + `vullist.xlsx` (долго; по умолчанию скачивание с сайта ФСТЭК, либо локальные пути через `APP_BDU_VULXML_ZIP_PATH` / `APP_BDU_VULLIST_XLSX_PATH`, см. `demo/bdu/README.md` и `docs/ENVIRONMENT.md`):

```bash
curl --max-time 0 -X POST "http://localhost:8081/api/v1/sync/bdu/bulk"
```

NVD (пример — одна CVE; полный синк без `cve_id`):

```bash
curl --max-time 0 -X POST "http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228"
```

### Схемы и таблицы (reference-data → PostgreSQL)

Миграция: `migrations/001_reference_schema.sql`. Справочники разнесены по схемам:

| Схема | Таблица | Назначение |
|-------|---------|------------|
| **catalog** | `reference_records` | Нормализованная запись угрозы/уязвимости: `source_code`, `external_id`, заголовок, описание, даты, `metadata_json`. Уникальность `(source_code, external_id)`. |
| **catalog** | `reference_aliases` | Алиасы для сопоставления с находками сканера (`alias_type`, `alias_value`), FK на `reference_records.id`. Типы вроде `CVE`, `BDU`, `CWE` (что именно зависит от источника). |
| **raw** | `reference_raw_items` | Сырой ответ источника (тело, `content_type`, хеш) для аудита и повторной обработки. |
| **audit** | `reference_sync_runs` | Журнал прогонов синка: `source_code`, статус, счётчики `items_*`, `error_message`. |

Значения **`source_code`** в коде сервиса:

- **`nvd`** — записи из NVD API 2.0; по сути каталог **CVE** (одна строка `reference_records` ≈ один CVE).
- **`bdu_fstec`** — записи из ленты БДУ ФСТЭК.

### SQL: сколько записей CVE (NVD) и БДУ

Подключение к БД из compose: хост `localhost`, порт `5432`, БД `asoc`, пользователь `asoc` / пароль `asoc` (см. `deploy/docker-compose.yml`).

```sql
-- CVE-каталог (источник NVD)
SELECT COUNT(*) AS cve_records_nvd
FROM catalog.reference_records
WHERE source_code = 'nvd';

-- Записи БДУ ФСТЭК
SELECT COUNT(*) AS bdu_records
FROM catalog.reference_records
WHERE source_code = 'bdu_fstec';

-- Сводка по всем источникам в справочнике
SELECT source_code, COUNT(*) AS cnt
FROM catalog.reference_records
GROUP BY source_code
ORDER BY source_code;
```

Дополнительно — сколько справочных записей имеют алиас `CVE` (в т.ч. БДУ, если в тексте фигурирует CVE):

```sql
SELECT COUNT(DISTINCT reference_record_id) AS records_with_cve_alias
FROM catalog.reference_aliases
WHERE alias_type = 'CVE';
```

---

## 2. Запуск Semgrep (находки → обработка)

`api-service` — порт **8080**. Перед первым прогоном с дефолтной целью: `./demo/scan-targets/clone-webgoat.sh`.

```bash
curl -X POST "http://localhost:8080/api/v1/scans" \
  -H "Content-Type: application/json" \
  -d '{"scanner_id":"semgrep","scanner_name":"semgrep"}'
```
