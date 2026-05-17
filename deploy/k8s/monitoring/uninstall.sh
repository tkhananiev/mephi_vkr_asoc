#!/usr/bin/env bash
set -euo pipefail
RELEASE_NAME="${RELEASE_NAME:-kps}"
NAMESPACE="${NAMESPACE:-monitoring}"
helm uninstall "$RELEASE_NAME" -n "$NAMESPACE" || true
echo "Если namespace $NAMESPACE больше не нужен: kubectl delete namespace $NAMESPACE"
