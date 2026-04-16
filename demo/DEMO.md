# Demo Guide

## Цель сценария

Показать сквозной поток:

`HTTP-клиент -> api-service -> semgrep-service (Semgrep) -> processing-service -> correlation with NVD/BDU -> vulnerability passport -> Jira ticket`

## Что нужно перед запуском

- Docker Desktop запущен
- Папка проекта: `mephi_vkr_aspm`

## Поднять стек

```bash
cd mephi_vkr_aspm
docker compose -f deploy/docker-compose.yml up -d --build
```

## Проверка здоровья сервисов

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8085/health
curl http://localhost:8090/health
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

## Подготовка справочников

### NVD по конкретному CVE

```bash
curl -X POST "http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228"
```

### БДУ ФСТЭК

```bash
curl -X POST "http://localhost:8081/api/v1/sync/bdu"
```

Примечание:
- в тестовом контуре БДУ может быть недоступен по сети или TLS;
- в этом случае сервис использует демонстрационный fallback/seed, чтобы защитный сценарий всё равно отработал.

## Основной запрос для предзащиты

```bash
curl -X POST "http://localhost:8080/api/v1/scans/semgrep" \
  -H "Content-Type: application/json" \
  -d '{
    "target_path": "/app/demo/vulnerable-app",
    "scanner_name": "semgrep"
  }'
```

## Что должно получиться

- `Semgrep` находит демонстрационную уязвимость
- finding уходит в `processing-service`
- происходит корреляция по `CVE-2021-44228`
- создаётся `vulnerability group`
- создаётся тикет в `jira-mock`

## Полезные запросы после прогона

```bash
curl "http://localhost:8081/api/v1/sync/runs?limit=10"
curl "http://localhost:8082/api/v1/groups?limit=10"
curl "http://localhost:8090/health"
```
