# Переменные окружения

Значения в compose: `deploy/compose.yaml`. Дефолты — в `config.go` соответствующего сервиса.

---

## Инфраструктура

| Сервис | Переменная | Значение |
|--------|------------|----------|
| `postgres` | `POSTGRES_DB` / `POSTGRES_USER` / `POSTGRES_PASSWORD` | `asoc` / `asoc` / `asoc` |
| `kafka` | см. `deploy/compose.yaml` | настройки KRaft |

---

## `api-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8080` | `8080` |
| `APP_PROCESSING_SERVICE_URL` | `http://processing-service:8082` | `http://localhost:8082` |
| `APP_JIRA_SERVICE_URL` | `http://jira-integration-service:8083` | `http://localhost:8083` |
| `APP_SEMGREP_SERVICE_URL` | `http://semgrep-service:8085` | `http://localhost:8085` |
| `APP_GITLEAKS_SERVICE_URL` | `http://gitleaks-service:8086` | `http://localhost:8086` |
| `APP_SCA_SERVICE_URL` | `http://trivy-sca-service:8088` | `http://localhost:8088` |
| `APP_DAST_SERVICE_URL` | `http://zap-dast-service:8089` | `http://localhost:8089` |
| `APP_FINDINGS_ADAPTER_URL` | `http://findings-adapter-service:8090` | `http://localhost:8090` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_ — передача находок по HTTP |
| `APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST` | — | `false` |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `asoc.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `asoc.findings.ingest.result` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat/` | то же |
| `APP_DEFAULT_SEMGREP_CONFIG` | `p/java` | `p/java` |
| `APP_POSTGRES_DSN` | из Secret `postgres-dsn` (k8s) | _(пусто)_ — без него `/api/v1/console/products` недоступен |
| `APP_JWT_SECRET` | из Secret `jwt-secret` | _(пусто)_ — без него разделение находок по пользователю не работает; минимум 32 байта |
| `APP_AUTH_API_KEY` | из Secret (k8s) | _(пусто)_ — если задан, защищённые `/api/*` требуют `Authorization: Bearer` или `X-API-Key` |
| `APP_K8S_OPS_ENABLED` | `true` (k8s) | `false` — эндпойнты `/admin/ops` работают через Kubernetes API |
| `APP_K8S_NAMESPACE` | через `fieldRef` | `asoc` |
| `APP_K8S_POD_CONTAINER` | — | `app` |
| `APP_K8S_KUBECONFIG` | — | _(пусто)_ |
| `APP_DOCKER_OPS_ENABLED` | часто `true` (compose) | `false` — те же эндпойнты через Docker CLI (`/var/run/docker.sock`) |

---

## `reference-data-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8081` | `8081` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `...@localhost:5432/...` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | `localhost:9092` |
| `APP_BDU_FEED_URL` | `https://bdu.fstec.ru/ubi/vul/rss` | то же |
| `APP_BDU_ROOT_CA_FILE` | — | _(пусто)_ |
| `APP_BDU_INSECURE_SKIP_VERIFY` | `true` | `true` |
| `APP_NVD_API_BASE_URL` | `https://services.nvd.nist.gov/rest/json/cves/2.0` | то же |
| `APP_NVD_API_KEY` | — | _(пусто)_ |
| `APP_NVD_PAGE_SIZE` | — | `2000` |
| `APP_NVD_MAX_PAGES` | — | `0` (все страницы) |
| `APP_NVD_HTTP_REQUEST_TIMEOUT` | — | `15m` |
| `APP_SYNC_SCHEDULER_ENABLED` | `true` | `true` |
| `APP_SYNC_INITIAL_DELAY` | `1m` | `1m` |
| `APP_SYNC_INTERVAL` | `24h` | `24h` |
| `APP_BDU_BULK_ENABLED` | — | `true` |
| `APP_BDU_VULXML_ZIP_URL` | — | URL для скачивания `vulxml.zip` |
| `APP_BDU_VULLIST_XLSX_URL` | — | URL для скачивания `vullist.xlsx` |
| `APP_BDU_VULXML_ZIP_PATH` | — | _(пусто)_ — локальный путь к файлу |
| `APP_BDU_VULLIST_XLSX_PATH` | — | _(пусто)_ — локальный путь к файлу |
| `APP_BDU_BULK_BATCH_SIZE` | — | `500` |

---

## `processing-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8082` | `8082` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `...@localhost:5432/...` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_ |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `asoc.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `asoc.findings.ingest.result` |
| `APP_HTTP_FINDINGS_INGEST` | — | `true` без брокера, `false` с брокером |

---

## `trivy-sca-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8088` | `8088` |
| `APP_TRIVY_BINARY` | — | `trivy` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat/` | то же |
| `APP_TRIVY_GIT_WORK_ROOT` | `/var/asoc-trivy-git` | `/tmp/asoc-trivy-git-work` |

---

## `zap-dast-service`

Образ на базе `ghcr.io/zaproxy/zaproxy:stable`.

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8089` | `8089` |
| `APP_ZAP_HOME` | `/zap` | `/zap` |
| `APP_ZAP_SCAN_TIMEOUT_MIN` | `8` | `8` |
| `APP_ZAP_USE_STUB` | — | `false` — `true` для отладки без образа ZAP |

---

## `semgrep-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8085` | `8085` |
| `APP_SEMGREP_CONFIG` | `p/java` | `p/java` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat/` | то же |
| `APP_SEMGREP_BINARY` | — | `semgrep` |

---

## `jira-integration-service`

| Переменная | Compose | Дефолт |
|------------|---------|--------|
| `APP_HTTP_PORT` | `8083` | `8083` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `...@localhost:5432/...` |
| `APP_JIRA_BASE_URL` | `http://jira-mock:8090` | `https://example.atlassian.net` |
| `APP_JIRA_PROJECT_KEY` | `ASOC` | `ASOC` |

---

## `jira-mock`

| Переменная | Назначение |
|------------|------------|
| `APP_HTTP_PORT` | Порт (по умолчанию `8090`). |
| `APP_JIRA_PUBLIC_BASE_URL` | База для поля `self`: `{base}/browse/{KEY}`. Для VPS: `http://<ip>:8090`. |

`GET /console` — список тикетов в памяти процесса. `GET /browse/ASOC-N` — карточка задачи.
