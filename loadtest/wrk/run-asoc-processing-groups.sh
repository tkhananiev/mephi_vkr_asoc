#!/usr/bin/env bash
# Нагрузка на processing-service: список групп (JSON, БД). Требуется port-forward на 8082.
set -euo pipefail
BASE="${1:-http://127.0.0.1:8082/api/v1/groups?limit=50}"
THREADS="${THREADS:-8}"
CONN="${CONN:-200}"
DURATION="${DURATION:-120s}"

echo "wrk -t$THREADS -c$CONN -d $DURATION → $BASE"
exec wrk -t"$THREADS" -c"$CONN" -d"$DURATION" --latency "$BASE"
