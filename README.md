# mephi_vkr_asoc

Backend **Atomic ASOC**: микросервисы на Go, PostgreSQL, Kafka для ingest находок, исполнители SAST (Semgrep, Gitleaks), SCA (Trivy), DAST (OWASP ZAP), интеграция с Jira. Локально — `deploy/compose.yaml`. Веб-клиент — репозиторий **`mephi_vkr_asoc_front`**.

Развёртывание и стенд: [`deploy/k8s/README.md`](deploy/k8s/README.md). HTTP API: OpenAPI в [`services/api-service/internal/swaggerui/openapi.yaml`](services/api-service/internal/swaggerui/openapi.yaml), UI — `/swagger` на `api-service`.

## Состав

- `services/api-service` — внешний API и оркестрация сценариев
- `services/reference-data-service` — синхронизация NVD и БДУ ФСТЭК
- `services/processing-service` — нормализация, корреляция, группировка
- `services/findings-adapter-service` — адаптация отчётов сканеров
- `services/jira-integration-service`, `services/jira-mock` — тикеты Jira
- `services/semgrep-service`, `gitleaks-service`, `trivy-sca-service`, `zap-dast-service` — сканирование
- `migrations` — схемы БД
- `deploy/k8s` — манифесты Kubernetes
- `deploy/scripts/rollout-stand.sh` — сборка образов и выкладка на стенд

## Сквозной сценарий

```text
Клиент → api-service (POST /api/v1/scans)
      → executor (semgrep | gitleaks | trivy-sca | zap-dast)
      → findings-adapter → Kafka ingest → processing-service
      → api-service (groups, report) → jira-integration-service
```

Ingest находок: при заданном `APP_KAFKA_BROKERS` — топик `asoc.findings.ingest`; иначе `api-service` может слать их в `processing-service` по HTTP (`POST /findings/ingest`). На K8s-стенде ingest через Kafka обязателен (см. `deploy/k8s/README.md`).

## Быстрый старт

```bash
docker compose -f deploy/compose.yaml up -d --build
```

Порты: api `8080`, reference-data `8081`, processing `8082`, jira `8083`, semgrep `8085`, kafka `9092`.

Kubernetes: см. [`deploy/k8s/README.md`](deploy/k8s/README.md).
