# Kubernetes (контур ASOC)

**Стенд по умолчанию — доступ по публичному IP, без FQDN:** в Ingress не задаётся `host`, в `public-urls.yaml` — `http://<IP>` и `http://<IP>/jira`. Домен не требуется.

Манифесты в стиле **Kustomize**: Postgres + Kafka + микросервисы. Публичное **api-service** защищено переменной **`APP_AUTH_API_KEY`** (см. Secret).

## Развернуть (1–2 команды)

**Перед этим один раз:** собери образы (§1) или залей `asoc/*:latest` и `asoc/web:latest` в registry; сделай `deploy/k8s/secret.yaml` из `secret.example.yaml` (пароль в DSN = `postgres-password`); в **`deploy/k8s/public-urls.yaml`** укажи свой публичный **IP**; установи **Ingress Controller** (класс `nginx`), повесь на него внешний IP.

Из **корня** `mephi_vkr_asoc`:

```bash
kubectl apply -f deploy/k8s/secret.yaml
kubectl apply --load-restrictor=LoadRestrictionsNone -k deploy/k8s
```

Одной строкой:

```bash
kubectl apply -f deploy/k8s/secret.yaml && kubectl apply --load-restrictor=LoadRestrictionsNone -k deploy/k8s
```

После смены только манифестов / `public-urls.yaml` достаточно второй команды (и при смене URL Jira — `kubectl -n asoc rollout restart deployment/jira-mock`).

## Состав (что создаёт `kubectl apply -k`)

| Тип | Имя / файлы |
|-----|----------------|
| Namespace | `asoc` |
| ConfigMap | `asoc-public-urls` (`public-urls.yaml`) — публичный **IP/URL** и база для ссылок Jira |
| ConfigMap | `asoc-migrations` (Kustomize `configMapGenerator`) — SQL из `migrations/` для init Postgres |
| Secret | **`asoc-secrets`** — **не в kustomization**; создай из `secret.example.yaml` **до** или вместе с первым apply |
| Service + StatefulSet | `postgres` + PVC |
| Service + Deployment | `kafka` |
| Service + Deployment | `reference-data-service`, `processing-service`, `semgrep-service`, `api-service`, `jira-integration-service`, `jira-mock` |
| Service + Deployment | `asoc-web` (фронт) |
| Ingress (×2) | `asoc-app` (`/` → asoc-web), `asoc-jira` (`/jira` → jira-mock + rewrite) |

**Образы** везде вида `asoc/<сервис>:latest` — собери локально или залей в registry и поменяй `image` / `imagePullSecrets`.

## 1. Сборка образов

Из **корня репозитория** `mephi_vkr_asoc`:

```bash
docker build -f services/api-service/Dockerfile -t asoc/api-service:latest .
docker build -f services/reference-data-service/Dockerfile -t asoc/reference-data-service:latest .
docker build -f services/processing-service/Dockerfile -t asoc/processing-service:latest .
docker build -f services/semgrep-service/Dockerfile -t asoc/semgrep-service:latest .
docker build -f services/jira-integration-service/Dockerfile -t asoc/jira-integration-service:latest .
docker build -f services/jira-mock/Dockerfile -t asoc/jira-mock:latest .
```

Образ фронта собирается **из репозитория `mephi_vkr_asoc_front`** (рядом с этим каталогом):

```bash
cd ../mephi_vkr_asoc_front
docker build -t asoc/web:latest .
```

**minikube:** `eval $(minikube docker-env)` перед `docker build`, и в манифестах оставьте `imagePullPolicy: IfNotPresent`.

## 2. Секреты

```bash
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml
# отредактируй postgres-password, postgres-dsn (пароль должен совпадать), api-auth-key
kubectl apply -f deploy/k8s/secret.yaml
```

Файл `secret.yaml` с реальными значениями не коммить.

## 3. Применение

Флажок **`--load-restrictor=LoadRestrictionsNone`** нужен, потому что SQL-миграции подключаются из `../../migrations` относительно этой папки.

```bash
kubectl apply --load-restrictor=LoadRestrictionsNone -k deploy/k8s
```

## 4. Фронт (asoc-web)

После `kubectl apply` в кластере появляется `Deployment`/`Service` **asoc-web** (один `nginx` с собранной статикой и прокси на бэкенды по внутренним DNS-именам, см. `mephi_vkr_asoc_front/nginx/default.conf`).

Проброс в браузер:

```bash
kubectl -n asoc port-forward svc/asoc-web 8088:80
```

Открой `http://127.0.0.1:8088/`. API и Swagger с того же origin идут через тот же под.

## 5. Доступ к API (только бэкенд)

Проброс порта:

```bash
kubectl -n asoc port-forward svc/api-service 8080:8080
```

Вызов сценария (подставьте ключ из Secret):

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/scans/semgrep \
  -H "Authorization: Bearer ВАШ_API_КЛЮЧ" \
  -H "Content-Type: application/json" \
  -d '{"scanner_name":"semgrep"}'
```

Без ключа по-прежнему доступны `GET /health`, `GET /openapi.yaml`, `GET /swagger/`.

## 6. Ingress по IP (без FQDN)

Схема по умолчанию — **доступ по зарезервированному IP** (например `77.223.100.196` в Selectel), **DNS не обязателен**.

1. **`public-urls.yaml`** — выставь свой IP в ключах `public-origin` и **`jira-public-base-url`** (сейчас `http://77.223.100.196` и `http://77.223.100.196/jira`). После смены IP: `kubectl apply -f deploy/k8s/public-urls.yaml -n asoc` и перезапуск `jira-mock`:

   ```bash
   kubectl -n asoc rollout restart deployment/jira-mock
   ```

2. **Ingress Controller** (часто [ingress-nginx](https://kubernetes.github.io/ingress-nginx/deploy/)). Класс в манифестах: **`nginx`** (`spec.ingressClassName`). Если у провайдера другой класс — поменяй во всех Ingress.

3. **Два Ingress** в `ingress.yaml`:
   - **`asoc-app`**: правило **без `host`** → `http://<IP>/` попадает в **asoc-web** (UI + прокси API).
   - **`asoc-jira`**: префикс **`/jira`** → **jira-mock**, с **rewrite** (внутри сервиса пути как у обычного Jira: `/browse/...`).

4. **Повесь внешний IP на Ingress** (зависит от кластера):
   - **LoadBalancer** у сервиса контроллера: в панели Selectel / Magnum укажи **зарезервированный** IP (например тот, что на `router`) для этого LB **или** создай LB с `loadBalancerIP` (если облако поддерживает).
   - После выдачи адреса: `kubectl -n <ns-ingress> get svc` — внешний IP должен совпасть с тем, что в `asoc-public-urls`.

5. **Проверка:** `http://<IP>/` — UI; `http://<IP>/jira/` — консоль мока; ссылки в тикетах должны начинаться с `jira-public-base-url` из ConfigMap.

**TLS:** когда появится домен — добавь `spec.tls` и сертификаты (или cert-manager); для чистого IP TLS обычно не ставят без своего DNS.
