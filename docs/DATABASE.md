# База данных PostgreSQL (схемы и таблицы)

Одна БД **`asoc`**, все схемы создаются миграциями в `migrations/`. Ниже — **что где лежит** и зачем, в логическом порядке потока данных: справочники → сырьё/аудит → обработка находок → интеграция с Jira.

---

## Обзор схем

| Схема | Роль |
|-------|------|
| **`catalog`** | Нормализованный справочник угроз/уязвимостей (NVD, БДУ, …) + алиасы для матчинга CVE/CWE/BDU. |
| **`raw`** | Сырые ответы внешних источников справочников (аудит, повторная обработка). |
| **`audit`** | Журнал **прогонов синхронизации** справочников (не путать с прогонами processing). |
| **`core`** | Находки сканера, нормализованные уязвимости, группы — **операционный слой** после ingest. |
| **`integration`** | Связь групп уязвимостей с тикетами Jira (мок или боевая Jira). |

Связь «наружу»: `core.vulnerabilities.reference_record_id` → `catalog.reference_records.id`.  
`integration.ticket_links.group_id` → `core.vulnerability_groups.id`.

---

## `catalog` — справочник (кратко; детально вы уже разобрали)

| Таблица | Что хранится |
|---------|----------------|
| **`reference_records`** | Одна строка = одна запись из источника: `source_code` (`nvd`, `bdu_fstec`, …), `external_id`, тексты, даты, `metadata_json`. Уникально `(source_code, external_id)`. |
| **`reference_aliases`** | Ключи поиска: `alias_type` (`CVE`, `CWE`, `BDU`, …), `alias_value`; FK на `reference_records`. |

**Пишет:** `reference-data-service` при синке; миграции **`004_demo_seed.sql`**, **`005_demo_cwe_alias.sql`** (демо-записи и алиас CWE для сценария без CVE в правиле).

**Читает:** `processing-service` при корреляции (JOIN по алиасам).

---

## `raw` — сырой контент справочников

| Таблица | Назначение |
|---------|------------|
| **`reference_raw_items`** | Тело ответа источника: `source_code`, `external_id`, `source_url`, `content_type`, `raw_payload`, `raw_hash` (SHA-256 от payload). Уникально `(source_code, external_id, raw_hash)` — одна и та же версия сырья не дублируется. |

**Пишет:** `reference-data-service` при каждом upsert записи справочника (вставка с `ON CONFLICT DO NOTHING`).

**Читает:** в коде MVP нет; при необходимости — вручную SQL / будущий реплей.

---

## `audit` — журнал синхронизации справочников

| Таблица | Назначение |
|---------|------------|
| **`reference_sync_runs`** | Один **прогон** синка БДУ/NVD: `source_code`, `status` (`running` / успех / ошибка), `started_at`, `finished_at`, счётчики `items_discovered`, `items_processed`, `items_inserted`, `items_updated`, `error_message`. |

**Пишет / читает:** `reference-data-service` (старт/финиш прогона; `GET /api/v1/sync/runs`).

Не путать с **`core.processing_runs`** — это другой процесс (обработка находок сканера).

---

## `core` — обработка находок и агрегация

Миграция **`002_processing_schema.sql`**.

### `core.processing_runs`

Один **батч ingest** в `processing-service`: от сканера пришло N находок, сервис обрабатывает их под одним `run_id`.

| Поле | Смысл |
|------|--------|
| `source_name` | Имя источника (например `semgrep`). |
| `status` | `running` / `completed` / `failed`. |
| `findings_received` | Сколько находок в запросе. |
| `findings_processed` | Сколько реально дошло до конца цикла. |
| `vulnerabilities_created` | Сколько **новых** строк в `core.vulnerabilities` (повторы по сигнатуре не считаются как созданные). |
| `groups_updated` | В коде инкрементируется на каждую обработанную находку (не «число уникальных групп»). |
| `error_message` | Текст ошибки при `failed`. |
| **`owner_user_id`** | _(миграция `014`.)_ Привязка прогона к **`authn.console_users.id`**; `NULL` — прогон без владельца (API-ключ, прямой HTTP-ingest без пользователя). Используется для фильтров отчёта и групп по консольному пользователю. |
| **`console_product_id`** | _(миграция `017`.)_ FK → **`core.console_products.id`**; какому **проекту консоли** принадлежит прогон. Фильтр **`?console_product_id=`** на **`GET /api/v1/groups`** и **`GET /api/v1/report/vulnerabilities`** (через api-service с JWT пользователя). |

**Пишет / читает:** `processing-service`.

---

### `core.findings`

**Сырая находка** сканера: одна строка на один элемент из ingest.

| Поле | Смысл |
|------|--------|
| `processing_run_id` | Прогон (может стать `NULL` при каскаде — в схеме `ON DELETE SET NULL`). |
| `scanner_name` | Сканер. |
| `asset_id` | Укороченный идентификатор актива (в сценарии Semgrep часто имя файла). |
| `raw_identifier` / `normalized_identifier` | Идентификатор правила/находки; нормализованный — для поиска (например CVE). |
| `severity`, `component`, `version` | Нормализованная важность, путь/компонент, версия. |
| `payload_json` | Вложенные `metadata` и `raw_payload` из API. |

**Пишет:** `processing-service` при разборе ingest.  
**Читает:** в коде MVP явных API нет; отчёты — SQL.

---

### `core.vulnerabilities`

**Нормализованная уязвимость** в терминах продукта: схлопывание по сигнатуре **CVE + продукт + версия + CWE** (через уникальный индекс с `COALESCE` для NULL).

| Поле | Смысл |
|------|--------|
| `cve_id`, `cwe` | Из находки (могут быть пустыми). |
| `product`, `version` | Обычно component/version из сканера. |
| `normalized_severity` | Приведённая важность. |
| `correlation_status` | Как сматчилось со справочником: `matched_by_cve`, `matched_by_cwe`, `not_found`, … |
| `reference_record_id` | FK на **`catalog.reference_records`**, если нашлась запись по алиасу; иначе `NULL`. |

**Пишет:** `processing-service`.  
**Читает:** косвенно через группы и связи; прямые отчёты SQL.

---

### `core.finding_vulnerabilities`

Связь **M:N**: одна находка ↔ одна нормализованная уязвимость (факт «эта строка finding относится к этой vulnerability»).

**Первичный ключ:** `(finding_id, vulnerability_id)`.

---

### `core.vulnerability_groups`

**Группа** для агрегирования (и дальше — тикет): уникальный **`group_key`** по-прежнему глобален по таблице; для прогона с **`owner_user_id`** в ключ добавляется префикс вида **`u:<id>:`**, чтобы находки разных пользователей не смешивались в одной строке группы. Базовая часть без префикса — как раньше: `CVE::CWE::component::version` (после нормализации полей).

**Пишет:** `processing-service` (`UPSERT` по `group_key`).  
**Читает:** `processing-service` (`GET /api/v1/groups`); браузер идёт через **`api-service`**, который проксирует запрос и ограничивает выборку по пользователю JWT **`role=user`**.

---

### `core.console_products` (миграция `014`)

**Продукты консоли** пользователя (`owner_user_id` → **`authn.console_users`**): имя, описание, поля SCM (`repository_url`, `repository_ref`, `repository_subdirectory`), `scan_target_path` (демо-путь без SCM).

**Пишет / читает:** **`api-service`** (`GET` / `POST /api/v1/console/products`; только JWT консольного пользователя).

---

### `core.group_vulnerabilities`

Связь **M:N**: группа ↔ уязвимость (какие `vulnerabilities` вошли в группу).

---

## `integration` — тикеты Jira

### `integration.ticket_links`

Одна строка на **одну группу** уязвимостей (`UNIQUE (group_id)`).

| Поле | Смысл |
|------|--------|
| `group_id` | FK → `core.vulnerability_groups.id`. |
| `jira_issue_key` | Ключ задачи (`ASOC-42`). |
| `jira_issue_url` | URL из ответа Jira/mock (желательно публичный базовый URL, см. `APP_JIRA_PUBLIC_BASE_URL`). |
| `sync_status` | Например `synced`. |
| `idempotency_key` | Уникальный ключ повторного запроса (`UNIQUE`). |

**Пишет:** `jira-integration-service` после успешного `POST` в Jira.  
**Читает:** в MVP нет REST-листа; смотреть SQL или `/console` у **jira-mock** (память процесса, не эта таблица).

---

## Демо-данные в `catalog` (миграции 004–005)

- Две записи справочника: NVD `CVE-2021-44228`, БДУ `BDU:2021-00001` с общим CVE-алиасом для демо корреляции.
- **`005`:** алиас `CWE-78` на ту же NVD-запись — для правил без CVE (учебный сценарий).

---

## Полезные SQL для осмотра

```sql
-- Сводка по справочнику
SELECT source_code, COUNT(*) FROM catalog.reference_records GROUP BY 1;

-- Последние прогоны справочников
SELECT id, source_code, status, items_processed, started_at
FROM audit.reference_sync_runs ORDER BY started_at DESC LIMIT 10;

-- Последние прогоны обработки находок (с владельцем консоли, если задан миграцией 014+)
SELECT id, source_name, owner_user_id, status, findings_received, findings_processed, vulnerabilities_created
FROM core.processing_runs ORDER BY started_at DESC LIMIT 10;

-- Продукты консоли по пользователям (после миграции 014)
SELECT id, owner_user_id, name, left(repository_url, 60) AS repo
FROM core.console_products ORDER BY created_at DESC LIMIT 20;

-- Сколько сырых находок и уязвимостей
SELECT (SELECT COUNT(*) FROM core.findings) AS findings,
       (SELECT COUNT(*) FROM core.vulnerabilities) AS vulnerabilities,
       (SELECT COUNT(*) FROM core.vulnerability_groups) AS groups;

-- Тикеты в БД
SELECT group_id, jira_issue_key, jira_issue_url, sync_status
FROM integration.ticket_links ORDER BY id;
```

---

## Файлы миграций (порядок применения)

| Файл | Содержимое |
|------|------------|
| `001_reference_schema.sql` | `catalog`, `raw`, `audit` + таблицы справочника. |
| `002_processing_schema.sql` | `core` + все таблицы обработки. |
| `003_integration_schema.sql` | `integration.ticket_links`. |
| `004_demo_seed.sql` | Сиды `catalog` для демо. |
| `005_demo_cwe_alias.sql` | Доп. алиас CWE для демо. |
| `006` … `013` | См. каталог `migrations/` и `deploy/k8s/kustomization.yaml` (синк курсоров, `authn`, отложенная регистрация, демо-алиасы и т.д.). |
| **`014_console_products_and_run_owner.sql`** | `core.console_products`, колонка **`core.processing_runs.owner_user_id`**. |
| **`017_processing_run_console_product.sql`** | Колонка **`core.processing_runs.console_product_id`** → `core.console_products`. | [`docs/SERVICES_AND_DATA.md`](SERVICES_AND_DATA.md).
