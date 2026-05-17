# Defect Dojo — стенд для сравнительного нагрузочного теста

Цель: поднять **Defect Dojo** отдельно от ASOC и прогнать **те же сценарии wrk** (см. `loadtest/README.md`), чтобы заполнить таблицу в `LOAD_TEST_REPORT.md`.

## Вариант A — официальный Docker Compose (рекомендуется для диплома)

Репозиторий upstream постоянно обновляется — клонируй актуальную ветку и следуй их README:

```bash
git clone https://github.com/DefectDojo/django-DefectDojo.git
cd django-DefectDojo
# Читай readme: обычно docker compose / dedicated compose file для «всё в одном»
```

После старта UI обычно на порту **8080** (уточни в выводе `docker compose` / `.env`). Базовый URL для `wrk`:

- главная или логин: `http://127.0.0.1:8080/login`
- при наличии API-токена — эндпоинты REST из документации Defect Dojo (для честного сравнения лучше один и тот же тип работы: **GET страницы** или **GET API** в обоих стендах).

## Изоляция от ASOC

Чтобы исключить взаимное влияние:

- остановите контейнеры / Helm-релизы ASOC, **или**
- в Kubernetes:  
  `kubectl scale deployment -n asoc --all --replicas=0`  
  (или `kubectl delete namespace asoc` — тяжелее по восстановлению).

Память/CPU на хосте должны позволять Defect Dojo (обычно Postgres + Redis + Celery + uwsgi — выше базового ASOC).

## HPA и мониторинг для Defect Dojo

Аналог ASOC можно настроить вручную:

- **Deployment** нескольких реплик uwsgi/web и **HPA** по CPU ~90% при заданных `resources.requests`, если чарт/манифест это поддерживает.
- Те же **Prometheus/Grafana** в namespace `monitoring` уже увидят новый Deployment в другом namespace — добавьте в Grafana панели с `namespace="<ns-defectdojo>"` или импортируйте общий Kubernetes dashboard.

Для упрощённого сравнения достаточно зафиксировать в отчёте **деградацию по wrk** (латентность, ошибки) без HPA на стороне DD.
