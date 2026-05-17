#!/usr/bin/env bash
# Скопировать vulxml.xml и vullist.xlsx из корня над mephi_vkr_asoc → под reference-data-service (PVC /bdu-import)
# и при необходимости перезапустить Job bdu-bulk.
#
# Расклад:  vkr_temp_arch/vulxml.xml
#           vkr_temp_arch/vullist.xlsx
#           vkr_temp_arch/mephi_vkr_asoc/deploy/scripts/этот_скрипт
#
# Использование:
#   export KUBECONFIG=...   # при необходимости
#   ./deploy/scripts/import-bdu-k8s-from-repo-root.sh [--only-copy | --only-job]
#
set -euo pipefail

NS="${NS:-asoc}"
REPO_ROOT="$(cd "$(dirname "$0")/../../../" && pwd)"
VUL="$REPO_ROOT/vulxml.xml"
XLS="$REPO_ROOT/vullist.xlsx"
ONLY_COPY=false
ONLY_JOB=false
for arg in "$@"; do
  case "$arg" in
    --only-copy) ONLY_COPY=true ;;
    --only-job) ONLY_JOB=true ;;
    *)
      echo "unknown arg: $arg" >&2
      exit 1
      ;;
  esac
done

if [[ "$ONLY_JOB" != true ]]; then
  if [[ ! -f "$VUL" ]] || [[ ! -f "$XLS" ]]; then
    echo "Ожидались файлы:" >&2
    echo "  $VUL" >&2
    echo "  $XLS" >&2
    exit 1
  fi
fi

kubectl -n "$NS" rollout status deployment/reference-data-service --timeout=180s >/dev/null
POD="$(kubectl -n "$NS" get pods -l app=reference-data-service -o jsonpath='{.items[0].metadata.name}')"
echo "namespace=$NS pod=$POD"

if [[ "$ONLY_JOB" != true ]]; then
  echo "копирую vullist.xlsx…"
  kubectl -n "$NS" cp "$XLS" "${POD}:/bdu-import/vullist.xlsx"
  echo "копирую vulxml.xml (долго при больших файлах)…"
  kubectl -n "$NS" cp "$VUL" "${POD}:/bdu-import/vulxml.xml"
  kubectl -n "$NS" exec "$POD" -- ls -lh /bdu-import/
fi

if [[ "$ONLY_COPY" == true ]]; then
  echo "остановились после копирования (--only-copy)"
  exit 0
fi

ROOT_K8S="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_K8S"
echo "запускаю Job bdu-bulk из deploy/k8s/bdu-bulk-job.yaml …"
kubectl -n "$NS" delete job bdu-bulk --ignore-not-found
kubectl apply -f deploy/k8s/bdu-bulk-job.yaml
JPOD="$(kubectl -n "$NS" get pods -l job-name=bdu-bulk -o jsonpath='{.items[0].metadata.name}')"
echo "триггер-под Job: $JPOD (ответ появится в логах по завершении POST — импорт может идти часы)"
echo "смотреть: kubectl -n $NS logs -f job/bdu-bulk"
