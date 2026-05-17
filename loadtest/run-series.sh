#!/usr/bin/env bash
# Ступенчатый прогон wrk: одни и те же TARGET_URL / THREADS / DURATION / RAMP — для сравнимых логов ASOC vs Defect Dojo.
# См. METHODOLOGY.md
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STAMP="$(date +%Y%m%d-%H%M%S)"
RESULTS="${RESULTS_DIR:-$ROOT/results/$STAMP}"

TARGET_URL="${TARGET_URL:?Задайте TARGET_URL, например http://127.0.0.1:8080/health}"
THREADS="${THREADS:-8}"
DURATION_PER_STEP="${DURATION_PER_STEP:-30s}"
# Ступени соединений (пробелы); переопределение: export RAMP_CS='50 100 200 400'
RAMP_CS="${RAMP_CS:-50 100 200 400 600 800}"

command -v wrk >/dev/null || { echo "wrk не найден в PATH" >&2; exit 1; }

mkdir -p "$RESULTS"
META="$RESULTS/run-meta.txt"
{
  echo "stamp=$STAMP"
  echo "target=$TARGET_URL"
  echo "threads=$THREADS"
  echo "duration_per_step=$DURATION_PER_STEP"
  echo "ramp=$RAMP_CS"
  echo "host=$(hostname 2>/dev/null || true)"
  date -u +"%Y-%m-%dT%H:%M:%SZ"
} | tee "$META"

SUMMARY="$RESULTS/summary.txt"
: >"$SUMMARY"

for c in $RAMP_CS; do
  OUT="$RESULTS/wrk-c${c}.txt"
  {
    echo "=============================================="
    echo "wrk -t${THREADS} -c${c} -d${DURATION_PER_STEP} --latency"
    echo "$TARGET_URL"
    echo "=============================================="
  } | tee "$OUT"
  # shellcheck disable=SC2086
  if wrk -t"${THREADS}" -c"${c}" -d"${DURATION_PER_STEP}" --latency "${TARGET_URL}" 2>&1 | tee -a "$OUT"; then
    echo "c=$c status=ok" >>"$SUMMARY"
  else
    echo "c=$c status=fail(nonzero wrk exit)" | tee -a "$SUMMARY"
  fi
  echo "" >>"$SUMMARY"
done

echo ""
echo "Готово. Логи: $RESULTS"
echo "Краткая сводка: $SUMMARY"
