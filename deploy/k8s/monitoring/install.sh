#!/usr/bin/env bash
# Установка Prometheus + Grafana (kube-prometheus-stack) в namespace monitoring.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALUES_FILE="${SCRIPT_DIR}/kube-prometheus-values.yaml"
RELEASE_NAME="${RELEASE_NAME:-kps}"
NAMESPACE="${NAMESPACE:-monitoring}"
CHART="prometheus-community/kube-prometheus-stack"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Не найдено: $1. Установите Helm 3 и kubectl." >&2
    exit 1
  }
}

require_cmd helm
require_cmd kubectl
test -f "$VALUES_FILE" || { echo "Нет файла $VALUES_FILE" >&2; exit 1; }

echo ">>> Helm repo: prometheus-community"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
helm repo update prometheus-community

echo ">>> Установка/обновление $RELEASE_NAME в namespace $NAMESPACE"
helm upgrade --install "$RELEASE_NAME" "$CHART" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  -f "$VALUES_FILE" \
  --wait \
  --timeout 15m

echo ""
echo ">>> Готово. Поды и сервисы (namespace $NAMESPACE):"
kubectl get pods,svc -n "$NAMESPACE"

echo ""
GRAFANA_SVC=""
if GRAFANA_SVC=$(kubectl get svc -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) && test -n "${GRAFANA_SVC:-}"; then
  :
else
  GRAFANA_SVC=$(kubectl get svc -n "$NAMESPACE" | awk '/[Gg]rafana/ {print $1; exit}')
fi

echo "Grafana Service: ${GRAFANA_SVC:-найти: kubectl get svc -n $NAMESPACE}"
echo "Пароль admin (Helm-секрет grafana, не TLS):"
GRAFANA_SECRET="${GRAFANA_SECRET:-kps-grafana}"
if kubectl get secret -n "$NAMESPACE" "$GRAFANA_SECRET" -o jsonpath='{.data.admin-password}' &>/dev/null; then
  echo "  kubectl get secret $GRAFANA_SECRET -n $NAMESPACE -o jsonpath='{.data.admin-password}' | base64 -d && echo"
else
  echo "  kubectl get secret -n $NAMESPACE -l app.kubernetes.io/instance=$RELEASE_NAME -o name | head -1 | xargs -I{} kubectl get {} -n $NAMESPACE -o jsonpath='{.data.admin-password}' | base64 -d && echo"
fi

echo ""
echo "Проброс UI (пример):"
echo "  kubectl port-forward -n $NAMESPACE svc/${GRAFANA_SVC:-GRAFANA_SVC} 3000:80"
echo "Открой http://localhost:3000 (логин admin, пароль из секрета выше)."
echo ""
echo "Дашборды ASOC: импорт JSON → $SCRIPT_DIR/grafana-dashboard-asoc-loadtest.json"
