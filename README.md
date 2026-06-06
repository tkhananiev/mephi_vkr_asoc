# Atomic ASOC (backend)

Микросервисы на Go, PostgreSQL, Kafka. SAST (Semgrep, Gitleaks), SCA (Trivy), DAST (ZAP), Jira. Веб-клиент — [`mephi_vkr_asoc_front`](https://github.com/tkhananiev/mephi_vkr_asoc_front).

Развёртывание: [`deploy/k8s/README.md`](deploy/k8s/README.md). API: [`openapi.yaml`](services/api-service/internal/swaggerui/openapi.yaml), UI — `/swagger` на `api-service`.

## Сервисы

| Каталог | Назначение |
|---------|------------|
| `services/api-service` | HTTP API, оркестрация сканов |
| `services/reference-data-service` | NVD, БДУ ФСТЭК |
| `services/processing-service` | нормализация, группы |
| `services/findings-adapter-service` | адаптеры отчётов сканеров |
| `services/jira-integration-service`, `jira-mock` | тикеты |
| `services/*-service` (semgrep, gitleaks, trivy-sca, zap-dast) | исполнители |
| `migrations/` | DDL PostgreSQL |
| `deploy/k8s/` | Kubernetes |
| `deploy/scripts/rollout-stand.sh` | сборка образов и выкладка |

## Поток данных

```text
POST /api/v1/scans → executor → findings-adapter → Kafka → processing → groups / report → Jira
```

Ingest находок: Kafka (`asoc.findings.ingest`) или HTTP fallback в compose. На K8s — только Kafka (`APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST=true`).

## Локально

```bash
docker compose -f deploy/compose.yaml up -d --build
```

Порты: api `8080`, reference-data `8081`, processing `8082`, jira `8083`, semgrep `8085`.

## Стенд

```bash
./deploy/scripts/rollout-stand.sh
```

Подробности — `deploy/k8s/README.md`.
