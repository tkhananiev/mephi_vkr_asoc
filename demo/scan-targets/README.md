# Каталог целей для Semgrep (SAST)

## Как это устроено

Semgrep читает **файлы** по пути `target_path` **внутри контейнера `semgrep-service`**. На хосте это подкаталог `mephi_vkr_aspm/demo/...`, в контейнере — `/app/demo/...`. Это не DAST по URL.

## OWASP WebGoat (основной сценарий в compose)

1. Один раз на хосте (из корня репозитория или из этой папки):

   ```bash
   ./demo/scan-targets/clone-webgoat.sh
   ```

   Либо: `cd demo/scan-targets && git clone --depth 1 https://github.com/WebGoat/WebGoat.git`

2. В `deploy/docker-compose.yml` у `api-service` заданы **`APP_DEFAULT_SCAN_TARGET_PATH=/app/demo/scan-targets/WebGoat`** и **`APP_DEFAULT_SEMGREP_CONFIG=p/java`**. Достаточно вызвать `POST /api/v1/scans/semgrep` с телом, например `{"scanner_name":"semgrep"}` (без `target_path` — подставится из env).

3. Первый запуск `p/java` может тянуть правила из реестра Semgrep — нужен исходящий интернет в контейнере `semgrep-service`.

Каталог **`WebGoat/`** в git не хранится (см. `.gitignore` в корне репозитория); перед сканом выполните **`clone-webgoat.sh`** в этой папке.

## Учебный `vulnerable-app` (короткое демо)

В репозитории есть `mephi_vkr_aspm/demo/vulnerable-app` (Python) и `demo/semgrep-rules.yml`. Пример пути в контейнере: `/app/demo/vulnerable-app`, при необходимости укажите `semgrep_config` под ваши правила.

## DVWA (PHP)

```bash
cd demo/scan-targets
git clone --depth 1 https://github.com/digininja/DVWA.git
```

В запросе: `"target_path": "/app/demo/scan-targets/DVWA"`, `"semgrep_config": "p/php"` (или `auto`).

## Важно

- Корреляция со справочником в прототипе — по **CVE** и/или **CWE** в метаданных находки; у произвольных правил реестра поля может не быть.
- Для защиты можно комбинировать короткий прогон на `vulnerable-app` и отдельно полный репозиторий (WebGoat/DVWA).
