#!/usr/bin/env bash
# Однократно для уже существующей БД: колонка core.processing_runs.console_product_id (см. migrations/017_*.sql).
# Init-контейнер Postgres не перезапускает initdb после первого тома — миграции из ConfigMap нужно подавать вручную.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
NS="${K8S_NAMESPACE:-asoc}"
SQL="${REPO_ROOT}/migrations/017_processing_run_console_product.sql"
if [[ ! -f "$SQL" ]]; then
  echo "Нет файла: $SQL" >&2
  exit 1
fi
echo "==> Применение 017 к БД asoc в namespace $NS ..."
kubectl -n "${NS}" exec -i statefulset/postgres -- psql -U asoc -d asoc -v ON_ERROR_STOP=1 -f /dev/stdin <"$SQL"
echo "==> Готово. Проверка: наличие колонки."
kubectl -n "${NS}" exec statefulset/postgres -- psql -U asoc -d asoc -tAc \
  "SELECT column_name FROM information_schema.columns WHERE table_schema='core' AND table_name='processing_runs' AND column_name='console_product_id';"
