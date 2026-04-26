# Nginx на хосте VPS (тесты до миграции в Kubernetes)

Фронт отдаётся **статикой** с диска; маршруты API проксируются на **docker-compose** на `127.0.0.1` (тот же план портов, что в `mephi_vkr_asoc_front/vite.config.ts`).

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
| `/api/v1/scans`, `/health`, `/openapi.yaml`, `/swagger` | api-service | 8080 |
| `/api/v1/sync` | reference-data-service | 8081 |
| `/api/v1/findings`, `/api/v1/groups` | processing-service | 8082 |
| `/api/v1/tickets` | jira-integration-service | 8083 |

Убедись, что контейнеры с этими портами подняты на той же машине.

## Образ для Kubernetes

Из каталога **`mephi_vkr_asoc_front`**:

```bash
docker build -t asoc/web:latest .
```

Манифест `deploy/k8s/frontend.yaml` — см. `deploy/k8s/README.md`.
