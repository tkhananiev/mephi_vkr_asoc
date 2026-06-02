# База данных PostgreSQL

Одна БД `asoc`, все схемы создаются миграциями в `migrations/`. Связи между схемами: `core.vulnerabilities.reference_record_id` → `catalog.reference_records.id`; `integration.ticket_links.group_id` → `core.vulnerability_groups.id`.

---

## Схемы

| Схема | Роль |
|-------|------|
| `catalog` | Нормализованный справочник уязвимостей (NVD, БДУ) + алиасы для матчинга CVE/CWE/BDU. |
| `raw` | Сырые ответы внешних источников (аудит, повторная обработка). |
| `audit` | Журнал прогонов синхронизации справочников. |
| `core` | Находки сканера, нормализованные уязвимости, группы. |
| `integration` | Связь групп уязвимостей с тикетами Jira. |

---

## `catalog`

| Таблица | Что хранится |
|---------|----------------|
| `reference_records` | Одна строка на запись источника: `source_code`, `external_id`, тексты, даты, `metadata_json`. Уникально `(source_code, external_id)`. |
| `reference_aliases` | Ключи поиска: `alias_type` (`CVE`, `CWE`, `BDU`), `alias_value`; FK на `reference_records`. |

Пишет `reference-data-service` при синке; миграции `004`, `005` — демо-данные.
Читает `processing-service` при корреляции (JOIN по алиасам).

---

## `raw`

| Таблица | Назначение |
|---------|------------|
| `reference_raw_items` | Тело ответа источника: `source_code`, `external_id`, `content_type`, `raw_payload`, `raw_hash`. Уникально `(source_code, external_id, raw_hash)`. |

Пишет `reference-data-service` (`ON CONFLICT DO NOTHING`). Читается вручную SQL при необходимости.

---

## `audit`

| Таблица | Назначение |
|---------|------------|
| `reference_sync_runs` | Прогон синка: `source_code`, статус, счётчики `items_*`, `error_message`, время. |

Пишет и читает `reference-data-service` (`GET /api/v1/sync/runs`). Не путать с `core.processing_runs`.

---

## `core`

Миграция `002_processing_schema.sql`.

### `core.processing_runs`

| Поле | Смысл |
|------|--------|
| `source_name` | Сканер (например `semgrep`). |
| `status` | `running` / `completed` / `failed`. |
| `findings_received` | Количество находок в запросе. |
| `findings_processed` | Количество обработанных находок. |
| `vulnerabilities_created` | Новые строки в `core.vulnerabilities`. |
| `groups_updated` | Инкрементируется на каждую обработанную находку. |
| `error_message` | Текст ошибки при `failed`. |
| `owner_user_id` | FK → `authn.console_users.id`; `NULL` — прогон без владельца (API-ключ). |
| `console_product_id` | FK → `core.console_products.id` (миграция `017`). |

### `core.findings`

Сырая находка сканера: `processing_run_id`, `scanner_name`, `asset_id`, `raw_identifier`, `normalized_identifier`, `severity`, `component`, `version`, `payload_json`.

### `core.vulnerabilities`

Нормализованная уязвимость. Дедупликация по сигнатуре `(cve_id, product, version, cwe)` через уникальный индекс с `COALESCE`.

| Поле | Смысл |
|------|--------|
| `cve_id`, `cwe` | Из находки (могут быть пустыми). |
| `product`, `version` | component/version из сканера. |
| `normalized_severity` | Приведённая важность. |
| `correlation_status` | `matched_by_cve`, `matched_by_cwe`, `not_found`, … |
| `reference_record_id` | FK → `catalog.reference_records`; `NULL`, если совпадений нет. |

### `core.finding_vulnerabilities`

M:N: находка ↔ уязвимость. PK: `(finding_id, vulnerability_id)`.

### `core.vulnerability_groups`

Группа уязвимостей. Уникальный `group_key`: для прогонов с `owner_user_id` получает префикс `u:<id>:`, чтобы данные разных пользователей не смешивались.

### `core.console_products` (миграция `014`)

Продукты консоли пользователя: `owner_user_id`, имя, описание, `repository_url`, `repository_ref`, `scan_target_path`. Читает и пишет `api-service`.

### `core.group_vulnerabilities`

M:N: группа ↔ уязвимость.

---

## `integration`

### `integration.ticket_links`

Одна строка на группу (`UNIQUE (group_id)`): `group_id`, `jira_issue_key`, `jira_issue_url`, `sync_status`, `idempotency_key`.

Пишет `jira-integration-service` после успешного POST в Jira.

---

## Демо-данные (миграции 004–005)

- `CVE-2021-44228` (NVD) и `BDU:2021-00001` с общим CVE-алиасом для проверки корреляции.
- `005`: алиас `CWE-78` на ту же NVD-запись — для правил без CVE.

---

## Полезные SQL

```sql
-- Справочник по источникам
SELECT source_code, COUNT(*) FROM catalog.reference_records GROUP BY 1;

-- Последние прогоны синхронизации
SELECT id, source_code, status, items_processed, started_at
FROM audit.reference_sync_runs ORDER BY started_at DESC LIMIT 10;

-- Последние прогоны обработки находок
SELECT id, source_name, owner_user_id, status, findings_received, vulnerabilities_created
FROM core.processing_runs ORDER BY started_at DESC LIMIT 10;

-- Статистика
SELECT (SELECT COUNT(*) FROM core.findings) AS findings,
       (SELECT COUNT(*) FROM core.vulnerabilities) AS vulnerabilities,
       (SELECT COUNT(*) FROM core.vulnerability_groups) AS groups;

-- Тикеты
SELECT group_id, jira_issue_key, sync_status FROM integration.ticket_links ORDER BY id;
```

---

## Миграции (порядок)

| Файл | Содержимое |
|------|------------|
| `001_reference_schema.sql` | `catalog`, `raw`, `audit`. |
| `002_processing_schema.sql` | `core`. |
| `003_integration_schema.sql` | `integration.ticket_links`. |
| `004_demo_seed.sql` | Демо-данные `catalog`. |
| `005_demo_cwe_alias.sql` | Алиас CWE для примера без CVE. |
| `006`–`013` | Синк-курсоры, `authn`, отложенная регистрация, доп. алиасы. |
| `014_console_products_and_run_owner.sql` | `core.console_products`, `owner_user_id`. |
| `017_processing_run_console_product.sql` | `console_product_id` → `core.console_products`. |
