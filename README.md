# mephi_vkr_asoc

MVP для управления уязвимостями: микросервисы на Go, общая PostgreSQL, Kafka для асинхронного ingest находок, сценарий сканирования Semgrep и постановки тикетов в Jira (на стенде — мок). Запуск backend: `deploy/docker-compose.yml`. Браузерный клиент — отдельный репозиторий **`mephi_vkr_asoc_front`** (React + Vite), те же REST API, что в `demo/DEMO.md`. Дополнительно сценарий можно проходить через `curl`/Postman и при необходимости смотреть таблицы в PostgreSQL.

Повествование по архитектуре, потокам и Kafka: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Сводные таблицы сервисов и схем данных: [`docs/SERVICES_AND_DATA.md`](docs/SERVICES_AND_DATA.md).

## Текущий состав

- `services/api-service` — внешний API и оркестрация сквозного сценария
- `services/reference-data-service` — синхронизация справочников `NVD/CVE` и `БДУ ФСТЭК`
- `services/processing-service` — нормализация, корреляция и группировка находок
- `services/jira-integration-service` — создание/обновление тикетов и `ticket_link`
- `services/jira-mock` — тестовый Jira-контур для локального smoke-теста
- `services/semgrep-service` — HTTP-обёртка над Semgrep (SAST), отдельный контейнер
- `migrations` — инициализация схем `catalog`, `audit`, `raw`
- веб-интерфейс — в репозитории **`mephi_vkr_asoc_front`** (рядом с этим каталогом)
- `deploy/docker-compose.yml` — локальный контур backend MVP

## Что уже реализовано

- запуск `reference-data-service`
- ручные REST-операции:
  - `POST /api/v1/sync/bdu`
  - `POST /api/v1/sync/nvd`
  - `POST /api/v1/sync/all`
  - `GET /api/v1/sync/runs`
  - `GET /health`
- загрузка `БДУ ФСТЭК` через RSS feed
- загрузка ограниченного набора `NVD` через API 2.0
- сохранение:
  - запусков синхронизации
  - сырых записей
  - нормализованных справочных записей
  - алиасов (`CVE`, `CWE`, и др.)
- прием находок в `processing-service` (HTTP и/или Kafka)
- корреляция со справочником по `CVE` и/или по `CWE`
- группировка в `vulnerability_groups`
- запуск `Semgrep` через `api-service`
- создание тикета через `jira-integration-service`
- тестовый Jira через `jira-mock`
- демонстрационный seed для устойчивого корреляционного сценария

## Сквозной демо-сценарий

```text
Клиент (HTTP)
  -> api-service (POST /api/v1/scans/semgrep; по умолчанию цель WebGoat + p/java через APP_DEFAULT_*)
  -> semgrep-service (POST /api/v1/scan; Semgrep в отдельном контейнере)
  -> api-service -> Kafka (asoc.findings.ingest) -> processing-service -> Kafka (asoc.findings.ingest.result); корреляция по CVE/CWE через PostgreSQL / catalog.* [или HTTP ingest без Kafka, если APP_KAFKA_BROKERS не задан]
  -> api-service -> GET groups -> POST /api/v1/tickets
  -> jira-integration-service -> jira-mock (на стенде)
```

Kafka в compose используется для **ingest находок** (`api-service` → топик → `processing-service` → топик ответа); подробности и noop для reference-data — в `docs/ARCHITECTURE.md`.

## Быстрый старт

Перед первым сканированием по умолчанию (WebGoat) **один раз** на хосте:

```bash
./demo/scan-targets/clone-webgoat.sh
```

Поднять контейнеры:

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

После запуска доступны:

- `api-service` — `http://localhost:8080`
- `reference-data-service` — `http://localhost:8081`
- `processing-service` — `http://localhost:8082`
- `jira-integration-service` — `http://localhost:8083`
- `jira-mock` — `http://localhost:8090`
- `semgrep-service` — `http://localhost:8085`

## Semgrep и цели сканирования

Semgrep в контейнере **`semgrep-service`** читает **файлы** по `target_path` (путь **внутри этого контейнера**). Каталог `demo/` смонтирован как `/app/demo/...`.

По умолчанию в compose для **`api-service`** заданы **`APP_DEFAULT_SCAN_TARGET_PATH=/app/demo/scan-targets/WebGoat`** и **`APP_DEFAULT_SEMGREP_CONFIG=p/java`**: перед первым прогоном выполните **`demo/scan-targets/clone-webgoat.sh`**. Альтернативы (DVWA, учебный `vulnerable-app`) — в `demo/scan-targets/README.md`.

## Demo-артефакты

- инструкция: `demo/DEMO.md`
- curl-сценарий: `demo/curl-demo.sh`
- примеры HTTP-запросов (коллекция для импорта в средства тестирования API): `demo/http-collection/MEPHI_VKR_ASOC_http_collection.json`
