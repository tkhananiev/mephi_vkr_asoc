# Nginx на хосте VPS (обратный прокси к локальному backend)

Фронт отдаётся **статикой** с диска; маршруты API проксируются на **службы на `127.0.0.1`** (порты задаёт ваш `deploy/compose.yaml` и `vite.config.ts` во фронте).

## Сборка фронта

Репозиторий **`mephi_vkr_asoc_front`** (рядом с `mephi_vkr_asoc`):

```bash
cd mephi_vkr_asoc_front
npm ci
npm run build
```

## Размещение файлов

Скопируй каталог **`dist`** на сервер, например:

```text
/var/www/asoc/dist
```

В `asoc-site.conf` выставь `root` на этот путь (в примере по умолчанию `/var/www/asoc/dist`).

## Подключение к nginx

- Скопируй `asoc-site.conf` в `sites-available`, сделай симлинк в `sites-enabled`, либо подключи `include` из основного `nginx.conf`.
- Проверка и перезагрузка:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

## Соответствие портам backend

| Путь (prefix) | Сервис | localhost:порт |
|---------------|--------|----------------|
| `/api/v1/scans`, `/health`, `/openapi.yaml`, `/swagger`, `/api/v1/groups`, `/api/v1/report`, `/api/v1/console`, `/api/v1/sync`, см. `asoc-site.conf` | api-service | 8080 |
| `/auth` | auth-service | 8091 |
| `/api/v1/findings/ingest` | api-service (ингест с API-ключом/JWT; дальше Kafka или HTTP в processing) | 8080 |
| `/api/v1/findings` (кроме `…/ingest`, если задан более длинный `location`) | processing-service | 8082 |
| `/health/reference`, `/health/processing`, `/health/jira`, `/health/semgrep` | агрегированные `/health` для UI | 8081–8085 |

В `asoc-site.conf` префикс `/api/v1/sync` идёт через api-service, поэтому требует тот же `Authorization`/`X-API-Key`, что остальные API-методы, а уже api-service проксирует запрос в reference-data-service. Для этого префикса заданы увеличенные таймауты (до 3600s) — это важно для **`POST /api/v1/sync/bdu/bulk`** (полный импорт БДУ) и длинного **`POST /api/v1/sync/nvd`** без `cve_id`.

Публичный `/api/v1/tickets` не проксируется напрямую в jira-integration-service: создание задач выполняется через авторизованный маршрут api-service `/api/v1/groups/{id}/jira-ticket`.

Убедись, что контейнеры с этими портами подняты на той же машине.

## Образ статики под полигон
Из каталога **`mephi_vkr_asoc_front`**:

```bash
docker build -t asoc/web:latest .
```

Манифест `deploy/k8s/frontend.yaml` — см. `deploy/k8s/README.md`.
