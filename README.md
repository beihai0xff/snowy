# ❄️ Snowy

**Snowy** 是一款面向高中生的 Web 端 AIGC 学习平台。基于 RAG 检索增强生成与 Agent 智能编排，为学生提供高可信度的知识问答、物理推导建模与生物概念建模能力。

> 首发聚焦：**知识检索** + **物理建模**（推导 + 2D 图表 + 参数调节）+ **生物建模**（概念识别 + 关系抽取 + 过程拆解）

---

## ✨ 核心能力

| 能力 | 说明 |
|---|---|
| 🔍 **知识检索** | 统一索引课本、考纲、题库与讲义；基于 RAG 的高可信知识问答 |
| 📐 **物理建模** | 条件抽取 → 公式推导 → 数值计算 → 2D 图表 → 参数调节 |
| 🧬 **生物建模** | 概念识别 → 关系抽取 → 过程拆解 → 实验变量分析 → 结构图/流程图 |
| 🤖 **Agent 编排** | 基于 Eino Graph 的意图识别、工具调用、多模型路由与结构化输出 |
| 🔄 **多模型路由** | `gpt5` 主推理、`gemini3` 备选，自动回退与成本管控 |

---

## 🏗️ 技术栈

| 层次 | 选型 |
|---|---|
| **后端语言** | Go 1.24+ |
| **HTTP 框架** | Gin |
| **Agent 编排** | [Eino](https://github.com/cloudwego/eino) (CloudWeGo) |
| **数据库** | MySQL 8.0+ (GORM + go-sql-driver/mysql) |
| **缓存 & 队列** | Redis 7 + Asynq |
| **搜索引擎** | OpenSearch（全文 + 向量 + 混合检索） |
| **对象存储** | MinIO (S3 兼容，开发环境) |
| **前端** | React / Next.js + TypeScript |
| **可观测** | OpenTelemetry + Prometheus + Grafana |

---

## 📂 项目结构

```text
snowy/
  cmd/
    api/                  # HTTP API 服务入口
    worker/               # Asynq 异步任务 Worker 入口
  internal/
    agent/                # Agent 编排域（Eino Graph）
    user/                 # 用户服务
    content/              # 内容入库域
    modeling/
      physics/            # 物理建模域
      biology/            # 生物建模域
    handler/http/         # HTTP Handler 层
    repo/                 # 基础设施层（MySQL / Redis / LLM / Embedding / OpenSearch / Storage）
    pkg/                  # 公共基础包（common / config / middleware）
  api/openapi/            # OpenAPI 契约
  web/snowy-web/          # 前端项目
  configs/                # 多环境配置
  deployments/docker/     # Docker Compose & Dockerfile
  docs/                   # 产品 & 技术方案文档
  Makefile                # 构建 / 测试 / 部署入口
```

> 完整目录结构与设计说明见 [技术方案 §7](./docs/tech-solution.md#7-go-项目工程结构社区标准)。

---

## 🚀 快速开始

### 前置要求

- Go 1.24+
- Docker & Docker Compose
- Make

### 1. 启动基础设施

```bash
make docker-up
```

将启动以下服务：

| 服务 | 地址 |
|---|---|
| MySQL | `localhost:3306` |
| Redis | `localhost:6379` |
| OpenSearch | `localhost:9200` |
| OpenSearch Dashboards | `localhost:5601` |
| MinIO Console | `localhost:9001` |
| Prometheus | `localhost:9090` |
| Grafana | `localhost:3000` |

### 2. 执行数据库迁移

```bash
make migrate-up
```

### 3. 编译 & 运行

```bash
# 编译全部
make build

# 本地运行 API 服务
make run-api

# 本地运行 Worker 服务
make run-worker

# 一键开发（启动基础设施 + 运行 API）
make dev
```

### 4. 构建 Docker 镜像

```bash
# 构建 api + worker 镜像
make docker-build

# 以容器方式运行（自动加入基础设施网络）
make docker-run-api
```

### 5. 查看全部 Make 目标

```bash
make help
```

### 6. 测试

推荐按分层执行测试：

- `make test` / `make test-unit`：纯单元测试，默认使用 mock，速度快、适合日常开发
- `make test-integration`：自动启动 Docker 中的 `MySQL` / `Redis` / `OpenSearch` / `MinIO`，执行带 `integration` tag 的真实依赖测试
- `make test-e2e`：执行带 `e2e` tag 的端到端测试

```bash
# 仅运行单元测试
make test

# 启动 MySQL / Redis / OpenSearch / MinIO Docker 依赖并运行集成测试
make test-integration

# 如需保留测试依赖容器，便于手动排查
./scripts/test.sh --integration --keep-deps
```

集成测试默认连接以下本地端点（可通过环境变量覆盖）：

- MySQL：`127.0.0.1:3306`
- Redis：`127.0.0.1:6379`
- OpenSearch：`http://127.0.0.1:9200`
- MinIO：`127.0.0.1:9000`

可覆盖的环境变量示例：

```bash
SNOWY_DATABASE_HOST=127.0.0.1
SNOWY_DATABASE_PORT=3306
SNOWY_DATABASE_USER=snowy
SNOWY_DATABASE_PASSWORD=snowy_secret
SNOWY_DATABASE_NAME=snowy
SNOWY_REDIS_ADDR=127.0.0.1:6379
SNOWY_REDIS_PASSWORD=
SNOWY_REDIS_DB=0
SNOWY_OPENSEARCH_ADDR=http://127.0.0.1:9200
SNOWY_OPENSEARCH_USERNAME=admin
SNOWY_OPENSEARCH_PASSWORD=admin
SNOWY_OPENSEARCH_INDEX=snowy-content-integration
SNOWY_MINIO_ENDPOINT=127.0.0.1:9000
SNOWY_MINIO_ACCESS_KEY=snowy_admin
SNOWY_MINIO_SECRET_KEY=snowy_minio_secret
SNOWY_MINIO_BUCKET=snowy
```

当前集成测试会在执行前自动：

- 等待 MySQL / Redis / OpenSearch / MinIO 健康检查通过
- 通过 `internal/repo/mysql` 的 GORM migration runner 初始化 MySQL 表结构
- 重建 OpenSearch 集成测试索引
- 清理 MinIO 测试 bucket 中的对象
- 清理 MySQL 表数据与 Redis DB，避免脏数据影响结果

仓库中的 GitHub Actions 工作流会自动执行：

- 单元测试 + `go vet` + `go build`
- 基于 Docker Compose 的基础设施集成测试矩阵（MySQL / Redis / OpenSearch / MinIO）

---

## 📖 文档

| 文档 | 说明 |
|---|---|
| [产品需求文档 (PRD)](./docs/prd.md) | 产品目标、MVP 范围、核心功能、页面流程、接口边界、指标体系与里程碑 |
| [技术方案](./docs/tech-solution.md) | 系统架构、Agent 编排、RAG 检索、多模型路由、物理/生物建模、数据库设计、可观测性 |

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request。请确保：

1. 代码通过 `make ci`（fmt → vet → lint → test → build）
2. 接口变更同步更新 `api/openapi/` 契约
3. 文档更新与代码变更在同一 PR 中提交

---

## 📄 License

本仓库采用 [`Snowy Non-Commercial License 1.0`](./LICENSE) 发布。

- ✅ 允许：个人学习、教学演示、学术研究、非商业验证与二次修改
- ❌ 禁止：未经授权的商业使用、商用部署、SaaS/API 对外服务、付费集成、销售与再授权

> 说明：由于加入了“禁止商业使用”限制，当前仓库属于 **source-available**，不再属于 OSI 定义下的标准开源协议。

如需商业授权，请联系仓库维护者。
