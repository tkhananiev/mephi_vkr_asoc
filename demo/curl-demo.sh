#!/usr/bin/env bash

set -euo pipefail

echo "[1/4] health"
curl -s http://localhost:8080/health && echo
curl -s http://localhost:8081/health && echo
curl -s http://localhost:8082/health && echo
curl -s http://localhost:8083/health && echo
curl -s http://localhost:8090/health && echo

echo "[2/4] sync NVD"
curl -s -X POST "http://localhost:8081/api/v1/sync/nvd?cve_id=CVE-2021-44228" && echo

echo "[3/4] sync BDU"
curl -s -X POST "http://localhost:8081/api/v1/sync/bdu" && echo

echo "[4/4] run semgrep flow"
curl -s -X POST "http://localhost:8080/api/v1/scans/semgrep" \
  -H "Content-Type: application/json" \
  -d '{
    "target_path": "/app/demo/vulnerable-app",
    "scanner_name": "semgrep"
  }' && echo
