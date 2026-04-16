#!/usr/bin/env bash
# 代码质量检查
set -euo pipefail

echo "▸ Running fmt..."
make fmt

echo "▸ Running vet..."
make vet

echo "▸ Running lint..."
make lint

