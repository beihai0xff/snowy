#!/usr/bin/env bash
# 运行测试：默认单元测试；--integration 时自动拉起 MySQL/Redis/OpenSearch/MinIO Docker 依赖后执行集成测试。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deployments/docker/docker-compose.yml"
PROJECT_NAME="${PROJECT_NAME:-snowy}"
DOCKER_COMPOSE=(docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME")
TEST_DEPS=(mysql redis opensearch minio)
RUN_INTEGRATION=false
KEEP_DEPS=false

for arg in "$@"; do
  case "$arg" in
    --integration)
      RUN_INTEGRATION=true
      ;;
    --keep-deps)
      KEEP_DEPS=true
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

wait_for_container() {
  local container_name="$1"
  local retries="${2:-60}"
  local delay="${3:-2}"
  local status=""

  for ((i=1; i<=retries; i++)); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_name" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      echo "✓ $container_name is $status"
      return 0
    fi
    sleep "$delay"
  done

  echo "✗ Timed out waiting for $container_name to become healthy (last status: ${status:-unknown})" >&2
  return 1
}

cleanup() {
  if [[ "$RUN_INTEGRATION" != "true" || "$KEEP_DEPS" == "true" ]]; then
    return 0
  fi

  echo "▸ Cleaning up test dependencies..."
  "${DOCKER_COMPOSE[@]}" stop "${TEST_DEPS[@]}" >/dev/null 2>&1 || true
  "${DOCKER_COMPOSE[@]}" rm -f "${TEST_DEPS[@]}" >/dev/null 2>&1 || true
}

trap cleanup EXIT

(cd "$ROOT_DIR" && make test-unit)

if [[ "$RUN_INTEGRATION" == "true" ]]; then
  echo "▸ Starting Docker test dependencies: ${TEST_DEPS[*]}..."
  (cd "$ROOT_DIR" && "${DOCKER_COMPOSE[@]}" up -d "${TEST_DEPS[@]}")

  wait_for_container "snowy-mysql"
  wait_for_container "snowy-redis"
  wait_for_container "snowy-opensearch" 90 2
  wait_for_container "snowy-minio"

  echo "▸ Running integration tests..."
  (
    cd "$ROOT_DIR"
    go test -race -count=1 -timeout 120s -tags=integration ./test/integration/...
  )
fi

