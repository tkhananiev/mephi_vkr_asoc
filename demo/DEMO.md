# Demo Guide

## Цель сценария

Показать сквозной поток:

`HTTP-клиент -> api-service -> semgrep-service (Semgrep) -> processing-service -> correlation with NVD/BDU -> vulnerability passport -> Jira ticket`

## Что нужно перед запуском

- Docker Desktop запущен
- Папка проекта: `mephi_vkr_aspm`
- Один раз клонировать **WebGoat** в `demo/scan-targets` (сканирование по умолчанию смотрит туда):

  ```bash
  ./demo/scan-targets/clone-webgoat.sh
  ```

## Поднять стек

```bash
cd mephi_vkr_aspm
docker compose -f deploy/docker-compose.yml up -d --build
```

В compose для `api-service` и `processing-service` задан **`APP_KAFKA_BROKERS=kafka:9092`**: ingest находок идёт через топики **`aspm.findings.ingest`** / **`aspm.findings.ingest.result`** (см. `docs/ARCHITECTURE.md`). Без этой переменной `api-service` использует только HTTP `POST .../findings/ingest`.

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

В compose для `api-service` заданы **`APP_DEFAULT_SCAN_TARGET_PATH`** (каталог WebGoat в контейнере semgrep) и **`APP_DEFAULT_SEMGREP_CONFIG=p/java`**, поэтому достаточно минимального тела:

```bash
curl -X POST "http://localhost:8080/api/v1/scans/semgrep" \
  -H "Content-Type: application/json" \
  -d '{"scanner_name":"semgrep"}'
```

Короткое демо на учебном Python (и явные пути в JSON):

```bash
curl -X POST "http://localhost:8080/api/v1/scans/semgrep" \
  -H "Content-Type: application/json" \
  -d '{
    "target_path": "/app/demo/vulnerable-app",
    "scanner_name": "semgrep",
    "semgrep_config": "/app/demo/semgrep-rules.yml"
  }'
```

## Что должно получиться

- `Semgrep` находит находки в коде цели (WebGoat — большой Java-проект; учебный `vulnerable-app` — один сценарий под `semgrep-rules.yml`)
- finding уходит в `processing-service`
- при наличии **CVE/CWE** в метаданных срабатывания — корреляция со справочником (для гарантированного сценария по **CWE-78** используйте запрос с `vulnerable-app` и локальным YAML выше)
- создаётся `vulnerability group`
- создаётся тикет в `jira-mock`

## Полезные запросы после прогона

```bash
curl "http://localhost:8081/api/v1/sync/runs?limit=10"
curl "http://localhost:8082/api/v1/groups?limit=10"
curl "http://localhost:8090/health"
```
