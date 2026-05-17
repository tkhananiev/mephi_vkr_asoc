#!/usr/bin/env bash
# Выкладка на Kubernetes-стенд (namespace asoc): kubectl apply (Kustomize) + образы + restart.
# Все Deployments намеренно с strategy: Recreate (без второго ReplicaSet-пода при обновлении — важно для RWO/PVC и чистых перезапусков).
# Использование (из любого каталога):
#   ./deploy/scripts/rollout-stand.sh           # все приложения из workloads + web (в т.ч. reference-data, jira)
#   ./deploy/scripts/rollout-stand.sh api       # только api-service
#   ./deploy/scripts/rollout-stand.sh auth    # только auth-service
#   ./deploy/scripts/rollout-stand.sh processing  # только processing-service
#   ./deploy/scripts/rollout-stand.sh reference-data  # только reference-data-service
#
# Переменные:
#   REGISTRY=cr.selcloud.ru/atomic-asoc  — префикс образов (как в workloads.yaml / frontend.yaml)
#   IMAGE_TAG=latest                     — тег для docker tag/push (по умолчанию как в манифестах)
#   PLATFORM=linux/amd64                 — ноды кластера amd64
#   FRONT_DIR=.../mephi_vkr_asoc_front — каталог фронта (рядом с репо по умолчанию)
#   SKIP_KUSTOMIZE_APPLY=1          — не делать kubectl apply (только сборка образов и rollout restart)
#
# В начале и в конце работы скрипт выводит, в какой registry уходят образы (${REGISTRY}/…:${IMAGE_TAG}).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
REGISTRY="${REGISTRY:-cr.selcloud.ru/atomic-asoc}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
FRONT_DIR="${FRONT_DIR:-${REPO_ROOT}/../mephi_vkr_asoc_front}"
NS="${K8S_NAMESPACE:-asoc}"

print_push_target() {
  local host="${REGISTRY%%/*}"
  local project="${REGISTRY#*/}"
  echo ""
  echo "━━ Образы: куда push ━━"
  echo "    Registry (хост):     ${host}"
  echo "    Проект / префикс:    ${project}"
  echo "    Полный шаблон тега:  ${REGISTRY}/<имя сервиса>:${IMAGE_TAG}"
  echo "    Пример (web):        ${REGISTRY}/web:${IMAGE_TAG}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
}

if [[ ! -f "${REPO_ROOT}/services/api-service/Dockerfile" ]]; then
  echo "Не найден api Dockerfile. REPO_ROOT=${REPO_ROOT}" >&2
  exit 1
fi

apply_manifests() {
  if [[ "${SKIP_KUSTOMIZE_APPLY:-}" == "1" ]]; then
    echo "==> пропуск kubectl apply (SKIP_KUSTOMIZE_APPLY=1)"
    return 0
  fi
  echo "==> kubectl apply (полный каталог через Kustomize, см. deploy/k8s/README.md)"
  kubectl kustomize "${REPO_ROOT}/deploy/k8s" --load-restrictor=LoadRestrictionsNone | kubectl apply -f -
}

docker_build_push_reference_data() {
  echo "==> docker build ${REGISTRY}/reference-data:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/reference-data-service/Dockerfile" \
    -t "${REGISTRY}/reference-data:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/reference-data:${IMAGE_TAG}"
  docker push "${REGISTRY}/reference-data:${IMAGE_TAG}"
}

docker_build_push_jira_mock() {
  echo "==> docker build ${REGISTRY}/jira-mock:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/jira-mock/Dockerfile" \
    -t "${REGISTRY}/jira-mock:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/jira-mock:${IMAGE_TAG}"
  docker push "${REGISTRY}/jira-mock:${IMAGE_TAG}"
}

docker_build_push_jira_integration() {
  echo "==> docker build ${REGISTRY}/jira-integration:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/jira-integration-service/Dockerfile" \
    -t "${REGISTRY}/jira-integration:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/jira-integration:${IMAGE_TAG}"
  docker push "${REGISTRY}/jira-integration:${IMAGE_TAG}"
}

docker_build_push_api() {
  echo "==> docker build ${REGISTRY}/api:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/api-service/Dockerfile" \
    -t "${REGISTRY}/api:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/api:${IMAGE_TAG}"
  docker push "${REGISTRY}/api:${IMAGE_TAG}"
}

docker_build_push_auth() {
  echo "==> docker build ${REGISTRY}/auth:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/auth-service/Dockerfile" \
    -t "${REGISTRY}/auth:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/auth:${IMAGE_TAG}"
  docker push "${REGISTRY}/auth:${IMAGE_TAG}"
}

docker_build_push_semgrep() {
  echo "==> docker build ${REGISTRY}/semgrep:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/semgrep-service/Dockerfile" \
    -t "${REGISTRY}/semgrep:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/semgrep:${IMAGE_TAG}"
  docker push "${REGISTRY}/semgrep:${IMAGE_TAG}"
}

docker_build_push_gitleaks() {
  echo "==> docker build ${REGISTRY}/gitleaks:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/gitleaks-service/Dockerfile" \
    -t "${REGISTRY}/gitleaks:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/gitleaks:${IMAGE_TAG}"
  docker push "${REGISTRY}/gitleaks:${IMAGE_TAG}"
}

docker_build_push_generic_scan_runner() {
  echo "==> docker build ${REGISTRY}/generic-scan-runner:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/generic-scan-runner/Dockerfile" \
    -t "${REGISTRY}/generic-scan-runner:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/generic-scan-runner:${IMAGE_TAG}"
  docker push "${REGISTRY}/generic-scan-runner:${IMAGE_TAG}"
}

docker_build_push_processing() {
  echo "==> docker build ${REGISTRY}/processing:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" \
    -f "${REPO_ROOT}/services/processing-service/Dockerfile" \
    -t "${REGISTRY}/processing:${IMAGE_TAG}" \
    "${REPO_ROOT}"
  echo "==> docker push ${REGISTRY}/processing:${IMAGE_TAG}"
  docker push "${REGISTRY}/processing:${IMAGE_TAG}"
}

docker_build_push_web() {
  if [[ ! -f "${FRONT_DIR}/Dockerfile" ]]; then
    echo "Каталог фронта не найден: ${FRONT_DIR}. Задай FRONT_DIR=..." >&2
    exit 1
  fi
  echo "==> docker build ${REGISTRY}/web:${IMAGE_TAG} (${PLATFORM})"
  docker build --platform "${PLATFORM}" -t "${REGISTRY}/web:${IMAGE_TAG}" "${FRONT_DIR}"
  echo "==> docker push ${REGISTRY}/web:${IMAGE_TAG}"
  docker push "${REGISTRY}/web:${IMAGE_TAG}"
}

rollout_all() {
  echo "==> rollout restart: сначала reference-data (Recreate + PVC), затем остальное (-n ${NS})"
  kubectl -n "${NS}" rollout restart deployment/reference-data-service
  kubectl -n "${NS}" rollout status deployment/reference-data-service --timeout=300s

  kubectl -n "${NS}" rollout restart \
    deployment/api-service \
    deployment/auth-service \
    deployment/jira-integration-service \
    deployment/jira-mock \
    deployment/asoc-web \
    deployment/semgrep-service \
    deployment/gitleaks-service \
    deployment/generic-scan-runner \
    deployment/processing-service

  kubectl -n "${NS}" rollout status deployment/api-service --timeout=240s
  kubectl -n "${NS}" rollout status deployment/auth-service --timeout=240s
  kubectl -n "${NS}" rollout status deployment/jira-integration-service --timeout=240s
  kubectl -n "${NS}" rollout status deployment/jira-mock --timeout=240s
  kubectl -n "${NS}" rollout status deployment/asoc-web --timeout=240s
  kubectl -n "${NS}" rollout status deployment/semgrep-service --timeout=240s
  kubectl -n "${NS}" rollout status deployment/gitleaks-service --timeout=240s
  kubectl -n "${NS}" rollout status deployment/generic-scan-runner --timeout=240s
  kubectl -n "${NS}" rollout status deployment/processing-service --timeout=240s
  echo "==> готово. Проверка: curl -sS https://atomic-asoc.ru/health"
}

TARGET="${1:-all}"

print_push_target

apply_manifests

case "${TARGET}" in
  api)
    docker_build_push_api
    kubectl -n "${NS}" rollout restart deployment/api-service
    kubectl -n "${NS}" rollout status deployment/api-service --timeout=240s
    ;;
  auth)
    docker_build_push_auth
    kubectl -n "${NS}" rollout restart deployment/auth-service
    kubectl -n "${NS}" rollout status deployment/auth-service --timeout=240s
    ;;
  web)
    docker_build_push_web
    kubectl -n "${NS}" rollout restart deployment/asoc-web
    kubectl -n "${NS}" rollout status deployment/asoc-web --timeout=240s
    ;;
  semgrep)
    docker_build_push_semgrep
    kubectl -n "${NS}" rollout restart deployment/semgrep-service
    kubectl -n "${NS}" rollout status deployment/semgrep-service --timeout=240s
    ;;
  gitleaks)
    docker_build_push_gitleaks
    kubectl -n "${NS}" rollout restart deployment/gitleaks-service
    kubectl -n "${NS}" rollout status deployment/gitleaks-service --timeout=240s
    ;;
  generic-scan-runner|runner)
    docker_build_push_generic_scan_runner
    kubectl -n "${NS}" rollout restart deployment/generic-scan-runner
    kubectl -n "${NS}" rollout status deployment/generic-scan-runner --timeout=240s
    ;;
  processing)
    docker_build_push_processing
    kubectl -n "${NS}" rollout restart deployment/processing-service
    kubectl -n "${NS}" rollout status deployment/processing-service --timeout=240s
    ;;
  reference-data|ref-data)
    docker_build_push_reference_data
    kubectl -n "${NS}" rollout restart deployment/reference-data-service
    kubectl -n "${NS}" rollout status deployment/reference-data-service --timeout=300s
    ;;
  jira-mock)
    docker_build_push_jira_mock
    kubectl -n "${NS}" rollout restart deployment/jira-mock
    kubectl -n "${NS}" rollout status deployment/jira-mock --timeout=240s
    ;;
  jira-integration|jira)
    docker_build_push_jira_integration
    kubectl -n "${NS}" rollout restart deployment/jira-integration-service
    kubectl -n "${NS}" rollout status deployment/jira-integration-service --timeout=240s
    ;;
  all)
    docker_build_push_reference_data
    docker_build_push_jira_mock
    docker_build_push_jira_integration
    docker_build_push_api
    docker_build_push_auth
    docker_build_push_web
    docker_build_push_semgrep
    docker_build_push_gitleaks
    docker_build_push_generic_scan_runner
    docker_build_push_processing
    rollout_all
    ;;
  *)
    echo "usage: $0 [api|auth|web|semgrep|gitleaks|generic-scan-runner|processing|reference-data|jira-mock|jira-integration|all]" >&2
    exit 1
    ;;
esac

echo ""
echo "Готово. Образы отправлены в registry «${REGISTRY%%/*}», проект «${REGISTRY#*/}», тег «${IMAGE_TAG}» (${REGISTRY}/api:${IMAGE_TAG}, ${REGISTRY}/web:${IMAGE_TAG}, …)."
echo ""
echo "Авторизация: auth-service (PostgreSQL, схема authn), вход пользователя POST /auth/v1/login; JWT проверяется api-service тем же jwt-secret."
