# Локальные файлы БДУ (vulxml + vullist)

Положите сюда копии с сайта ФСТЭК:

- `vulxml.zip`
- `vullist.xlsx`

Файлы игнорируются git (см. `.gitignore`).

## Стенд в Kubernetes

**Основной сценарий полного дампа для кластера** — том `bdu-catalog-import`, монтирование в под `reference-data-service` как `/bdu-import` и `kubectl cp` дампов, затем Job `bdu-bulk`. Пошагово: **`deploy/k8s/README.md`** (раздел про БДУ).

## Локально (docker-compose, разработка)

Задайте переменные окружения для `reference-data-service` в compose:

- `APP_BDU_VULXML_ZIP_PATH` — путь к `vulxml.zip`, к **`vulxml.xml`** (распакованный файл выгрузки) или `file:///…`
- `APP_BDU_VULLIST_XLSX_PATH` — путь к `vullist.xlsx` (или `file:///…`)

Если путь задан, скачивание по `APP_BDU_VULXML_ZIP_URL` / `APP_BDU_VULLIST_XLSX_URL` для этого файла не выполняется. Можно смешивать: например ZIP с диска, таблица по URL.

В `deploy/docker-compose.yml` для `reference-data-service` можно раскомментировать volume с этим каталогом и переменные `APP_BDU_*_PATH`.
