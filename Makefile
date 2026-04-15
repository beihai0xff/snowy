# ============================================================
#  Snowy Makefile
#  参考技术方案 §7 工程结构 & §6 技术栈选型
#
#  用法:
#    make help          — 查看所有可用目标
#    make build         — 编译全部 Go 二进制
#    make test          — 运行测试
#    make docker-up     — 启动基础设施 (MySQL/Redis/OpenSearch/MinIO/...)
#    make docker-build  — 构建应用 Docker 镜像
#    make run-api       — 本地运行 API 服务
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
MIGRATION_DIR  := $(ROOT_DIR)/internal/repo/mysql/migrations

# ── Go 参数 ─────────────────────────────────────────────────
GO             := go
GOFLAGS        :=
LDFLAGS        := -s -w \
                  -X main.Version=$(VERSION) \
                  -X main.BuildTime=$(BUILD_TIME) \
                  -X main.Commit=$(COMMIT)
GOTEST_FLAGS   := -race -count=1 -timeout 120s
TEST_DEPS_SERVICES := mysql redis opensearch minio

# ── Docker 参数 ─────────────────────────────────────────────
DOCKER_COMPOSE := docker compose -f $(DEPLOY_DIR)/docker-compose.yml -p $(PROJECT_NAME)
DOCKER_REG     ?=
IMAGE_API      := $(if $(DOCKER_REG),$(DOCKER_REG)/)$(PROJECT_NAME)-api:$(VERSION)
IMAGE_WORKER   := $(if $(DOCKER_REG),$(DOCKER_REG)/)$(PROJECT_NAME)-worker:$(VERSION)

# ── 工具 ────────────────────────────────────────────────────
GOLANGCI_LINT  := $(shell command -v golangci-lint 2>/dev/null)
MIGRATE        := $(shell command -v migrate 2>/dev/null)
SQLC           := $(shell command -v sqlc 2>/dev/null)

# ── 数据库 (本地开发默认值) ─────────────────────────────────
DB_HOST        ?= localhost
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

.PHONY: build build-api build-worker clean

## build: 编译全部 Go 二进制 (api + worker)
build: build-api build-worker

## build-api: 编译 API 服务
build-api:
	@echo "$(GREEN)▸ Building snowy-api...$(RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/snowy-api $(CMD_DIR)/api
	@echo "$(GREEN)✓ $(BIN_DIR)/snowy-api$(RESET)"

## build-worker: 编译 Worker 服务
build-worker:
	@echo "$(GREEN)▸ Building snowy-worker...$(RESET)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/snowy-worker $(CMD_DIR)/worker
	@echo "$(GREEN)✓ $(BIN_DIR)/snowy-worker$(RESET)"

## clean: 清理编译产物
clean:
	@echo "$(YELLOW)▸ Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR)
	@$(GO) clean -cache -testcache
	@echo "$(YELLOW)✓ Cleaned$(RESET)"

# ============================================================
#  Test & Lint
# ============================================================

.PHONY: test test-unit test-integration test-e2e test-coverage lint fmt vet test-deps-up test-deps-down

## test: 运行单元测试
test: test-unit

## test-unit: 运行单元测试
test-unit:
	@echo "$(GREEN)▸ Running unit tests...$(RESET)"
	$(GO) test $(GOTEST_FLAGS) ./internal/...

## test-integration: 启动 MySQL/Redis/OpenSearch/MinIO Docker 依赖并运行集成测试
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

## lint: 运行 golangci-lint
lint:
ifndef GOLANGCI_LINT
	@echo "$(YELLOW)▸ Installing golangci-lint...$(RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
endif
	@echo "$(GREEN)▸ Running linter...$(RESET)"
	golangci-lint run ./...

## fmt: 格式化代码
fmt:
	@echo "$(GREEN)▸ Formatting code...$(RESET)"
	$(GO) fmt ./...
	@echo "$(GREEN)✓ Formatted$(RESET)"

## vet: 静态分析
vet:
	@echo "$(GREEN)▸ Running vet...$(RESET)"
	$(GO) vet ./...

## test-deps-up: 启动测试所需 Docker 依赖 (MySQL/Redis/OpenSearch/MinIO)
test-deps-up:
	@echo "$(CYAN)▸ Starting test dependencies: $(TEST_DEPS_SERVICES)...$(RESET)"
	$(DOCKER_COMPOSE) up -d $(TEST_DEPS_SERVICES)

## test-deps-down: 停止测试所需 Docker 依赖 (MySQL/Redis/OpenSearch/MinIO)
test-deps-down:
	@echo "$(YELLOW)▸ Stopping test dependencies: $(TEST_DEPS_SERVICES)...$(RESET)"
	-$(DOCKER_COMPOSE) stop $(TEST_DEPS_SERVICES)
	-$(DOCKER_COMPOSE) rm -f $(TEST_DEPS_SERVICES)

# ============================================================
#  Docker — 基础设施 (docker-compose)
# ============================================================

.PHONY: docker-up docker-down docker-ps docker-logs docker-clean

## docker-up: 启动全部基础设施 (MySQL/Redis/OpenSearch/MinIO/Prometheus/Grafana)
docker-up:
	@echo "$(CYAN)▸ Starting infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(CYAN)✓ Infrastructure is up$(RESET)"
	@echo ""
	@echo "  MySQL      : localhost:3306"
	@echo "  Redis      : localhost:6379"
	@echo "  OpenSearch : localhost:9200"
	@echo "  OS Dashboard: localhost:5601"
	@echo "  MinIO API  : localhost:9000"
	@echo "  MinIO Console: localhost:9001"
	@echo "  Prometheus : localhost:9090"
	@echo "  Grafana    : localhost:3000"

## docker-down: 停止全部基础设施 (保留数据卷)
docker-down:
	@echo "$(YELLOW)▸ Stopping infrastructure...$(RESET)"
	$(DOCKER_COMPOSE) down
	@echo "$(YELLOW)✓ Infrastructure stopped$(RESET)"

## docker-ps: 查看基础设施状态
docker-ps:
	$(DOCKER_COMPOSE) ps

## docker-logs: 查看基础设施日志 (用法: make docker-logs SVC=redis)
docker-logs:
	$(DOCKER_COMPOSE) logs -f $(SVC)

## docker-clean: 停止全部基础设施并删除数据卷 (⚠️ 数据将丢失)
docker-clean:
	@echo "$(YELLOW)▸ Destroying infrastructure and volumes...$(RESET)"
	$(DOCKER_COMPOSE) down -v --remove-orphans
	@echo "$(YELLOW)✓ Infrastructure destroyed$(RESET)"

# ============================================================
#  Docker — 应用镜像构建 & 运行
# ============================================================

.PHONY: docker-build docker-build-api docker-build-worker docker-run-api docker-run-worker docker-push

## docker-build: 构建全部应用 Docker 镜像 (api + worker)
docker-build: docker-build-api docker-build-worker

## docker-build-api: 构建 API 服务镜像
docker-build-api:
	@echo "$(CYAN)▸ Building Docker image: $(IMAGE_API)...$(RESET)"
	docker build \
		--build-arg TARGET=api \
		-f $(DEPLOY_DIR)/Dockerfile \
		-t $(IMAGE_API) \
		-t $(PROJECT_NAME)-api:latest \
		$(ROOT_DIR)
	@echo "$(CYAN)✓ $(IMAGE_API)$(RESET)"

## docker-build-worker: 构建 Worker 服务镜像
docker-build-worker:
	@echo "$(CYAN)▸ Building Docker image: $(IMAGE_WORKER)...$(RESET)"
	docker build \
		--build-arg TARGET=worker \
		-f $(DEPLOY_DIR)/Dockerfile \
		-t $(IMAGE_WORKER) \
		-t $(PROJECT_NAME)-worker:latest \
		$(ROOT_DIR)
	@echo "$(CYAN)✓ $(IMAGE_WORKER)$(RESET)"

## docker-run-api: 以容器方式运行 API 服务 (连接本地基础设施)
docker-run-api:
	@echo "$(CYAN)▸ Running snowy-api container...$(RESET)"
	docker run --rm -it \
		--name snowy-api \
		--network $(PROJECT_NAME)_default \
		-p 8080:8080 \
		-e DATABASE_URL="mysql://$(DB_USER):$(DB_PASSWORD)@tcp(snowy-mysql:3306)/$(DB_NAME)" \
		-e REDIS_ADDR="snowy-redis:6379" \
		-e OPENSEARCH_URL="http://snowy-opensearch:9200" \
		-e MINIO_ENDPOINT="snowy-minio:9000" \
		$(PROJECT_NAME)-api:latest

## docker-run-worker: 以容器方式运行 Worker 服务 (连接本地基础设施)
docker-run-worker:
	@echo "$(CYAN)▸ Running snowy-worker container...$(RESET)"
	docker run --rm -it \
		--name snowy-worker \
		--network $(PROJECT_NAME)_default \
		-p 8081:8081 \
		-e DATABASE_URL="mysql://$(DB_USER):$(DB_PASSWORD)@tcp(snowy-mysql:3306)/$(DB_NAME)" \
		-e REDIS_ADDR="snowy-redis:6379" \
		-e OPENSEARCH_URL="http://snowy-opensearch:9200" \
		-e MINIO_ENDPOINT="snowy-minio:9000" \
		$(PROJECT_NAME)-worker:latest

## docker-push: 推送应用镜像到远端仓库 (需设置 DOCKER_REG)
docker-push:
ifndef DOCKER_REG
	$(error DOCKER_REG is not set. Usage: make docker-push DOCKER_REG=your-registry.com)
endif
	docker push $(IMAGE_API)
	docker push $(IMAGE_WORKER)

# ============================================================
#  Run — 本地开发运行
# ============================================================

.PHONY: run-api run-worker dev

## run-api: 本地运行 API 服务 (需先 make docker-up 启动基础设施)
run-api: build-api
	@echo "$(GREEN)▸ Running snowy-api locally...$(RESET)"
	DATABASE_URL="$(DATABASE_URL)" \
	REDIS_ADDR="localhost:6379" \
	OPENSEARCH_URL="http://localhost:9200" \
	MINIO_ENDPOINT="localhost:9000" \
	$(BIN_DIR)/snowy-api

## run-worker: 本地运行 Worker 服务 (需先 make docker-up 启动基础设施)
run-worker: build-worker
	@echo "$(GREEN)▸ Running snowy-worker locally...$(RESET)"
	DATABASE_URL="$(DATABASE_URL)" \
	REDIS_ADDR="localhost:6379" \
	OPENSEARCH_URL="http://localhost:9200" \
	MINIO_ENDPOINT="localhost:9000" \
	$(BIN_DIR)/snowy-worker

## dev: 启动基础设施 + 本地运行 API 服务 (一键开发)
dev: docker-up run-api

# ============================================================
#  Database Migration (golang-migrate)
# ============================================================

.PHONY: migrate-up migrate-down migrate-create migrate-force migrate-version

## migrate-up: 执行全部待应用的迁移
migrate-up:
ifndef MIGRATE
	@echo "$(YELLOW)▸ Installing golang-migrate...$(RESET)"
	@go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
endif
	@echo "$(GREEN)▸ Running migrations up...$(RESET)"
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" up

## migrate-down: 回滚最近一次迁移
migrate-down:
	@echo "$(YELLOW)▸ Rolling back last migration...$(RESET)"
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" down 1

## migrate-create: 创建新迁移文件 (用法: make migrate-create NAME=create_users)
migrate-create:
ifndef NAME
	$(error NAME is required. Usage: make migrate-create NAME=create_users)
endif
	@mkdir -p $(MIGRATION_DIR)
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $(NAME)
	@echo "$(GREEN)✓ Migration files created in $(MIGRATION_DIR)$(RESET)"

## migrate-force: 强制设置迁移版本 (用法: make migrate-force V=1)
migrate-force:
ifndef V
	$(error V is required. Usage: make migrate-force V=1)
endif
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" force $(V)

## migrate-version: 查看当前迁移版本
migrate-version:
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" version

# ============================================================
#  Code Generation
# ============================================================

.PHONY: generate sqlc-generate

## generate: 运行全部代码生成 (go generate + sqlc)
generate: sqlc-generate
	@echo "$(GREEN)▸ Running go generate...$(RESET)"
	$(GO) generate ./...

## sqlc-generate: 运行 sqlc 代码生成
sqlc-generate:
ifndef SQLC
	@echo "$(YELLOW)▸ Installing sqlc...$(RESET)"
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
endif
	@echo "$(GREEN)▸ Running sqlc generate...$(RESET)"
	sqlc generate

# ============================================================
#  Dependencies
# ============================================================

.PHONY: deps tidy vendor

## deps: 下载 Go 依赖
deps:
	@echo "$(GREEN)▸ Downloading dependencies...$(RESET)"
	$(GO) mod download

## tidy: 整理 go.mod / go.sum
tidy:
	@echo "$(GREEN)▸ Tidying modules...$(RESET)"
	$(GO) mod tidy

## vendor: 创建 vendor 目录
vendor:
	@echo "$(GREEN)▸ Vendoring dependencies...$(RESET)"
	$(GO) mod vendor

# ============================================================
#  Tools Installation
# ============================================================

.PHONY: tools

## tools: 安装全部开发工具
tools:
	@echo "$(GREEN)▸ Installing dev tools...$(RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
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

