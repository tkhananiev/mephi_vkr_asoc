#!/usr/bin/env bash

set -euo pipefail

echo "[1/5] health"
curl -s http://localhost:8080/health && echo
curl -s http://localhost:8081/health && echo
curl -s http://localhost:8082/health && echo
curl -s http://localhost:8083/health && echo
curl -s http://localhost:8090/health && echo

echo "[2/5] sync NVD"
curl -s -X POST "http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228" && echo

echo "[3/5] sync BDU"
curl -s -X POST "http://localhost:8081/api/v1/sync/bdu" && echo

echo "[4/5] clone WebGoat if missing"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -x "$SCRIPT_DIR/scan-targets/clone-webgoat.sh" ]; then
  "$SCRIPT_DIR/scan-targets/clone-webgoat.sh"
else
  echo "skip: clone-webgoat.sh not found"
fi

echo "[5/5] run semgrep flow (defaults: WebGoat + p/java from APP_DEFAULT_*)"
curl -s -X POST "http://localhost:8080/api/v1/scans/semgrep" \
  -H "Content-Type: application/json" \
  -d '{"scanner_name":"semgrep"}' && echo
