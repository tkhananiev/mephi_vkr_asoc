# Архитектура и взаимодействие микросервисов (актуально для репозитория)

Документ описывает **реализованный в коде** поток данных и границы сервисов. Планируемые доработки (React, Swagger/OpenAPI, авторизация, полноценный Kafka) перечислены в конце.

## Состав сервисов

| Сервис | Роль |
|--------|------|
| `api-service` | Внешний HTTP API, оркестрация сценария (вызов `semgrep-service` → processing → группы → Jira) |
| `semgrep-service` | Запуск Semgrep по HTTP (`POST /api/v1/scan`), пути к коду — внутри этого контейнера |
| `reference-data-service` | Синхронизация справочников NVD и БДУ ФСТЭК, запись в БД |
| `processing-service` | Приём находок, нормализация, корреляция по CVE, группировка |
| `jira-integration-service` | Создание тикетов, идемпотентность, запись `ticket_links` |
| `jira-mock` | Упрощённая имитация Jira REST для локального стенда |

Инфраструктура: **PostgreSQL** (общая БД для всех схем), **Kafka** (развёрнута в compose; прикладная интеграция в MVP — заглушка).

## Ключевой принцип: корреляция через общую БД

`reference-data-service` и `processing-service` **не вызывают друг друга по HTTP**. Справочники заполняются в таблицах схемы `catalog` (и связанных). `processing-service` при корреляции выполняет **SQL-запросы** к тем же таблицам (поиск записи по CVE / алиасу). Так снижается связность и не дублируется контракт «справочного» REST.

## Semgrep: что именно сканируется

Semgrep установлен **в образе `semgrep-service`**. `api-service` вызывает его по **`APP_SEMGREP_SERVICE_URL`** (в compose — `http://semgrep-service:8085`). В теле сценария указываются **путь к каталогу с исходниками** (`target_path`, должен существовать в контейнере `semgrep-service`) и набор правил (`APP_SEMGREP_CONFIG` в сервисе сканера или поле `semgrep_config` в запросе к `api-service`). Это **SAST по файлам**, а не сканирование работающего веб-приложения по HTTP. Пример с исходниками DVWA: см. `demo/scan-targets/README.md`.

## Основной сценарий (защитный демо)

1. Клиент вызывает `POST /api/v1/scans/semgrep` на `api-service`.
2. `api-service` вызывает `POST /api/v1/scan` на `semgrep-service`, получает JSON Semgrep.
3. `api-service` формирует ingest и вызывает `POST /api/v1/findings/ingest` на `processing-service`.
4. `processing-service` пишет находки и уязвимости, **читает `catalog.reference_*` из PostgreSQL** для сопоставления по CVE, выполняет группировку.
5. `api-service` запрашивает `GET /api/v1/groups` у `processing-service`, затем `POST /api/v1/tickets` у `jira-integration-service`.
6. `jira-integration-service` обращается к Jira (на стенде — к `jira-mock`), сохраняет связь в `integration.ticket_links`.

Все шаги **синхронные HTTP**; очередь между processing и Jira **не используется** в текущей реализации.

## Синхронизация справочников

- Вызываются **явно** через REST `reference-data-service` (`/api/v1/sync/...`), фонового планировщика в процессе MVP нет.
- После успешной синхронизации вызывается заглушка **Kafka publisher** (`noop`: запись в лог), чтобы сохранить место под будущие доменные события.

## Порты (docker-compose)

| Сервис | Порт |
|--------|------|
| api-service | 8080 |
| reference-data-service | 8081 |
| processing-service | 8082 |
| jira-integration-service | 8083 |
| semgrep-service | 8085 |
| jira-mock | 8090 |

## План развития (не реализовано в текущем MVP)

- **Web UI** на React + TypeScript.
- **Swagger / OpenAPI** и публикация спецификации с `api-service`.
- **Авторизация и разграничение доступа** к внешнему API.
- **Реальная публикация/подписка Kafka** (события синхронизации, групп, тикетов) вместо noop и синхронных вызовов там, где нужна асинхронность.

При обновлении главы ВКР имеет смысл опираться на этот файл и помечать в тексте прототип: «текущая реализация» vs «целевой контур».
