#!/usr/bin/env bash
# 等待指定 Docker 容器进入 healthy / running 状态。
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "Usage: $0 <container-name> [retries] [delay-seconds]" >&2
  exit 1
fi

container_name="$1"
retries="${2:-60}"
delay="${3:-2}"
last_state="unknown"

for ((i=1; i<=retries; i++)); do
  last_state="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_name" 2>/dev/null || echo missing)"
  if [[ "$last_state" == "healthy" || "$last_state" == "running" ]]; then
    echo "✓ $container_name is $last_state"
    exit 0
  fi

  echo "▸ waiting for $container_name ($i/$retries), current state: $last_state"
  sleep "$delay"
done

echo "✗ Timed out waiting for $container_name to become ready (last state: $last_state)" >&2
exit 1
