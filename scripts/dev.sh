#!/usr/bin/env bash
# 本地开发一键启动：启动基础设施 + 运行 API 服务
set -euo pipefail

echo "▸ Starting infrastructure..."
make bootstrap

echo "▸ Starting API server..."
exec make run-api

