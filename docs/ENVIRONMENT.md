# Переменные окружения (сводка)

Значения для docker-compose: **`deploy/docker-compose.yml`**.  
Если переменной нет в compose, при локальном запуске действует **дефолт из `config.go`** указанного сервиса.

---

## Инфраструктура (не `APP_*`)

| Сервис | Переменная | Значение в compose |
|--------|------------|-------------------|
| `postgres` | `POSTGRES_DB` | `asoc` |
| | `POSTGRES_USER` | `asoc` |
| | `POSTGRES_PASSWORD` | `asoc` |
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
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_ — тогда передача находок в `processing` по **HTTP** |
| `APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST` | — | `false`; **`true`** — в проде/ДИТ без брокера `api-service` не стартует |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `asoc.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `asoc.findings.ingest.result` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat/` | `/app/demo/scan-targets/WebGoat/` |
| `APP_DEFAULT_SEMGREP_CONFIG` | `p/java` | `p/java` |
| `APP_POSTGRES_DSN` | в `deploy/k8s`: Secret `postgres-dsn` | _(пусто)_ — если **не задано**, недоступны `GET|POST /api/v1/console/products` (ответ 503 «products store unavailable»). Та же строка подключения, что у остальных сервисов к БД **`asoc`**. |
| `APP_JWT_SECRET` | из Secret `jwt-secret` (общий с auth-service) | _(пусто)_ — без секрета `api-service` **не ставит контекст консольного пользователя**: скан может идти с API-ключом, но **без разделения находок по пользователю**; секрет должен быть **не короче 32 байт**, иначе проверка JWT отключена (см. лог сервиса при старте). |
| `APP_AUTH_API_KEY` | в `deploy/k8s` из Secret | _(пусто)_ — если задано, для защищённых префиксов `/api/*` нужен заголовок `Authorization: Bearer <ключ>` или `X-API-Key`, либо **Bearer JWT** от `auth-service` (выписан тем же секретом, что **`APP_JWT_SECRET`**); без ключа — только допустимый JWT пользователя или админа. `/health`, Swagger без ключа. |
| `APP_K8S_OPS_ENABLED` | в `deploy/k8s` задаётся **`true`** вместе с RBAC **`api-pod-ops-rbac.yaml`** и `serviceAccountName: api-service` | `false` — при **`true`** админ-эндпойнты **`/api/v1/admin/ops/docker/logs`** и **`…/restart`** работают через **Kubernetes API** (хвост лога пода, rolling restart deployment). Имеет **приоритет** над Docker. |
| `APP_K8S_NAMESPACE` | в манифесте: `fieldRef` → namespace пода | `asoc` |
| `APP_K8S_POD_CONTAINER` | — | `app` — имя контейнера в `workloads.yaml` |
| `APP_K8S_KUBECONFIG` | — | _(пусто)_ — в кластере используется in-cluster config; для локальной отладки снаружи — путь к kubeconfig |
| `APP_DOCKER_OPS_ENABLED` | в локальном compose часто `true` (см. `deploy/docker-compose.yml`) | `false` — если **`APP_K8S_OPS` выключен** и **`true`**, те же эндпойнты вызывают CLI Docker (**нужен** **`/var/run/docker.sock`** в контейнер `api-service`). |

Для пользователя консоли (JWT **`role=user`**) `api-service` подставляет **`owner_user_id`** в ingest (Kafka или HTTP на `processing`) и проксирует **`GET /api/v1/groups`** и **`GET /api/v1/report/vulnerabilities`** в `processing-service` с внутренним заголовком `X-ASOC-Console-User-ID` (клиент не должен сам его задавать). Прямой вызов `processing-service:8082` без заголовка по-прежнему даёт общий список (нагрузочные скрипты, отладка).

`APP_DEFAULT_*` подставляются, если в `POST /api/v1/scans` с `scanner_id: "semgrep"` не указаны `target_path` / `semgrep_config` (то же поведение у устаревшего `POST /api/v1/scans/semgrep`). Путь — **в контейнере `semgrep-service`**, каталог `WebGoat` нужно один раз клонировать: `demo/scan-targets/clone-webgoat.sh`.

---

## `reference-data-service`

Файл дефолтов: `services/reference-data-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8081` | `8081` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | `localhost:9092` |
| `APP_BDU_FEED_URL` | `https://bdu.fstec.ru/ubi/vul/rss` | RSS 2.0 ленты уязвимостей. Путь `/feed` отдаёт HTML «список каналов»; если в конфиге остался `.../feed`, клиент после неуспешного разбора XML сам пробует `.../ubi/vul/rss` (только `bdu.fstec.ru`) |
| `APP_BDU_ROOT_CA_FILE` | — | _(пусто)_; опционально PEM (корень+sub); при ротации УЦ может не совпасть с листом ФСТЭК |
| `APP_BDU_INSECURE_SKIP_VERIFY` | `true` | `true`; надёжный режим для фида БДУ при смене промежуточных сертификатов |
| `APP_NVD_API_BASE_URL` | `https://services.nvd.nist.gov/rest/json/cves/2.0` | то же |
| `APP_NVD_API_KEY` | — | _(пусто)_; ключ NVD для более высокого лимита запросов |
| `APP_NVD_PAGE_SIZE` | — | `2000` |
| `APP_NVD_MAX_PAGES` | — | `0` (= все страницы); иначе ограничение числа страниц за один `POST /sync/nvd` |
| `APP_NVD_HTTP_REQUEST_TIMEOUT` | — | Таймаут **одного** HTTP GET к NVD (например `20m`). По умолчанию **15m**; раньше было 120s — при полном синке и большой странице возможна ошибка `context deadline exceeded`. |
| `APP_SYNC_SCHEDULER_ENABLED` | `true` | `true` |
| `APP_SYNC_INITIAL_DELAY` | `1m` | `1m` |
| `APP_SYNC_INTERVAL` | `24h` | `24h` |
| `APP_BDU_BULK_ENABLED` | — | `true`; при `false` ручной `POST /api/v1/sync/bdu/bulk` недоступен (импортёр не создаётся) |
| `APP_BDU_VULXML_ZIP_URL` | — | URL `vulxml.zip`; не используется, если задан `APP_BDU_VULXML_ZIP_PATH` |
| `APP_BDU_VULLIST_XLSX_URL` | — | URL `vullist.xlsx`; не используется, если задан `APP_BDU_VULLIST_XLSX_PATH` |
| `APP_BDU_VULXML_ZIP_PATH` | — | _(пусто)_ Локальный путь к **`vulxml.zip`** или к **распакованному `vulxml.xml`**, или `file:///…` (офлайн; см. `demo/bdu/`). Расширение `.xml` → потоковое чтение файла без zip |
| `APP_BDU_VULLIST_XLSX_PATH` | — | _(пусто)_ Локальный путь к `vullist.xlsx` или `file:///…` |
| `APP_BDU_BULK_BATCH_SIZE` | — | `500` — размер батча при полном импорте |

**Режимы БДУ:** периодический и ручной **`POST /api/v1/sync/bdu`** читают **RSS** (`APP_BDU_FEED_URL`) — актуальные записи ленты. Полная база из официальных статических файлов ФСТЭК — ручной **`POST /api/v1/sync/bdu/bulk`**: по умолчанию скачивает `vulxml.zip` и `vullist.xlsx` с HTTPS URL выше; при задании **`APP_BDU_*_PATH`** соответствующий файл берётся с диска (без HTTP). Для compose/K8s смонтируйте каталог с копиями файлов и укажите пути внутри контейнера. Прогон bulk может занимать заметное время.

---

## `processing-service`

Файл дефолтов: `services/processing-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8082` | `8082` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable` |
| `APP_KAFKA_BROKERS` | `kafka:9092` | _(пусто)_; если задан — consumer ingest + producer result |
| `APP_KAFKA_TOPIC_FINDINGS_INGEST` | — | `asoc.findings.ingest` |
| `APP_KAFKA_TOPIC_FINDINGS_RESULT` | — | `asoc.findings.ingest.result` |
| `APP_HTTP_FINDINGS_INGEST` | — | если брокер **задан** — по умолчанию **`false`** (маршрут `POST /api/v1/findings/ingest` не регистрируется); **`true`** — включить для отладки. Если брокера **нет** — по умолчанию **`true`** (единственный способ принять находки) |

---

## `semgrep-service`

Файл дефолтов: `services/semgrep-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8085` | `8085` |
| `APP_SEMGREP_CONFIG` | `p/java` | `p/java` |
| `APP_DEFAULT_SCAN_TARGET_PATH` | `/app/demo/scan-targets/WebGoat/` | `/app/demo/scan-targets/WebGoat/` |
| `APP_SEMGREP_BINARY` | — | `semgrep` |

---

## `jira-integration-service`

Файл дефолтов: `services/jira-integration-service/internal/config/config.go`

| Переменная | В compose | Дефолт в коде |
|------------|-----------|---------------|
| `APP_HTTP_PORT` | `8083` | `8083` |
| `APP_POSTGRES_DSN` | `postgres://asoc:asoc@postgres:5432/asoc?sslmode=disable` | `postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable` |
| `APP_JIRA_BASE_URL` | `http://jira-mock:8090` | `https://example.atlassian.net` |
| `APP_JIRA_PROJECT_KEY` | `ASOC` | `ASOC` |

---

## `jira-mock`

Файл: `services/jira-mock/internal/config/config.go`. В `deploy/docker-compose.yml` для публичных ссылок: **`JIRA_PUBLIC_BASE_URL`** на хосте (подставляется в `APP_JIRA_PUBLIC_BASE_URL`).

| Переменная | Назначение |
|------------|------------|
| `APP_HTTP_PORT` | Порт сервера (по умолчанию `8090`). |
| `APP_JIRA_PUBLIC_BASE_URL` | База для поля `self` при создании issue: `{base}/browse/{KEY}` — для VPS задай **http://ваш-ip:8090**, иначе по умолчанию `http://localhost:8090`. |

- **`GET /console`** — список всех тикетов, созданных в этом процессе mock (память; после перезапуска контейнера список пустой).
- **`GET /browse/ASOC-N`** — карточка задачи (ссылка с `/console` или из поля `self` в API).
