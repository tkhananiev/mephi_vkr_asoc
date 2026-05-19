# Демо: Semgrep + Gitleaks в GitLab CI → Atomic ASOC

Это **отдельный сценарий для проверки пайплайнов**, не часть деплоя **ASOC**. Платформа Atomic/ASOC выкладывается в кластер как в `deploy/k8s/`; здесь только пример **`gitlab-ci-webgoat/.gitlab-ci.yml`**, каталог **`ci/`** со скриптами нормализации отчётов и подсказки по **GitLab Runner**.

---

## Что положить в свой проект GitLab

Скопируйте в **корень своего репозитория** (например уже существующий WebGoat):

- [`gitlab-ci-webgoat/.gitlab-ci.yml`](.gitlab-ci.yml) → переименуйте/положите как **`.gitlab-ci.yml`** в корне проекта;
- каталог [`gitlab-ci-webgoat/ci/`](ci/) → **`ci/`** в корне с двумя скриптами `*_to_asoc_payload.py`.

Пайплайн сканирует **текущий checkout** репозитория (`CI_PROJECT_DIR`), ничего дополнительно не клонирует.

---

## GitLab Runner в Kubernetes

Раннер ставится **в кластер** (отдельный namespace, например `gitlab-runner`). Используйте **kubernetes executor**: джобы стартуют как Pod’ы, образы `semgrep/semgrep` и `zricethezav/gitleaks` задаются в `.gitlab-ci.yml`.

1. Поставьте chart **gitlab/gitlab-runner** (официальный Helm-репозиторий GitLab).
2. Скопируйте [`k8s/runner-values.example.yaml`](k8s/runner-values.example.yaml), подставьте **`gitlabUrl`** и **`runnerRegistrationToken`** из GitLab (project или group runner).
3. Выполните установку/upgrade (см. комментарии в начале файла).

Подробности и актуальные ключи values: [Install GitLab Runner on Kubernetes](https://docs.gitlab.com/runner/install/kubernetes.html).

GitLab при этом может быть **gitlab.com**, корпоративный инстанс за Ingress или вообще другой кластер — важен только доступ раннера до **gitlabUrl** по сети.

---

## Связь с ASOC

В **`.gitlab-ci.yml`** после сканирования выполняются джобы **`semgrep_send_asoc`** и **`gitleaks_send_asoc`** (стейдж `report`): скрипты **`ci/semgrep_to_asoc_payload.py`** и **`ci/gitleaks_to_asoc_payload.py`** собирают тело **`POST /api/v1/findings/ingest`** с **`"channel": "ci"`**.

### Владелец и проект консоли

1. **`ASOC_CONSOLE_JWT`** (masked) — JWT пользователя консоли с ролью **`user`**. Тогда прогон получает **`owner_user_id`** и попадает в ваш контур (не в «общий» от API-ключа).

2. **`ASOC_CONSOLE_PRODUCT_ID`** — числовой **`id`** строки из **`GET /api/v1/console/products`** (тот же пользователь JWT). Скрипты добавляют в JSON ingest поле **`console_product_id`** на верхнем уровне и дублируют **`console_product_id`** в **`metadata`** находок.

Пайплайн при наличии JWT отправляет ingest только с **`Authorization: Bearer …`** (без **`X-API-Key`** в том же запросе — иначе api-service не подставит владельца). Если заданы и JWT, и **`ASOC_API_KEY`**, для POST используется **JWT**.

Джобы отправки создаются, если задано **`ASOC_API_KEY`** или **`ASOC_CONSOLE_JWT`**.

Опционально **`ASOC_INGEST_URL`**; если не задан — подставляется `https://atomic-asoc.ru/api/v1/findings/ingest`.

### Отчёт и группы только по этому проекту

После деплоя миграции **`017_processing_run_console_product.sql`** строки отчёта и группы можно ограничить продуктом:

- **`GET /api/v1/report/vulnerabilities?console_product_id=<id>`**
- **`GET /api/v1/groups?console_product_id=<id>`**

Нужен **JWT пользователя** (не только API-ключ); api-service проверяет, что продукт принадлежит этому пользователю. Без параметра **`console_product_id`** по-прежнему возвращаются **все** прогоны вашего пользователя (все продукты).

См. также OpenAPI **`openapi.yaml`** в **api-service** и **`docs/DATABASE.md`** (`core.processing_runs.console_product_id`).
