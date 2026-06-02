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
| `/api/v1/scans`, `/health`, `/openapi.yaml`, `/swagger`, `/api/v1/groups`, `/api/v1/report`, `/api/v1/console`, см. `asoc-site.conf` | api-service | 8080 |
| `/api/v1/findings/ingest` | api-service (ингест с API-ключом/JWT; дальше Kafka или HTTP в processing) | 8080 |
| `/api/v1/sync` | reference-data-service | 8081 |
| `/api/v1/findings` (кроме `…/ingest`, если задан более длинный `location`) | processing-service | 8082 |
| `/api/v1/tickets` | jira-integration-service | 8083 |
| `/health/reference`, `/health/processing`, `/health/jira`, `/health/semgrep` | агрегированные `/health` для UI | 8081–8085 |

В `asoc-site.conf` для префикса `/api/v1/sync` заданы увеличенные таймауты (до 3600s) — это важно для **`POST /api/v1/sync/bdu/bulk`** (полный импорт БДУ) и длинного **`POST /api/v1/sync/nvd`** без `cve_id`.

Убедись, что контейнеры с этими портами подняты на той же машине.

## Образ статики под полигон
Из каталога **`mephi_vkr_asoc_front`**:

```bash
docker build -t asoc/web:latest .
```

Манифест `deploy/k8s/frontend.yaml` — см. `deploy/k8s/README.md`.
