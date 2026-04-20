# Сервисы, данные и Kafka

Сводные таблицы для `mephi_vkr_aspm`: вызовы между сервисами, схемы PostgreSQL и топики Kafka. За **связным текстом** про микросервисы, диаграммами и сценарием см. [`ARCHITECTURE.md`](ARCHITECTURE.md). Одна БД PostgreSQL (`aspm`), миграции в `migrations/`. Детализация полей таблиц: [`DATABASE.md`](DATABASE.md).

---

## 1. Сервисы

| Сервис | Порт | Вход (кто дергает) | Исходящие вызовы | Куда пишет результат |
|--------|------|-------------------|------------------|----------------------|
| **reference-data-service** | 8081 | HTTP: `POST /api/v1/sync/*`, `GET /api/v1/sync/runs`, планировщик синка (если включён) | HTTPS: фид БДУ ФСТЭК (`APP_BDU_FEED_URL`), NVD API 2.0 (`APP_NVD_API_BASE_URL`) | PostgreSQL: `audit.*`, `raw.*`, `catalog.*`. Kafka: **нет** (noop-publisher, только лог) |
| **processing-service** | 8082 | HTTP: `POST /api/v1/findings/ingest`, `GET /api/v1/groups`. Kafka: consumer топика ingest (см. §3) | Нет внешних HTTP; только PostgreSQL | PostgreSQL: `core.*`. Kafka: **producer** в топик result |
| **api-service** | 8080 | HTTP: `POST /api/v1/scans/semgrep`, `GET /health` | HTTP: `semgrep-service` (`/api/v1/scan`), `processing-service` **или** Kafka (см. §3), `jira-integration-service` (`/api/v1/tickets`) | Нигде в БД не пишет. Ответ клиенту: JSON «паспорт» (`findings`, `processing`, `groups`, `tickets`) |
| **semgrep-service** | 8085 | HTTP: `POST /api/v1/scan` | Запуск Semgrep по файлам по `target_path` **внутри контейнера** | Не использует БД; возвращает JSON с находками |
| **jira-integration-service** | 8083 | HTTP: `POST /api/v1/tickets`, `GET /health` | HTTP: Jira REST `POST /rest/api/2/issue` на `APP_JIRA_BASE_URL` | PostgreSQL: `integration.ticket_links`. Ответ: ключ/URL задачи |
| **jira-mock** | 8090 | HTTP: имитация Jira (`/rest/api/2/issue`) | Нет | Только счётчик в памяти; БД не трогает |

**Типовой сценарий:** клиент → `api-service` → Semgrep → (Kafka или HTTP) → `processing-service` → БД → `api-service` читает группы → `jira-integration-service` → `jira-mock` + запись `integration.ticket_links`.

---

## 2. PostgreSQL: схемы и таблицы

### 2.1. `catalog` — нормализованный справочник угроз/уязвимостей

Файл: `001_reference_schema.sql` (+ сиды `004_demo_seed.sql`, `005_demo_cwe_alias.sql`).

| Таблица | Назначение |
|---------|------------|
| `reference_records` | Одна строка на объект источника: `source_code` (`nvd`, `bdu_fstec`, …), `external_id`, тексты, даты, `metadata_json`. Уникально `(source_code, external_id)`. |
| `reference_aliases` | Ключи корреляции: `alias_type` / `alias_value` (`CVE`, `BDU`, `CWE`, …), FK → `reference_records.id`. |

| Кто пишет | Кто читает |
|-----------|------------|
| **Пишет:** `reference-data-service` (upsert при синке: сначала запись, затем удаление старых алиасов записи + вставка новых). **Пишет (разово):** миграции 004, 005. | **Читает:** `processing-service` (JOIN записей и алиасов по `CVE` / `CWE` для `reference_record_id`). **Читает:** `reference-data-service` косвенно через те же таблицы при синке (обновление). |

### 2.2. `raw` — сырьё справочников

| Таблица | Назначение |
|---------|------------|
| `reference_raw_items` | Сырой payload, `content_type`, хеш; уникально `(source_code, external_id, raw_hash)`. |

| Кто пишет | Кто читает |
|-----------|------------|
| **Пишет:** `reference-data-service` при синке (`ON CONFLICT DO NOTHING`). | В коде MVP **нет** читателей; для аудита/отладки вручную. |

### 2.3. `audit` — журнал синхронизации справочников

| Таблица | Назначение |
|---------|------------|
| `reference_sync_runs` | Прогон синка: `source_code`, статус, счётчики `items_*`, `error_message`, время. |

| Кто пишет | Кто читает |
|-----------|------------|
| **Пишет:** `reference-data-service` (старт/финиш прогона). | **Читает:** `reference-data-service` — `GET /api/v1/sync/runs`. |

### 2.4. `core` — обработка находок и корреляция

Файл: `002_processing_schema.sql`.

| Таблица | Назначение |
|---------|------------|
| `processing_runs` | Один прогон ingest: статус, сколько находок обработано, создано уязвимостей, групп, ошибка. |
| `findings` | Сырая находка сканера (run, идентификаторы, severity, component, version, `payload_json`). |
| `vulnerabilities` | Нормализованная уязвимость; `correlation_status`; FK `reference_record_id` → `catalog.reference_records`. Дедуп по сигнатуре `(cve_id, product, version, cwe)` (уникальный индекс с `COALESCE`). |
| `finding_vulnerabilities` | M:N находка ↔ уязвимость. |
| `vulnerability_groups` | Агрегат по `group_key` (уникален), `grouping_rule`, `severity_max`, `assets_count`. |
| `group_vulnerabilities` | M:N группа ↔ уязвимость. |

| Кто пишет | Кто читает |
|-----------|------------|
| **Пишет:** только `processing-service` (полный цикл `ProcessFindings`). | **Читает:** `processing-service` — `GET /api/v1/groups` (`vulnerability_groups`). **Читает:** `jira-integration-service` **не читает** таблицы `core` напрямую; сведения о группе приходят в теле запроса от `api-service`. Вручную/SQL — любой клиент БД. |

### 2.5. `integration` — связь с Jira

Файл: `003_integration_schema.sql`.

| Таблица | Назначение |
|---------|------------|
| `ticket_links` | Связь `group_id` → `jira_issue_key`, URL, `sync_status`, `idempotency_key`; одна строка на `group_id`. |

| Кто пишет | Кто читает |
|-----------|------------|
| **Пишет:** `jira-integration-service` (`UPSERT` по `group_id`). | В коде MVP отдельного list API нет; просмотр через SQL или расширение сервиса. |

---

## 3. Kafka

Брокер задаётся `APP_KAFKA_BROKERS` (в compose: `kafka:9092`). Топики по умолчанию:

| Топик | Назначение |
|-------|------------|
| `aspm.findings.ingest` | Пакет находок: JSON `{"correlation_id","ingest":{scanner_name, findings[]}}`. |
| `aspm.findings.ingest.result` | Результат: `{"correlation_id","processing":{...}}` или `"error"`. |

| Участник | Роль |
|----------|------|
| **api-service** | При ненулевом `APP_KAFKA_BROKERS`: **producer** ingest, **consumer** result (ожидание ответа с тем же `correlation_id`). При пустом брокере — прямой HTTP `POST` на `processing-service`. |
| **processing-service** | При ненулевом `APP_KAFKA_BROKERS`: **consumer** ingest (group `processing-findings-ingest`), **producer** result. HTTP ingest обрабатывается параллельно тем же `ProcessFindings`. |
| **reference-data-service** | Подключается к брокеру в конфиге, но публикация событий **заглушена** (noop). |

Создание топиков: `EnsureTopics` в `api-service` и `processing-service` при старте (1 партиция, RF=1).

---

## 4. Внешние системы (вне контура)

| Система | Кто обращается | Зачем |
|---------|----------------|-------|
| БДУ ФСТЭК (HTTPS) | reference-data-service | RSS/XML справочника |
| NVD (HTTPS) | reference-data-service | REST JSON CVE 2.0 |
| Jira (HTTP у мока) | jira-integration-service | Создание задачи |
