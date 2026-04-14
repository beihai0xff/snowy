#!/usr/bin/env bash
# 运行测试
set -euo pipefail

echo "▸ Running unit tests..."
make test

if [ "${1:-}" = "--integration" ]; then
    echo "▸ Running integration tests..."
    make test-integration
fi

