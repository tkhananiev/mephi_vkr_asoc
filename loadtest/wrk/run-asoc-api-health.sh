#!/usr/bin/env bash
# Нагрузка на api-service (лёгкий GET /health). Аргумент: полный URL (по умолчанию локальный port-forward).
set -euo pipefail
BASE="${1:-http://127.0.0.1:8080/health}"
THREADS="${THREADS:-8}"
CONN="${CONN:-400}"
DURATION="${DURATION:-120s}"

echo "wrk -t$THREADS -c$CONN -d $DURATION → $BASE"
exec wrk -t"$THREADS" -c"$CONN" -d"$DURATION" --latency "$BASE"
