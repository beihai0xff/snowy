# agent.md — Snowy 项目大模型约束

## 适用范围

本文件约束在 Snowy 仓库根目录及其所有子目录内工作的 AI Agent / 大模型。除非用户明确给出更高优先级的临时要求，否则实现、重构、排障、文档更新、测试与运行服务时都必须遵守本文件。

## 项目定位

Snowy 是面向高中生的 Web 端 AIGC 科学学习平台，核心能力包括：

- RAG / 知识检索问答；
- 物理 / 3D 场景建模、推导、代码生成与浏览器渲染；
- 生物概念建模、关系抽取、过程拆解与实验变量分析；
- 基于 Eino Graph 的 Agent 编排、多模型路由、结构化输出与可观测监控。

技术栈：Go + Gin + GORM/MySQL + Redis/Asynq + OpenAI-compatible LLM Provider + Next.js/React/TypeScript。

## 重要目录

- `cmd/snowy/`：默认单体服务入口，API + embedded worker。
- `internal/app/`：应用装配与运行面组合根。
- `internal/agent/`：Agent 编排域，Eino Graph、节点、工具、路由、回调。
- `internal/modeling/physics/`：物理 / 场景建模域。
- `internal/modeling/biology/`：生物建模域。
- `internal/modeling/generative/`：生成式模型包编译与校验。
- `internal/handler/http/`：HTTP handler 与路由。
- `internal/repo/llm/`：统一 LLM Provider 接口与 OpenAI-compatible 实现。
- `internal/pkg/config/`：配置结构、加载、环境变量绑定。
- `internal/pkg/llmroute/`：有序模型链路与重试逻辑。
- `internal/monitoring/`：LLM 调用监控、脱敏 dashboard 数据。
- `web/snowy-web/`：Next.js 前端。
- `configs/`：多环境配置。
- `deployments/docker/`：Docker Compose、服务镜像与 Nginx 配置。
- `docs/`：PRD、技术方案、v4/v5 蓝图。

## 不可违反的硬约束

### 1. LLM 配置与调用链路

必须保持单一、统一、OpenAI-compatible 的 LLM 链路：

- 模型清单只从 `llm.models[]` 读取。
- 调用顺序严格等于 YAML 中 `llm.models[]` 的声明顺序。
- 每个模型通过统一 OpenAI-compatible `/chat/completions` 协议调用。
- `model_provider` 只能作为可选请求字段 / 监控元数据透传，不允许触发 provider 分支。
- 单模型内部只通过 `max_retries` / `retry_interval` 做同模型重试。
- 多模型只通过 `llm.models[]` 声明顺序继续尝试下一模型。
- LLM 请求超时时间默认 / 目标值为 `10m`；不得改回短超时或无限等待。

禁止重新引入下列历史遗留：

- `llm.primary`
- `llm.fallback`
- `primary/fallback` 双槽模型配置语义
- `SNOWY_LLM_PRIMARY`
- `SNOWY_LLM_FALLBACK`
- `MIMO_API_KEY`
- `XIAOMI_MIMO_API_KEY`
- MiMo / Mino / Xiaomi 专用 LLM provider 实现
- Gemini 专用 LLM provider 实现
- Google LLM provider 配置或实现
- 按 provider 名称分派到不同厂商 SDK / 不同 HTTP 协议的代码

注意：`auth.google_oauth` / Google OAuth 用户身份字段属于登录身份体系，不等于 Google LLM provider。除非用户明确要求移除 OAuth，否则不要因为清理 Google LLM provider 而误删用户登录相关配置或代码。

### 2. 密钥与配置安全

- 不得把真实 API Key、JWT secret、数据库密码、OAuth secret 等敏感值写入仓库。
- 示例配置只能保留空值、占位值或本地默认开发密码。
- 真实 LLM 密钥运行时通过 `OPENAI_API_KEY` 或配置注入；不要新增历史 provider 专用密钥变量。
- 输出日志、错误、监控数据时不得泄漏 API Key 或带密钥的 URL。

### 3. 前端产品口径

- v5 页面、文案、路由和新增功能的用户可见口径应使用 `v5` / `V5`，不要新增面向用户的 `v4` / `V4` 文案，除非是在历史文档或明确对比说明中。
- 全局登录入口位于 `web/snowy-web/components/layout/AppLayout.tsx`。
- 前端 API 客户端位于 `web/snowy-web/lib/api.ts`。
- 登录态本地存储 key：
  - `snowy_access_token`
  - `snowy_refresh_token`
- 登录态变更事件：`snowy-auth-change`。
- `web/snowy-web/app/learning/page.tsx` 中已有学习页登录 / 注册卡片；未经用户要求不要擅自删除。

### 4. 后端架构边界

- 新 HTTP 接口放在 `internal/handler/http/`，路由集中通过 router 注册。
- 新业务逻辑优先放在对应 domain/service 包，不要把复杂业务写进 handler。
- 基础设施实现放在 `internal/repo/`，业务层依赖接口或组合根注入，不要在业务层直接散落外部客户端初始化。
- 配置字段必须先进入 `internal/pkg/config` 的结构体，再由组合根装配使用。
- 数据库 schema / repository 修改要同时考虑 migration、测试、JSON 字段序列化与历史数据兼容。
- 生成式模型包必须保留 schema/domain/safety/evidence 校验与 fallback 信息，不要为了“看起来成功”跳过校验。

### 5. Docker 与运行面

- 默认服务入口是 `cmd/snowy`，默认运行模式为 `server.run_mode=all`。
- `SNOWY_SERVER_RUN_MODE=api|worker` 只用于临时拆分 API / worker surface。
- Docker Compose 项目文件：`deployments/docker/docker-compose.yml`。
- 应用容器通过 Compose 服务名访问基础设施：`mysql`、`redis`。
- 不要把临时调试用代理、私有镜像源、个人路径或本机绝对路径提交到 Dockerfile / Compose / 配置文件。

## 代码风格与实现准则

- Go 代码保持小而明确的函数边界，错误要带上下文，避免吞错。
- TypeScript / React 代码保持显式类型，避免把核心接口退化为 `any`。
- 不要用“大重写”解决局部问题；优先做最小、可验证、可回滚的修改。
- 不要覆盖用户已有改动。修改前后都应查看 `git status --short`。
- 不要提交构建产物、缓存、`node_modules`、`.next`、`out`、本地二进制或真实 `.env`。
- 文档、配置、代码必须互相一致；改配置语义时同步 README / docs / tests。

## 推荐验证命令

根据变更范围选择验证，完成较大改动时优先跑完整链路：

```bash
go test ./...
cd web/snowy-web && npm run lint
cd web/snowy-web && npm run build
go build -o bin/snowy ./cmd/snowy
git diff --check
```

Docker 运行态验证：

```bash
docker compose -f deployments/docker/docker-compose.yml -p snowy build snowy snowy-web
docker compose -f deployments/docker/docker-compose.yml -p snowy up -d snowy snowy-web
docker compose -f deployments/docker/docker-compose.yml -p snowy ps
curl -fsS http://localhost:8080/healthz
curl -fsSI http://localhost:3001
```

如涉及集成测试或数据库 / Redis 行为，按需执行：

```bash
make test-integration
./scripts/test.sh --integration --keep-deps
```

## 交付前检查清单

在向用户汇报前，必须确认：

1. `git status --short` 中的变更都与当前任务有关；如存在用户此前改动，明确说明未覆盖。
2. 已扫描并确认没有重新引入被禁止的 LLM 历史遗留关键词。
3. 已按变更范围运行必要验证，并记录通过 / 失败 / 未运行的原因。
4. 服务运行类任务必须确认容器状态与健康检查，而不是只确认命令已执行。
5. 最终回复必须列出修改文件、关键行为变化、验证命令与运行态状态。
