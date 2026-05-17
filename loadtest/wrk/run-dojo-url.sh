#!/usr/bin/env bash
# Нагрузка на Defect Dojo (или любой URL). Пример: ./run-dojo-url.sh http://127.0.0.1:8081/login
set -euo pipefail
BASE="${1:?usage: $0 <base-url>}"
THREADS="${THREADS:-8}"
CONN="${CONN:-200}"
DURATION="${DURATION:-120s}"

echo "wrk -t$THREADS -c$CONN -d $DURATION → $BASE"
exec wrk -t"$THREADS" -c"$CONN" -d"$DURATION" --latency "$BASE"
