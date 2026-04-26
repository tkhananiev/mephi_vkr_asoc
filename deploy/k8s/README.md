# Kubernetes (контур ASOC)

Манифесты в стиле **Kustomize**: Postgres + Kafka + микросервисы. Публичное **api-service** защищено переменной **`APP_AUTH_API_KEY`** (см. Secret).

## 1. Сборка образов

Из **корня репозитория** `mephi_vkr_asoc`:

```bash
docker build -f services/api-service/Dockerfile -t asoc/api-service:latest .
docker build -f services/reference-data-service/Dockerfile -t asoc/reference-data-service:latest .
docker build -f services/processing-service/Dockerfile -t asoc/processing-service:latest .
docker build -f services/semgrep-service/Dockerfile -t asoc/semgrep-service:latest .
docker build -f services/jira-integration-service/Dockerfile -t asoc/jira-integration-service:latest .
docker build -f services/jira-mock/Dockerfile -t asoc/jira-mock:latest .
docker build -f web/Dockerfile -t asoc/web:latest .
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

После `kubectl apply` в кластере появляется `Deployment`/`Service` **asoc-web** (один `nginx` с собранной статикой и прокси на бэкенды по внутренним DNS-именам, см. `web/nginx/default.conf`).

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

## 6. Ingress / TLS

В манифестах не заданы: при необходимости добавьте Ingress и TLS в своём кластере, URL Jira mock (`APP_JIRA_PUBLIC_BASE_URL`) выставьте на внешний адрес.
