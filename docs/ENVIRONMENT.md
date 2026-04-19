# Переменные окружения (сводка)

Значения для docker-compose: **`deploy/docker-compose.yml`**.  
Если переменной нет в compose, при локальном запуске действует **дефолт из `config.go`** указанного сервиса.

---

## Инфраструктура (не `APP_*`)

| Сервис | Переменная | Значение в compose |
|--------|------------|-------------------|
| `postgres` | `POSTGRES_DB` | `aspm` |
| | `POSTGRES_USER` | `aspm` |
| | `POSTGRES_PASSWORD` | `aspm` |
| `kafka` | см. `deploy/docker-compose.yml` | настройки брокера KRaft |

---

## `api-service`

Файл дефолтов: `services/api-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8080` | `8080` |
| `APP_PROCESSING_SERVICE_URL` | `http://processing-service:8082` | `http://localhost:8082` |
| `APP_JIRA_SERVICE_URL` | `http://jira-integration-service:8083` | `http://localhost:8083` |
| `APP_SEMGREP_SERVICE_URL` | `http://semgrep-service:8085` | `http://localhost:8085` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_ |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `aspm.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `aspm.findings.ingest.result` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat` | _(пусто)_ |
| `APP_DEFAULT_SEMGREP_CONFIG` | `p/java` | _(пусто)_ |

`APP_DEFAULT_*` подставляются, если в `POST /api/v1/scans/semgrep` не указаны `target_path` / `semgrep_config`. Путь — **в контейнере `semgrep-service`**, каталог `WebGoat` нужно один раз клонировать: `demo/scan-targets/clone-webgoat.sh`.

---

## `reference-data-service`

Файл дефолтов: `services/reference-data-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8081` | `8081` |
| `APP_POSTGRES_DSN` | `postgres://aspm:aspm@postgres:5432/aspm?sslmode=disable` | `postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | `localhost:9092` |
| `APP_BDU_FEED_URL` | `https://bdu.fstec.ru/feed` | то же |
| `APP_BDU_INSECURE_SKIP_VERIFY` | — | `true` |
| `APP_NVD_API_BASE_URL` | `https://services.nvd.nist.gov/rest/json/cves/2.0` | то же |
| `APP_SYNC_SCHEDULER_ENABLED` | `true` | `true` |
| `APP_SYNC_INITIAL_DELAY` | `1m` | `1m` |
| `APP_SYNC_INTERVAL` | `24h` | `24h` |

---

## `processing-service`

Файл дефолтов: `services/processing-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8082` | `8082` |
| `APP_POSTGRES_DSN` | `postgres://aspm:aspm@postgres:5432/aspm?sslmode=disable` | `postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_ |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `aspm.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `aspm.findings.ingest.result` |

---

## `semgrep-service`

Файл дефолтов: `services/semgrep-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8085` | `8085` |
| `APP_SEMGREP_CONFIG` | `/app/demo/semgrep-rules.yml` | `/app/demo/semgrep-rules.yml` |
| `APP_SEMGREP_BINARY` | — | `semgrep` |

---

## `jira-integration-service`

Файл дефолтов: `services/jira-integration-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8083` | `8083` |
| `APP_POSTGRES_DSN` | `postgres://aspm:aspm@postgres:5432/aspm?sslmode=disable` | `postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable` |
| `APP_JIRA_BASE_URL` | `http://jira-mock:8090` | `https://example.atlassian.net` |
| `APP_JIRA_PROJECT_KEY` | `ASPM` | `ASPM` |

---

## `jira-mock`

Переменных окружения нет: порт **`8090`** зашит в `services/jira-mock/internal/app/app.go` (`:8090`).
