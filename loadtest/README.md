# Нагрузочное тестирование ASOC vs Defect Dojo

**Единая методика (фазы, параметры, сравнение без «убийства» Ingress):** **`METHODOLOGY.md`**.  
**Ступенчатый прогон wrk с логами в `results/`:** `./run-series.sh`.

**Цель:** сравнить **поведение приложений** (ASOC и Defect Dojo) под нагрузкой по NFR — выявить точку деградации **прикладного** контура и зафиксировать различия в отчёте.  
**Про Ingress:** намеренно **не доводить ingress-nginx до предела** — иначе замеряется «кто упёрся в прокси», а не качество сервисов. Для чистого сравнения предпочтительно грузить **напрямую бэкенд** (например `kubectl port-forward` на `api-service` / `processing-service` на localhost и wrk на `127.0.0.1`), либо следить за метриками Ingress и снижать `-c`, если узким местом становится контроллер. Сценарий «снаружи через публичный URL» оставьте для иллюстрации end-to-end, но для сопоставления с Defect Dojo зафиксируйте **одинаковый тип доступа** (оба с порт-forward / оба через внешний вход).

Инструменты: **[wrk](https://github.com/wg/wrk)** (или `hey`, `vegeta`). Скрипты ниже предполагают установленный `wrk` в `PATH`.

## Быстрый старт (ASOC, Kubernetes)

1. Развернуть ASOC (`deploy/k8s`), **metrics-server**, **HPA** (`hpa.yaml` уже в kustomize).
2. Поднять **Prometheus + Grafana** — `deploy/k8s/monitoring/README.md`, импорт `grafana-dashboard-asoc-loadtest.json`.
3. В отдельном терминале — проброс цели нагрузки:
   - только **api-service**:  
     `kubectl port-forward -n asoc svc/api-service 8080:8080`
   - только **processing** (нагрузка на БД):  
     `kubectl port-forward -n asoc svc/processing-service 8082:8082`
   - через **Ingress** (как в проде): используйте базовый URL сайта и путь `/health` или `/api/...` (см. фронтовой nginx).

4. Запустить сценарий из таблицы отчёта:  
   `./wrk/run-asoc-api-health.sh http://127.0.0.1:8080/health`  
   `./wrk/run-asoc-processing-groups.sh http://127.0.0.1:8082/api/v1/groups?limit=50`

5. В Grafana снять скриншоты: виджеты CPU + **реплики HPA** во время роста `-c` (потоков/соединений wrk).

### Наработка до деградации

Увеличивайте нагрузку ступенями (пример):

```bash
for c in 50 100 200 400 800 1200; do
  echo "=== connections $c ==="
  wrk -t 12 -c "$c" -d 30s --latency http://127.0.0.1:8080/health || break
done
```

Критерий деградации (зафиксируйте в отчёте): доля **5xx** \> 1%, рост **латентности p99** \> порога, **таймауты**, ошибки wrk, **OOMKilled** подов (`kubectl describe pod`).

**НФТ по масштабированию:** целевой порог CPU для HPA — **90%** от `requests` (`deploy/k8s/hpa.yaml`).

---

## Defect Dojo (шаг 2)

См. **`defectdojo/README.md`**: отдельный compose/Helm, при чистом эксперименте можно **остановить** namespace ASOC (`kubectl scale deployment --all --replicas=0 -n asoc`) или не поднимать его.

Сравнение «при равных нагрузках» — те же значения **`-t` / `-c` / duration** wrk против выбранной страницы DD (например `/login` или API при наличии токена).

---

## Отчёт

Шаблон с тест-кейсами и таблицей сравнения: **`LOAD_TEST_REPORT.md`**. Методика прогонов: **`METHODOLOGY.md`**.
