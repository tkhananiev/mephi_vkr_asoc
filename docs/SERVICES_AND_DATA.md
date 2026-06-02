# Сервисы, данные и Kafka

Сводные таблицы для `mephi_vkr_asoc`. Архитектура и диаграммы — [`ARCHITECTURE.md`](ARCHITECTURE.md). Поля таблиц — [`DATABASE.md`](DATABASE.md).

---

## Сервисы

| Сервис | Порт | Вход | Исходящие вызовы | Результат |
|--------|------|------|------------------|-----------|
| **reference-data-service** | 8081 | `POST /api/v1/sync/*`, планировщик | RSS БДУ, `vulxml.zip`/`vullist.xlsx`, NVD API 2.0 | PostgreSQL: `audit.*`, `raw.*`, `catalog.*` |
| **processing-service** | 8082 | `GET /api/v1/groups`, `GET /api/v1/report/*`; опц. `POST /api/v1/findings/ingest` | PostgreSQL, Kafka | PostgreSQL: `core.*` |
| **api-service** | 8080 | `POST /api/v1/scans`, `GET /api/v1/integrations`, продукты консоли, группы/отчёт | executor-сервисы, Kafka или HTTP на processing, jira-integration-service | PostgreSQL: `core.console_products` |
| **semgrep-service** | 8085 | `POST /api/v1/scan` | Semgrep внутри контейнера | JSON `results` |
| **gitleaks-service** | 8086 | `POST /api/v1/scan` | Gitleaks | JSON находок |
| **trivy-sca-service** | 8088 | `POST /api/v1/scan` | `trivy fs` | JSON Trivy `Results` |
| **zap-dast-service** | 8089 | `POST /api/v1/scan` | OWASP ZAP baseline | JSON `{findings:[…]}` |
| **jira-integration-service** | 8083 | `POST /api/v1/tickets` | Jira REST | PostgreSQL: `integration.ticket_links` |
| **jira-mock** | 8090 | Имитация Jira | — | Память процесса |

---

## PostgreSQL: схемы и таблицы

### `catalog` — справочник

Файл: `001_reference_schema.sql`.

| Таблица | Назначение |
|---------|------------|
| `reference_records` | Запись источника: `(source_code, external_id)`, тексты, `metadata_json`. |
| `reference_aliases` | Ключи корреляции: `alias_type`/`alias_value` → `reference_records`. |

### `raw` — сырьё

| Таблица | Назначение |
|---------|------------|
| `reference_raw_items` | Сырой payload, `content_type`, SHA-256 хеш. |

### `audit` — журнал синхронизации

| Таблица | Назначение |
|---------|------------|
| `reference_sync_runs` | Прогон синка: статус, счётчики, время. |

### `core` — обработка находок

Файл: `002_processing_schema.sql`.

| Таблица | Назначение |
|---------|------------|
| `processing_runs` | Батч ingest: статус, счётчики, `owner_user_id`, `console_product_id`. |
| `findings` | Сырая находка сканера. |
| `vulnerabilities` | Нормализованная уязвимость с дедупликацией по `(cve_id, product, version, cwe)`. |
| `finding_vulnerabilities` | M:N находка ↔ уязвимость. |
| `vulnerability_groups` | Группа по `group_key` (с префиксом `u:<id>:` для консольных прогонов). |
| `group_vulnerabilities` | M:N группа ↔ уязвимость. |
| `console_products` | Продукты консоли пользователя (миграция `014`). |

### `integration` — Jira

| Таблица | Назначение |
|---------|------------|
| `ticket_links` | `group_id` → `jira_issue_key`, `idempotency_key`. |

---

## Kafka

`APP_KAFKA_BROKERS` — адрес брокера (в compose: `kafka:9092`).

| Топик | Назначение |
|-------|------------|
| `asoc.findings.ingest` | Пакет находок: `{correlation_id, ingest:{scanner_name, findings[], owner_user_id?}}`. |
| `asoc.findings.ingest.result` | Результат: `{correlation_id, processing:{...}}` или `{error}`. |

| Сервис | Роль |
|--------|------|
| api-service | Producer ingest, consumer result. Без брокера — HTTP POST на processing. |
| processing-service | Consumer ingest, producer result. HTTP ingest через `APP_HTTP_FINDINGS_INGEST`. |
| reference-data-service | Подключается, но публикация событий заглушена (noop). |

---

## Внешние системы

| Система | Кто обращается | Зачем |
|---------|----------------|-------|
| БДУ ФСТЭК (HTTPS) | reference-data-service | RSS/XML справочника |
| NVD (HTTPS) | reference-data-service | REST JSON CVE 2.0 |
| Jira | jira-integration-service | Создание задачи |
