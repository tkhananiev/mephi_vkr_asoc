# Мониторинг: Prometheus + Grafana

Стек **kube-prometheus-stack**: метрики подов, Deployment/HPA, дашборды ASOC и (опционально) Defect Dojo.

## Предпосылки

1. **metrics-server** в кластере (`kubectl top pods`).
2. **Helm 3**, репозиторий prometheus-community.

## Установка

Из корня `mephi_vkr_asoc`:

```bash
./deploy/k8s/monitoring/install.sh
```

Пароль Grafana:

```bash
kubectl get secret -n monitoring kps-grafana -o jsonpath='{.data.admin-password}' | base64 -d && echo
```

Port-forward:

```bash
kubectl port-forward -n monitoring svc/kps-grafana 3000:80
```

## Ingress

DNS **grafana.atomic-asoc.ru** на IP Ingress Controller, затем:

```bash
kubectl apply -f deploy/k8s/monitoring/ingress-grafana.yaml
```

## Дашборды

```bash
kubectl apply -k deploy/k8s/monitoring
```

| Дашборд | UID |
|---------|-----|
| ASOC — нагрузка и масштабирование | `asoc-loadtest-hpa` |
| Defect Dojo — нагрузка и масштабирование | `defectdojo-loadtest-hpa` |

Импорт вручную: JSON в каталоге `deploy/k8s/monitoring/`.

Метрики: `container_cpu_usage_seconds_total`, `kube_deployment_status_replicas`, `kube_horizontalpodautoscaler_*`. Namespace по умолчанию — `asoc`.

## HPA

Порог масштабирования api-service и processing-service — **90%** CPU от `requests` (`deploy/k8s/hpa.yaml`).

## Удаление

```bash
./deploy/k8s/monitoring/uninstall.sh
```
