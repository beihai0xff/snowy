# ============================================================
#  Snowy Makefile — run `make help` for available targets
# ============================================================

# ── 项目元信息 ──────────────────────────────────────────────
PROJECT_NAME   := snowy
MODULE         := $(shell head -1 go.mod 2>/dev/null | awk '{print $$2}' || echo "github.com/snowy/snowy")
VERSION        := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# ── 路径 ────────────────────────────────────────────────────
ROOT_DIR       := $(shell pwd)
BIN_DIR        := $(ROOT_DIR)/bin
CMD_DIR        := $(ROOT_DIR)/cmd
DEPLOY_DIR     := $(ROOT_DIR)/deployments/docker
CONFIG_DIR     := $(ROOT_DIR)/configs

# ── Go 参数 ─────────────────────────────────────────────────
GO             := go
GOFLAGS        :=
LDFLAGS        := -s -w \
                  -X main.Version=$(VERSION) \
                  -X main.BuildTime=$(BUILD_TIME) \
                  -X main.Commit=$(COMMIT)
GOTEST_FLAGS   := -race -count=1 -timeout 120s
TEST_DEPS_SERVICES := mysql redis
INFRA_SERVICES := mysql redis

# ── Docker 参数 ─────────────────────────────────────────────
DOCKER_COMPOSE := docker compose -f $(DEPLOY_DIR)/docker-compose.yml -p $(PROJECT_NAME)

# Load local, git-ignored runtime secrets when present (e.g. OPENAI_API_KEY).
# This keeps `make docker-run` / `make run` usable without re-exporting env vars.
ifneq (,$(wildcard $(ROOT_DIR)/.env))
include $(ROOT_DIR)/.env
export
endif
DOCKER_REG     ?=
IMAGE_SERVER   := $(if $(DOCKER_REG),$(DOCKER_REG)/)$(PROJECT_NAME):$(VERSION)

# ── 工具 ────────────────────────────────────────────────────
GOLANGCI_LINT  := $(shell command -v golangci-lint 2>/dev/null)
GOLANGCI_CONFIG := $(ROOT_DIR)/.golangci.yml
GOLANGCI_FMT_CMD := golangci-lint fmt -c $(GOLANGCI_CONFIG)
GOLANGCI_RUN_CMD := golangci-lint run -c $(GOLANGCI_CONFIG) ./...
MYSQL_MIGRATE_CMD := $(GO) run ./cmd/migrate -config $(CONFIG_DIR)/config.yaml
WAIT_FOR_CONTAINER := bash $(ROOT_DIR)/scripts/wait-for-container.sh

# ── 数据库 (本地开发默认值) ─────────────────────────────────
DB_HOST        ?= 127.0.0.1
DB_PORT        ?= 3306
DB_USER        ?= snowy
DB_PASSWORD    ?= snowy_secret
DB_NAME        ?= snowy
DATABASE_URL   ?= mysql://$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)

# ── 颜色 ────────────────────────────────────────────────────
GREEN  := \033[0;32m
YELLOW := \033[0;33m
CYAN   := \033[0;36m
RESET  := \033[0m

# ============================================================
#  默认目标
# ============================================================
.DEFAULT_GOAL := help

# ============================================================
#  Build
# ============================================================

.PHONY: build build-server clean

## build: 编译默认单体服务二进制
build: build-server

## build-server: 编译统一服务二进制
build-server:
	@echo "$(GREEN)▸ Building snowy...$(RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/snowy $(CMD_DIR)/snowy
	@echo "$(GREEN)✓ $(BIN_DIR)/snowy$(RESET)"

## clean: 清理编译产物
clean:
	@echo "$(YELLOW)▸ Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR)
	@$(GO) clean -cache -testcache
	@echo "$(YELLOW)✓ Cleaned$(RESET)"

# ============================================================
#  Test & Lint
# ============================================================

.PHONY: test test-unit test-integration test-e2e test-coverage lint fmt vet test-deps-up test-deps-down ensure-golangci-lint

## test: 运行单元测试
test: test-unit

## test-unit: 运行单元测试
test-unit:
	@echo "$(GREEN)▸ Running unit tests...$(RESET)"
	$(GO) test $(GOTEST_FLAGS) ./internal/...

## test-integration: 启动 MySQL/Redis Docker 依赖并运行集成测试
test-integration:
	@echo "$(GREEN)▸ Running integration tests with Docker dependencies...$(RESET)"
	@bash ./scripts/test.sh --integration

## test-e2e: 运行端到端测试
test-e2e:
	@echo "$(GREEN)▸ Running e2e tests...$(RESET)"
	$(GO) test $(GOTEST_FLAGS) -tags=e2e ./test/e2e/...

## test-coverage: 生成测试覆盖率报告
test-coverage:
	@echo "$(GREEN)▸ Generating coverage report...$(RESET)"
	@mkdir -p $(BIN_DIR)
	$(GO) test $(GOTEST_FLAGS) -coverprofile=$(BIN_DIR)/coverage.out ./internal/...
	$(GO) tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "$(GREEN)✓ Coverage report: $(BIN_DIR)/coverage.html$(RESET)"

## ensure-golangci-lint: 确保 golangci-lint 已安装
ensure-golangci-lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)▸ Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

## lint: 运行 golangci-lint
lint: ensure-golangci-lint
	@echo "$(GREEN)▸ Running linter...$(RESET)"
	$(GOLANGCI_RUN_CMD)

## fmt: 格式化代码
fmt: ensure-golangci-lint
	@echo "$(GREEN)▸ Formatting code...$(RESET)"
	$(GOLANGCI_FMT_CMD)
	@echo "$(GREEN)✓ Formatted$(RESET)"

## vet: 静态分析
vet:
	@echo "$(GREEN)▸ Running vet...$(RESET)"
	$(GO) vet ./...

## test-deps-up: 启动测试所需 Docker 依赖 (MySQL/Redis)
test-deps-up:
	@echo "$(CYAN)▸ Starting test dependencies: $(TEST_DEPS_SERVICES)...$(RESET)"
	$(DOCKER_COMPOSE) up -d $(TEST_DEPS_SERVICES)

## test-deps-down: 停止测试所需 Docker 依赖 (MySQL/Redis)
test-deps-down:
	@echo "$(YELLOW)▸ Stopping test dependencies: $(TEST_DEPS_SERVICES)...$(RESET)"
	-$(DOCKER_COMPOSE) stop $(TEST_DEPS_SERVICES)
	-$(DOCKER_COMPOSE) rm -f $(TEST_DEPS_SERVICES)

# ============================================================
#  Docker — 基础设施 (docker-compose)
# ============================================================

.PHONY: docker-up docker-down docker-ps docker-logs docker-observability-up docker-clean bootstrap

## docker-up: 启动必需基础设施 (MySQL/Redis) 并等待健康检查通过后自动执行 GORM migration
docker-up:
	@echo "$(CYAN)▸ Starting infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) up -d $(INFRA_SERVICES)
	@echo "$(CYAN)▸ Waiting for infrastructure health checks...$(RESET)"
	@$(WAIT_FOR_CONTAINER) snowy-mysql 90 2
	@$(WAIT_FOR_CONTAINER) snowy-redis 60 2
	@$(MAKE) migrate-up
	@echo "$(CYAN)✓ Required infrastructure is healthy and MySQL schema is migrated$(RESET)"
	@echo ""
	@echo "  MySQL      : localhost:3306"
	@echo "  Redis      : localhost:6379"

## docker-down: 停止全部基础设施 (保留数据卷)
docker-down:
	@echo "$(YELLOW)▸ Stopping infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) down
	@echo "$(YELLOW)✓ Infrastructure stopped$(RESET)"

## docker-ps: 查看基础设施状态
docker-ps:
	$(DOCKER_COMPOSE) ps

## docker-logs: 查看基础设施日志（示例 make docker-logs SVC=redis）
docker-logs:
	$(DOCKER_COMPOSE) logs -f $(SVC)

## docker-observability-up: 启动可选观测组件 (Prometheus/Grafana)
docker-observability-up:
	@echo "$(CYAN)▸ Starting optional observability services...$(RESET)"
	$(DOCKER_COMPOSE) up -d prometheus grafana
	@echo "$(CYAN)✓ Observability services are running$(RESET)"
	@echo ""
	@echo "  Prometheus : localhost:9090"
	@echo "  Grafana    : localhost:3000"

## docker-clean: 停止全部基础设施并删除数据卷 (⚠️ 数据将丢失)
docker-clean:
	@echo "$(YELLOW)▸ Destroying infrastructure and volumes...$(RESET)"
	$(DOCKER_COMPOSE) down -v --remove-orphans
	@echo "$(YELLOW)✓ Infrastructure destroyed$(RESET)"

## bootstrap: 下载依赖并启动基础设施，等待健康后自动迁移
bootstrap: deps docker-up

# ============================================================
#  Docker — 应用镜像构建 & 运行
# ============================================================

.PHONY: docker-build docker-build-server docker-build-web docker-run docker-smoke docker-push

## docker-build: 构建默认单体服务与前端镜像
docker-build: docker-build-server docker-build-web

## docker-build-server: 构建统一服务镜像
docker-build-server:
	@echo "$(CYAN)▸ Building Docker image: $(IMAGE_SERVER)...$(RESET)"
	docker build \
		--build-arg TARGET=snowy \
		-f $(DEPLOY_DIR)/Dockerfile \
		-t $(IMAGE_SERVER) \
		-t $(PROJECT_NAME):latest \
		$(ROOT_DIR)
	@echo "$(CYAN)✓ $(IMAGE_SERVER)$(RESET)"

## docker-build-web: 构建前端 Nginx 服务镜像
docker-build-web:
	@echo "$(CYAN)▸ Building Docker image: $(PROJECT_NAME)-web:latest...$(RESET)"
	docker build \
		-f $(DEPLOY_DIR)/Dockerfile.web \
		-t $(PROJECT_NAME)-web:latest \
		$(ROOT_DIR)
	@echo "$(CYAN)✓ $(PROJECT_NAME)-web:latest$(RESET)"

## docker-run: 通过 docker compose 一键启动统一服务与 Web（会先确保基础设施与迁移完成）
docker-run: docker-up
	@if [ -z "$${OPENAI_API_KEY:-}" ]; then \
		echo "$(YELLOW)✗ OPENAI_API_KEY is required for real LLM calls.$(RESET)"; \
		echo "  Usage: OPENAI_API_KEY='<runtime only>' make docker-run"; \
		exit 1; \
	fi
	@echo "$(GREEN)▸ Starting Snowy and Web services...$(RESET)"
	$(DOCKER_COMPOSE) up -d snowy snowy-web
	@echo "$(GREEN)✓ Services are running$(RESET)"
	@echo ""
	@echo "  API    : http://localhost:8080"
	@echo "  Web    : http://localhost:3001"

## docker-smoke: 纯 Docker 运行态冒烟检查
docker-smoke:
	@bash ./scripts/docker-smoke.sh

## docker-push: 推送应用镜像到远端仓库 (需设置 DOCKER_REG)
docker-push:
ifndef DOCKER_REG
	$(error DOCKER_REG is not set. Usage: make docker-push DOCKER_REG=your-registry.com)
endif
	docker push $(IMAGE_SERVER)

# ============================================================
#  Run — 本地开发运行
# ============================================================

.PHONY: dev run web-dev web-build


## dev: 一键开发（先 make bootstrap，再本地运行统一服务）
dev: bootstrap
	@echo "$(GREEN)▸ Running Snowy service locally...$(RESET)"
	$(GO) run $(CMD_DIR)/snowy

## run: 本地运行统一服务
run:
	@echo "$(GREEN)▸ Running Snowy service locally...$(RESET)"
	$(GO) run $(CMD_DIR)/snowy

## web-dev: 启动前端开发服务器 (localhost:3000)
web-dev:
	@echo "$(GREEN)▸ Starting frontend dev server...$(RESET)"
	cd web/snowy-web && npm run dev

## web-build: 构建前端静态产物
web-build:
	@echo "$(GREEN)▸ Building frontend...$(RESET)"
	cd web/snowy-web && npm run build
	@echo "$(GREEN)✓ Frontend built: web/snowy-web/out/$(RESET)"

# ============================================================
#  Database Migration (GORM)
# ============================================================

.PHONY: migrate-up migrate-reset

## migrate-up: 使用 GORM 初始化 / 同步 MySQL Schema
migrate-up:
	@echo "$(GREEN)▸ Running GORM migrations...$(RESET)"
	SNOWY_DATABASE_HOST="$(DB_HOST)" \
	SNOWY_DATABASE_PORT="$(DB_PORT)" \
	SNOWY_DATABASE_USER="$(DB_USER)" \
	SNOWY_DATABASE_PASSWORD="$(DB_PASSWORD)" \
	SNOWY_DATABASE_NAME="$(DB_NAME)" \
	$(MYSQL_MIGRATE_CMD)

## migrate-reset: 重启基础设施并重新应用 GORM Schema
migrate-reset: docker-down docker-up

# ============================================================
#  Code Generation
# ============================================================

.PHONY: generate

## generate: 运行全部代码生成 (go generate)
generate:
	@echo "$(GREEN)▸ Running go generate...$(RESET)"
	$(GO) generate ./...

# ============================================================
#  Dependencies
# ============================================================

.PHONY: deps tidy

## deps: 下载 Go 依赖
deps:
	@echo "$(GREEN)▸ Downloading dependencies...$(RESET)"
	$(GO) mod download

## tidy: 整理 go.mod / go.sum
tidy:
	@echo "$(GREEN)▸ Tidying modules...$(RESET)"
	$(GO) mod tidy

# ============================================================
#  Tools Installation
# ============================================================

.PHONY: tools

## tools: 安装全部开发工具
tools:
	@echo "$(GREEN)▸ Installing dev tools...$(RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ All tools installed$(RESET)"

# ============================================================
#  CI Pipeline (组合目标)
# ============================================================

.PHONY: ci check

## ci: CI 全流程 — fmt → vet → lint → test → build
ci: fmt vet lint test build
	@echo "$(GREEN)✓ CI pipeline passed$(RESET)"

## check: 快速检查 — vet → test
check: vet test

# ============================================================
#  Help
# ============================================================

.PHONY: help

## help: 显示所有可用目标
help:
	@echo ""
	@echo "$(CYAN)Snowy Makefile$(RESET)"
	@echo "$(CYAN)──────────────────────────────────────────────$(RESET)"
	@echo ""
	@echo "$(YELLOW)项目信息:$(RESET)"
	@echo "  Version : $(VERSION)"
	@echo "  Commit  : $(COMMIT)"
	@echo "  Module  : $(MODULE)"
	@echo ""
	@echo "$(YELLOW)可用目标:$(RESET)"
	@grep -E '^## ' $(MAKEFILE_LIST) | \
		sed -e 's/^## //' | \
		awk -F': ' '{printf "  $(GREEN)%-22s$(RESET) %s\n", $$1, $$2}'
	@echo ""
