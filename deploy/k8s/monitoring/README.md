# Мониторинг: Prometheus + Grafana (нагрузочное тестирование / HPA)

Цель: снимать **скриншоты** с ростом нагрузки по CPU и **числом подов** при срабатывании HPA (`deploy/k8s/hpa.yaml`, порог **90%** CPU от `requests`).

## Предпосылки в кластере

1. **metrics-server** — нужен для `kubectl top pods` и для метрик CPU в HPA:
   - minikube: `minikube addons enable metrics-server`
   - kind / kubeadm: установите [metrics-server](https://github.com/kubernetes-sigs/metrics-server) по инструкции дистрибутива
2. **Helm 3** и репозиторий prometheus-community.

## Установка kube-prometheus-stack (рекомендуется)

Даёт Prometheus, Grafana, **kube-state-metrics** (реплики Deployment/HPA), node-exporter, целевые правила.

**Из корня репозитория** `mephi_vkr_asoc`:

```bash
./deploy/k8s/monitoring/install.sh
```

Скрипт: добавляет Helm-репозиторий, выполняет `helm upgrade --install` релиза **`kps`** в namespace **`monitoring`** с файлом `kube-prometheus-values.yaml`, ждёт готовность (`--wait`, до 15 минут).

Без скрипта то же вручную:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  -f deploy/k8s/monitoring/kube-prometheus-values.yaml \
  --wait --timeout 15m
```

Пароль администратора Grafana:

```bash
kubectl get secrets -n monitoring -o name | grep grafana
# затем, например:
kubectl get secret -n monitoring kps-grafana -o jsonpath='{.data.admin-password}' | base64 -d && echo
```

(имя Secret зависит от имени Helm-релиза; подставь своё из первой команды.)

Проброс Grafana на localhost (имя сервиса может отличаться — смотри вывод `kubectl get svc -n monitoring`):

```bash
kubectl get svc -n monitoring | grep -i grafana
kubectl port-forward -n monitoring svc/<ИМЯ_SVC_GRAFANA> 3000:80
```

Открой `http://localhost:3000` (user `admin`).

### Публичный доступ (Ingress)

1. Укажи **DNS**: запись **A** (или AAAA) для **`grafana.atomic-asoc.ru`** на внешний IP того же **Ingress Controller**, что обслуживает ASOC.
2. Применяй манифест (отдельно от `kubectl apply -k deploy/k8s`, т.к. Ingress в namespace **`monitoring`**):

   ```bash
   kubectl apply -f deploy/k8s/monitoring/ingress-grafana.yaml
   ```

3. После изменения домена перекати Grafana с обновлёнными values (поле `grafana.ini.server.root_url` в `kube-prometheus-values.yaml`):

   ```bash
   ./deploy/k8s/monitoring/install.sh
   ```

Открой **https://grafana.atomic-asoc.ru/** (TLS выдаст **cert-manager**, ClusterIssuer `letsencrypt-prod` — как у основного Ingress).

## Импорт дашборда ASOC

**Автоматически (рекомендуется):** ConfigMap с лейблом `grafana_dashboard` подхватывает sidecar Grafana.

```bash
kubectl apply -k deploy/k8s/monitoring
```

Через ~30–60 с в UI появится дашборд **«ASOC — нагрузка и масштабирование»** (папка зависит от настроек sidecar, часто корень «General» или «default»).

Вручную: **Dashboards → Import** → загрузить файл `grafana-dashboard-asoc-loadtest.json` или вставить содержимое.

Если панели **No data**, в настройках дашборда укажи datasource **Prometheus** (в стеке его UID обычно `prometheus`; иначе выбери свой в выпадающем списке для каждой панели или сделай переменную datasource).

В панелях используются метики:

- **cAdvisor / kubelet**: `container_cpu_usage_seconds_total`, память подов
- **kube-state-metrics**: `kube_deployment_status_replicas{namespace="asoc"}`, `kube_horizontalpodautoscaler_*`

Если запросы пустые — проверь, что namespace **`asoc`** совпадает и поды имеют имена `api-service-…`, `processing-service-…`.

## Ingress Nginx (опционально, RPS)

Если нагрузка идёт через **ingress-nginx**, в stack уже часто собираются метики `nginx_ingress_controller_requests` — на дашборде включена опциональная панель «Ingress RPS».

## Соответствие НФТ

Зафиксируйте в отчёте (см. `loadtest/LOAD_TEST_REPORT.md`): при **средней утилизации CPU ≥ 90%** от `requests` HPA увеличивает реплики; порог **90%** задан в `hpa.yaml` как целевое значение масштабирования по CPU.

## Удаление

```bash
./deploy/k8s/monitoring/uninstall.sh
# или: helm uninstall kps -n monitoring
```
