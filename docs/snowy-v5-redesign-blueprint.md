# Snowy v5 重构蓝图：面向高中生的 AI 科学任务舱

> v5 的核心判断：Snowy 不能继续停留在“单次匿名 AI 演示”。它必须升级为一个 **可登录、可沉淀、可反馈、可观测、可切换模型** 的学习产品。否则答案质量、用户留存、成本治理和模型调优都会缺少闭环。

## 1. 核心设计目标

### 1.1 重构 API / 大模型接入模块

Snowy v5 的模型接入层使用 **`llm.models[]` 有序模型列表 + OpenAI-compatible 统一协议 + 可诊断错误处理**；所有供应商统一经 OpenAI-compatible `/chat/completions` 链路接入。

每个模型配置必须至少包含：

| 配置项 | 说明 | 设计约束 |
|---|---|---|
| `provider` | 统一 Provider 标识，默认 `openai` | 用于监控和脱敏展示；调用实现统一走 OpenAI-compatible Provider |
| `model_provider` | 部分网关或聚合服务要求的厂商字段 | 可选；仅作为请求字段透传，不触发专用 Provider 分支 |
| `base_url` | OpenAI-compatible 或厂商 API Base URL | 统一去除尾部 `/`，不得展示 API Key |
| `api_key` | 运行时密钥 | 允许由环境变量覆盖；仓库配置文件不得写明文密钥 |
| `model` / `model_name` | 实际模型名 | 兼容两种字段，最终归一为 `EffectiveModel()` |
| `temperature` | 默认采样温度 | 业务请求未指定时使用模型级默认值 |
| `max_tokens` | 默认输出 token 上限 | 不指定时使用 Snowy 安全上限 |
| `max_retries` | 单模型错误重试次数 | 只对网络、限流、5xx、超时等瞬时错误重试 |
| `retry_interval` | 单模型重试间隔 | 支持配置化，默认 1s |
| `timeout` | 单次模型调用超时 | 默认配置为 10m，不允许无限等待 |

核心机制：

1. **统一 Request / Response**：业务层只依赖 `llm.Provider`，不直接调用厂商 SDK。
2. **有序模型列表**：按 `llm.models[]` 的 YAML 声明顺序尝试。
3. **可观测的兜底链路**：每一次模型尝试都记录 provider、model、latency、status、error、tokens、user_id、operation。
4. **错误分类**：参数错误、认证错误、配额/限流、超时、5xx、解析失败应能区分；只有可恢复错误进入 retry / fallback。
5. **单一配置入口**：模型清单只从 `llm.models[]` 读取，避免同一语义存在多套配置来源。

### 1.2 实现用户登录、收藏与学习档案

v5 必须从匿名用户升级为真实学习账户系统。首发不做复杂 OAuth 运营后台，但必须提供邮箱注册 / 登录能力。

#### 登录与账户

- 邮箱 + 密码注册。
- 邮箱 + 密码登录。
- 密码使用 bcrypt 哈希存储，不保存明文。
- 登录成功返回 access token / refresh token。
- 未登录用户仍允许浏览与试用，但写入行为默认落到匿名用户；登录后必须落到真实用户。

#### 收藏与学习档案

用户可以收藏：

- 问题 / 搜索答案；
- 引用证据；
- 生成式模型包；
- 建模代码或渲染结果；
- 模型配置快照（仅脱敏配置）。

学习档案至少展示：

- 历史问题；
- 收藏内容；
- 点赞 / 点踩记录；
- 生成式模型包；
- 最近使用的模型与模型质量反馈。

### 1.3 答案、建模代码与社区反馈持久化

v5 不接受“生成后即丢”的临时演示链路。所有对学习价值有贡献的内容都必须具备可追溯记录。

#### 必须持久化

1. 每个问题的回答摘要、知识标签、引用证据、置信度。
2. 每个生成式模型包，包括模型规格、推理摘要、交互方案、校验报告、代码包或渲染 Manifest。
3. 每个用户对答案 / 证据 / 模型包 / 渲染代码的 `like` / `dislike`。
4. 聚合后的点赞数、点踩数、当前用户态，以及在隐私允许时的公开用户列表。

#### 社区反馈规则

- 单个用户对同一目标只能保留一个反馈：`like` 或 `dislike`。
- 用户可以撤销反馈。
- 聚合计数必须实时可读。
- 用户列表默认只展示公开反馈；隐私策略未完成前，接口仅返回脱敏用户摘要。
- 反馈数据要能反向进入排序、推荐、模型评估和人工审核。

### 1.4 增强 AI 监控模块

v4 的内存监控只能用于当前进程排障；v5 需要持久化、按用户聚合、支持诊断。

监控记录至少包含：

| 字段 | 用途 |
|---|---|
| `user_id` | 用户粒度调用分析、成本归因、异常定位 |
| `provider/model/base_url` | 模型路由和供应商质量分析 |
| `operation` | 区分知识问答、模型包编译、渲染生成等链路 |
| `status/error` | 成功率、错误率、异常诊断 |
| `latency_ms` | 延迟分布、P50 / P95 / Max |
| `input_tokens/output_tokens` | 成本估算与 token 预算治理 |
| `prompt_preview/system_pe/user_prompt` | Prompt Profile 诊断，必须截断并脱敏 |
| `started_at/finished_at` | 时间过滤与趋势分析 |

监控看板要求：

- 展示调用频次、成功率、错误率、延迟分布、Token 总量；
- 支持按时间、用户、模型、供应商、operation 过滤；
- 展示 Prompt Profile 注册表与最近 Prompt 快照；
- 监控数据落 MySQL，容器重启后仍可查询最近记录；
- API Key 永不返回前端。

## 2. v5 产品原则

1. **Evidence First**：面向高中生的结论必须尽量绑定证据、公式、推导或可解释模型。
2. **Persistent Learning Loop**：搜索、建模、收藏、反馈、复盘形成长期学习链路。
3. **Observable AI**：所有模型调用都要可追踪、可聚合、可诊断，而不是黑盒。
4. **Configurable Reliability**：可靠性来自多模型路由、重试、降级、校验，而不是寄希望于单一模型永远可用。
5. **Beautiful by Default**：前端保持科技感任务舱视觉，但交互必须清晰、可读、低认知负担。
6. **Privacy-aware Community**：社区反馈优先呈现质量信号，用户列表必须受隐私策略控制。

## 3. 系统架构调整

```mermaid
graph TB
    Web[Snowy Web v5]
    Auth[Auth Middleware\nJWT + Anonymous fallback]
    User[User Service\nEmail login / Profile]
    Learning[Learning Archive\nHistory / Favorites / Reactions]
    ModelRouter[LLM Ordered Router\nmodels[] declaration order]
    ProviderA[Provider Adapter A]
    ProviderB[Provider Adapter B]
    ProviderN[Provider Adapter N]
    Monitor[LLM Recorder\nMemory + MySQL Store]
    MySQL[(MySQL)]

    Web --> Auth
    Auth --> User
    Auth --> Learning
    Auth --> ModelRouter
    ModelRouter --> ProviderA
    ModelRouter --> ProviderB
    ModelRouter --> ProviderN
    ProviderA --> Monitor
    ProviderB --> Monitor
    ProviderN --> Monitor
    Monitor --> MySQL
    User --> MySQL
    Learning --> MySQL
```

## 4. 数据模型设计

### 4.1 `users`

新增字段：

- `password_hash`：bcrypt 密码哈希；Google 用户或匿名用户为空。
- `email` 建议建立唯一索引；兼容旧数据时允许空字符串。

### 4.2 `favorites`

v5 扩展 `target_type`：

- `search`
- `answer`
- `evidence`
- `physics`
- `biology`
- `model_package`
- `render_code`
- `model_config`

后续可增加 `metadata_json` 保存脱敏快照。

### 4.3 `reactions`

| 字段 | 说明 |
|---|---|
| `id` | UUID |
| `user_id` | 用户 ID |
| `target_type` | answer/evidence/model_package/render_code/search/physics/biology |
| `target_id` | 被反馈对象 ID |
| `reaction_type` | like/dislike |
| `visibility` | public/private |
| `created_at/updated_at` | 时间 |

约束：`(user_id, target_type, target_id)` 唯一。

### 4.4 `llm_call_records`

持久化监控事件，字段与 `monitoring.LLMCallRecord` 对齐，并对 `user_id`、`provider`、`model`、`operation`、`finished_at` 建索引。

## 5. API 设计

### 5.1 Auth

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/v1/auth/register` | 邮箱注册 |
| `POST` | `/api/v1/auth/login` | 邮箱登录 |
| `GET` | `/api/v1/user/profile` | 当前用户资料 |

### 5.2 收藏与反馈

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/v1/favorites` | 添加收藏 |
| `GET` | `/api/v1/favorites` | 收藏列表 |
| `PUT` | `/api/v1/reactions` | 设置 like/dislike |
| `DELETE` | `/api/v1/reactions` | 撤销反馈 |
| `GET` | `/api/v1/reactions` | 当前用户反馈列表 |
| `GET` | `/api/v1/reactions/summary` | 指定目标聚合计数 |

### 5.3 监控

`GET /api/v1/monitoring/llm` 支持查询参数：

- `user_id`
- `provider`
- `model`
- `operation`
- `since`
- `until`
- `limit`

## 6. 前端体验设计

### 6.1 导航与视觉

- 顶部版本标记升级为 `V5`。
- 登录态显示用户昵称 / 邮箱；未登录显示“访客模式”。
- 保持 v4 的深色玻璃拟态、霓虹网格与任务舱语言，但页面信息层级要更克制。

### 6.2 学习中心

学习中心从“两栏历史/收藏”升级为四个视图：

1. 历史问题；
2. 收藏；
3. 点赞 / 点踩；
4. 生成式模型包。

### 6.3 监控看板

- 顶部增加筛选器；
- 最近调用支持展开 Prompt 快照；
- 模型配置只显示脱敏信息；
- 如果持久化存储不可用，看板必须给出明确诊断。

## 7. 可靠性与安全设计

### 7.1 可靠性

- 模型调用按优先级路由；
- 单模型可配置 retry；
- 所有生成式模型包继续经过 schema / domain / safety 校验；
- LLM 失败时保留规则兜底或模板兜底；
- 监控记录不得阻塞主业务链路，持久化失败只记录警告。

### 7.2 安全

- 密码 bcrypt 哈希；
- JWT secret 生产环境必须覆盖；
- API Key 只允许运行时注入，不进入响应；
- Prompt 快照截断并避免敏感信息；
- 渲染代码仍然执行白名单 / 黑名单校验；
- 用户列表默认遵循 `visibility`。

## 8. 分阶段实施计划

### Phase 1：v5 基础闭环（当前实施）

- [x] 完善 v5 蓝图文档；
- [x] `llm.models[]` 多模型配置与优先级路由；
- [x] 邮箱注册 / 登录；
- [x] JWT 鉴权从“永远匿名”升级为“有效 token 使用真实用户，无 token 回落匿名”；
- [x] `reactions` 反馈持久化与聚合 API；
- [x] LLM 调用记录 MySQL 持久化，并支持用户 / 模型 / 时间过滤；
- [x] 前端 API 客户端接入 token、auth、reaction、monitoring filters；
- [x] OpenAPI 更新；
- [x] 单元测试覆盖关键路径。

### Phase 2：学习档案增强

- [ ] 学习中心展示反馈记录和模型包记录；
- [ ] 搜索答案持久化为 `answer_records`；
- [ ] 收藏 metadata 快照；
- [ ] 社区反馈影响答案排序与推荐。

### Phase 3：模型运营与质量评估

- [ ] 模型 A/B 实验；
- [ ] Prompt Profile 版本化入库；
- [ ] 质量评分与人工审核后台；
- [ ] 每日成本、错误率、慢调用诊断报告。

## 9. 验收标准

1. 配置 `llm.models[]` 多个模型后，调用链路按 YAML 声明顺序尝试，当前模型失败会进入下一模型。
2. 通过邮箱注册 / 登录能获得 JWT；携带 JWT 请求 `/user/profile` 返回真实用户。
3. 未携带 JWT 的旧链路仍可匿名使用，避免破坏现有体验。
4. 用户对同一目标设置 like 后再设置 dislike，聚合计数应从 like 转移到 dislike。
5. LLM 调用结束后 `llm_call_records` 有持久化记录；重启后看板仍能读取数据库最近记录。
6. 监控 API 支持按用户、模型、时间过滤。
7. API Key、password_hash 不出现在任何前端响应中。
8. `go test ./...` 与前端 lint/build 至少通过可执行验证；若环境依赖不可用，必须记录阻塞原因。
