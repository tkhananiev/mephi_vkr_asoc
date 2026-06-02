# Развёртывание в кластере (контур ASOC)

**Стенд по умолчанию — доступ по публичному IP, без FQDN:** в Ingress не задаётся `host`, в `public-urls.yaml` — `http://<IP>` и `http://<IP>/jira`. Домен не требуется.

Манифесты в стиле **Kustomize**: Postgres + Kafka + микросервисы. Публичное **api-service** защищено переменной **`APP_AUTH_API_KEY`** (см. Secret). **`APP_POSTGRES_DSN`** у **`api-service`** (из того же секрета, что DSN других сервисов) нужен для **продуктов консоли** в БД. У **`api-service`** включено **`APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST=true`** — передача находок в `processing-service` только через Kafka (брокер обязателен для старта).

## Развернуть (1–2 команды)

**Перед этим один раз:** собери образы (§1) или залей `asoc/*:latest` и `asoc/web:latest` в registry; сделай `deploy/k8s/secret.yaml` из `secret.example.yaml` (пароль в DSN = `postgres-password`); в **`deploy/k8s/public-urls.yaml`** укажи свой публичный **IP**; установи **Ingress Controller** (класс `nginx`), повесь на него внешний IP.

Из **корня** `mephi_vkr_asoc`:

```bash
kubectl apply -f deploy/k8s/secret.yaml
kubectl kustomize deploy/k8s --load-restrictor=LoadRestrictionsNone | kubectl apply -f -
```

(Если `kubectl apply -k` у вас понимает `--load-restrictor=LoadRestrictionsNone`, можно использовать и его.)

Одной строкой:

```bash
kubectl apply -f deploy/k8s/secret.yaml && kubectl kustomize deploy/k8s --load-restrictor=LoadRestrictionsNone | kubectl apply -f -
```

После смены только манифестов / `public-urls.yaml` достаточно второй команды (и при смене URL Jira — `kubectl -n asoc rollout restart deployment/jira-mock`).

## Синхронизация Git ↔ стенд (консистентность)

**Источник правды для манифестов приложения** — код в этом репозитории каталог **`deploy/k8s/`** (и собранные образы сервисов под поля **`image`** в **`workloads.yaml`** / ваш registry). То, что стенд «как Git», означает:

1. **Зафиксируй коммит** (или тег релиза), от которого катишь деплой.
2. **Манифесты ASOC** после `git pull` тех же изменений выполни заново из корня проекта:

   ```bash
   kubectl apply -f deploy/k8s/secret.yaml   # уже есть secret — можно не повторять, если только меняешь приложение без секретов
   kubectl kustomize deploy/k8s --load-restrictor=LoadRestrictionsNone | kubectl apply -f -
   ```

3. Если менялся **код Go** или **Dockerfile** сервиса — недостаточно только `apply`: нужно **собрать и залить образ** с тем **тегом**, который указан в Deployment (или обновить `image:` в **`workloads.yaml`** и применить), затем:

   ```bash
   kubectl -n asoc rollout restart deployment/<имя-deployment>
   ```

4. **Мониторинг** (Prometheus/Grafana через Helm **kps**) описан в **`deploy/k8s/monitoring/`** (`install.sh`, `kube-prometheus-values.yaml`, Ingress Grafana). После изменения values — снова **`./deploy/k8s/monitoring/install.sh`**. Конфиг **дашборда** через **`kubectl apply -k deploy/k8s/monitoring`**.

Стенд **не равен** Git автоматически: любое расхождение появляется, если делали **`kubectl edit`**, патчи вне репозитория или образы собраны **не из того коммита**. Тогда заноси изменения обратно в Git или восстанавливай ресурсы из манифестов.

## Состав (что задаёт каталог `deploy/k8s/`)

| Тип | Имя / файлы |
|-----|----------------|
| Namespace | `asoc` |
| ConfigMap | `asoc-public-urls` (`public-urls.yaml`) — публичный **IP/URL** и база для ссылок Jira |
| ConfigMap | `asoc-migrations` (Kustomize `configMapGenerator`) — SQL из `migrations/` для **первого init** Postgres (пустой том данных). Добавление файла в `kustomization.yaml` не выполнит DDL на уже запущенном кластере — см. § **«Новые миграции при уже существующей БД»**. |
| Secret | **`asoc-secrets`** — **не в kustomization**; создай из `secret.example.yaml` **до** или вместе с первым apply |
| Service + StatefulSet | `postgres` + PVC |
| Service + Deployment | `kafka` |
| **PVC `bdu-catalog-import`** | **`bdu-catalog-pvc.yaml`** — том **`/bdu-import`** под **`vulxml.xml`** и **`vullist.xlsx`**; загрузка и Job — см. § **«БДУ: полный дамп в назначенном окружении»** ниже |
| Service + Deployment | `reference-data-service`, `processing-service`, `semgrep-service`, `api-service`, `jira-integration-service`, `jira-mock` |
| Service + Deployment | `asoc-web` (фронт) |
| Ingress | `asoc-app`: основной домен (`/`, `/api`, `/jira` → **asoc-web**); **`jira.atomic-asoc.ru`** → **jira-mock:8090** |

**Образы** везде вида `asoc/<сервис>:latest` — собери локально или залей в registry и поменяй `image` / `imagePullSecrets`.

## Новые миграции при уже существующей БД

PostgreSQL StatefulSet монтирует `asoc-migrations` в **`/docker-entrypoint-initdb.d`** — скрипты выполняются **только при первой инициализации тома данных**. Если кластер давно живёт, после `git pull` с файлом например **`migrations/014_console_products_and_run_owner.sql`** нужно выполнить его содержимое **один раз** в рабочей БД (`asoc`), например:

```bash
# Подставьте имя пода postgres из своего стенда
kubectl -n asoc exec -i statefulset/postgres -- psql -U asoc -d asoc \
  -f /dev/stdin < migrations/014_console_products_and_run_owner.sql
```

(Удобно запускать из **корня** репозитория `mephi_vkr_asoc`.)

После DDL перезапустите **`deployment/api-service`** и **`deployment/processing-service`**, если менялись только образы через `kubectl rollout restart`; само наличие `APP_POSTGRES_DSN` у `api-service` задано в **`workloads.yaml`** из секрета `postgres-dsn`.

## БДУ: полный дамп в назначенном окружении (том PVC + Job)

Полная выгрузка ФСТЭК (`vulxml.xml` + `vullist.xlsx`) **импортируется во время работы приложения только когда том смонтирован в целевом полигоне**: файлы должны лежать в томе `bdu-catalog-import`, смонтированном у `reference-data-service` как **`/bdu-import`** (см. `workloads.yaml`). Упрощённая локальная сборка этого сценария не задаётся — на действующей площадке достаточно шагов ниже.

### 1. Применить PVC и дождаться пода справочника

После общего деплоя в кластере:

```bash
kubectl -n asoc get pvc bdu-catalog-import
kubectl -n asoc rollout status deployment/reference-data-service
```

При необходимости поправьте **`storageClassName`** и размер тома в **`bdu-catalog-pvc.yaml`** под свой кластер, затем снова примените kustomization.

### 2. Скопировать дампы с рабочей машины в под (в один и тот же PVC)

Имена в контейнере **обязательно** `/bdu-import/vulxml.xml` и `/bdu-import/vullist.xlsx` (или скопировать и переименовать).

```bash
export NS=asoc
POD=$(kubectl -n "$NS" get pods -l app=reference-data-service -o jsonpath='{.items[0].metadata.name}')
kubectl -n "$NS" cp ./vullist.xlsx  "$POD":/bdu-import/vullist.xlsx
kubectl -n "$NS" cp ./vulxml.xml   "$POD":/bdu-import/vulxml.xml
kubectl -n "$NS" exec "$POD" -- ls -lh /bdu-import/
```

Подставьте пути к файлам на своём ноуте (`./vullist.xlsx` может быть абсолютным, например `~/Downloads/vullist.xlsx`).

У **ReadWriteOnce** том висит за конкретным узлом: копировать нужно в тот же под `reference-data-service`, который реально монтирует PVC (реплик обычно 1).

### 3. Запустить фоновый импорт (Job, не Ingress)

Не идёт через Ingress — нет ограничения по времени на прокси. Манифест **`bdu-bulk-job.yaml`** не входит в `kustomization`, подключается отдельно.

```bash
kubectl -n asoc delete job bdu-bulk --ignore-not-found
kubectl apply -f deploy/k8s/bdu-bulk-job.yaml
kubectl -n asoc logs -f job/bdu-bulk
```

Ответ триггер-контейнера в конце — JSON со счётчиками от `reference-data-service`. Датасет большой, прогон может занять многие часы.

**Что смотреть, если «тишина»:**

1. Логи приложения (раз в ~90 с или каждые 20 батчей — строки **`[bdu-bulk]`**, фазы **vulxml** / **vullist**):

   ```bash
   kubectl -n asoc logs deployment/reference-data-service --tail=100 -f
   ```

2. Пока парсится огромный XML, до **первых 500** записей сообщений из п.1 может не быть какое-то время — в логах всё равно должна быть строка **`phase vulxml — streaming`**. После неё идут heartbeat’ы с `batch=…`.

3. В БД (если образ с `UpdateSyncRunProgress`): строка в **`audit.reference_sync_runs`** со `status='running'` и растущими **`items_processed`**.

### 4. Проверка в PostgreSQL

Пока идёт bulk, смотри **активную** строку (`status='running'`) — в актуальных образах `reference-data-service` счётчики **`items_processed` / `items_inserted`** обновляются после каждого батча. До этого обновления кода они оставались нулевыми до конца всего импорта.

```bash
kubectl -n asoc exec postgres-0 -- psql -U asoc -d asoc -c \
  "SELECT id,status,items_discovered,items_processed,items_inserted,items_updated,
          started_at, NOW()-started_at AS elapsed
   FROM audit.reference_sync_runs
   WHERE source_code='bdu_fstec'
   ORDER BY started_at DESC LIMIT 3;"
```

Рост числа записей в каталоге (должен увеличиваться параллельно с vulxml-батчами):

```bash
kubectl -n asoc exec postgres-0 -- psql -U asoc -d asoc -tAc \
  "SELECT COUNT(*) FROM catalog.reference_records WHERE source_code='bdu_fstec';"
```

Если долго **`items_*` = 0** и счётчик каталога не растёт — сервис ещё только читает/парсит большой `vulxml.xml` до первого батча записей в БД.

Последние синки после завершения:

```bash
kubectl -n asoc exec postgres-0 -- psql -U asoc -d asoc -c \
  "SELECT id,status,items_processed,items_inserted,items_updated,error_message,started_at,finished_at FROM audit.reference_sync_runs WHERE source_code='bdu_fstec' ORDER BY started_at DESC LIMIT 5;"
```

**Быстро с ноутбука**, если дампы лежат **в родительском каталоге над `mephi_vkr_asoc`** как `vullist.xlsx` и `vulxml.xml`: из **`mephi_vkr_asoc`** выполни `deploy/scripts/import-bdu-k8s-from-repo-root.sh` (переменная **`NS`** при другом namespace). Флаги **`--only-copy`** / **`--only-job`** — только загрузить в том или только перезапустить Job.

В **`workloads.yaml`** уже заданы `APP_BDU_BULK_ENABLED`, `APP_BDU_VULXML_ZIP_PATH` и `APP_BDU_VULLIST_XLSX_PATH`; нужен актуальный образ **`reference-data-service`**. RSS (`APP_BDU_FEED_URL`) дополняет лентой; bulk пишет в `catalog.reference_records` и `catalog.reference_aliases`.

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

Флаг **`--load-restrictor=LoadRestrictionsNone`** нужен для `kubectl kustomize`, потому что SQL-миграции подключаются из `../../migrations` относительно каталога `deploy/k8s`.

```bash
kubectl kustomize deploy/k8s --load-restrictor=LoadRestrictionsNone | kubectl apply -f -
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
curl -sS -X POST http://127.0.0.1:8080/api/v1/scans \
  -H "Authorization: Bearer ВАШ_API_КЛЮЧ" \
  -H "Content-Type: application/json" \
  -d '{"scanner_id":"semgrep","scanner_name":"semgrep"}'
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
   - Второй вход к моку: хост **`jira.atomic-asoc.ru`** на том же Ingress (TLS общий секрет **`asoc-tls`**) — нужны **A-запись** и включение имени в сертификат (перевыпуск cert-manager после `apply`).
   - **`/jira/`** на основном домене остаётся: **asoc-web** → **jira-mock**.

4. **Повесь внешний IP на Ingress** (зависит от кластера):
   - **LoadBalancer** у сервиса контроллера: в панели Selectel / Magnum укажи **зарезервированный** IP (например тот, что на `router`) для этого LB **или** создай LB с `loadBalancerIP` (если облако поддерживает).
   - После выдачи адреса: `kubectl -n <ns-ingress> get svc` — внешний IP должен совпасть с тем, что в `asoc-public-urls`.

5. **Проверка:** `http://<IP>/` — UI; `http://<IP>/jira/` — консоль мока; ссылки в тикетах должны начинаться с `jira-public-base-url` из ConfigMap.

**TLS:** когда появится домен — добавь `spec.tls` и сертификаты (или cert-manager); для чистого IP TLS обычно не ставят без своего DNS.

## 7. metrics-server и HPA (нагрузочные тесты)

- **`metrics-server`** должен быть установлен в кластере (для `kubectl top` и метрик CPU в HPA). В **minikube**: `minikube addons enable metrics-server`.
- Файл **`hpa.yaml`** (входит в `kubectl apply -k`): HorizontalPodAutoscaler для **`api-service`** и **`processing-service`** с целевой утилизацией **CPU 90%** от `resources.requests` в `workloads.yaml`. Порог задан как нефункциональное требование к автомасштабированию.
- Проверка: `kubectl get hpa -n asoc -w` во время нагрузки (каталог **`../loadtest/`** вне репозитория).

## 8. Prometheus и Grafana

См. **`deploy/k8s/monitoring/README.md`**: установка **kube-prometheus-stack** (Helm), импорт дашборда **`grafana-dashboard-asoc-loadtest.json`**, съём скриншотов нагрузки и реплик.

## 9. Нагрузочное тестирование и отчёт

Каталог **`../loadtest/`** (рядом с репозиторием, вне Git):

- Скрипты **wrk** / **hey** и инструкция: **`README.md`**, **`METHODOLOGY.md`**
- Сравнение с **Defect Dojo**: **`defectdojo/README.md`**
- Шаблон отчёта: **`LOAD_TEST_REPORT.md`**
