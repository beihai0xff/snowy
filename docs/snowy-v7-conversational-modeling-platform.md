# Snowy v7 · 会话化建模平台升级方案

> 状态：草案，已与产品方对齐范围（2026-05-16）
> 范围：`internal/agent/`、`internal/modeling/`（含新增 chemistry）、`internal/handler/http/`、`internal/repo/`、`web/snowy-web/` 全链路
> 上游依赖：
> - `docs/snowy-v5-redesign-blueprint.md`（多模型路由 / 可靠性）
> - `docs/snowy-v6-modeling-visual-upgrade.md`（学科色 / R3F / 语义化生物图）
> - Gemini 平抛运动示例（同消息文本 + 内嵌可交互演示）
> 配套设计 token：`docs/design/v6/`

---

## 0. 文档定位

v7 把 4 块原本独立推进的工作合并为同一期交付，互为依赖、必须同窗口落地：

| 能力域 | 说明 | 与现状关系 |
|---|---|---|
| **A · 会话化内嵌演示** | 对话内同时给出文本 + 可调参演示卡；追问可局部重算或 LLM 重生成 | 沿用 §1–§11 既定方案（M1–M5），见 §3 |
| **B · R3F 场景视觉补完** | v6 已落骨架（`components/physics/r3f/scenes/*`，44–97 行最小可跑版），本期补齐材质 / 光照 / postFX / 场景专属模型 | 见 §4 |
| **C · 化学建模域（chemistry）** | 新增第三个建模学科：反应物 / 生成物 / 配平 / 电子转移动画 | 见 §5 |
| **D · 多用户协同 / 实时分享** | 演示卡可生成分享链接；多用户进入同卡片时参数、HUD 实时同步 | 见 §6 |

A 是骨架，B/C/D 都挂在 A 的契约之上。任一拆出去单独发布都会再绕一遍 SSE / 渲染层 / 持久化 / 路由集成，因此**本期一次性收敛**。

---

## 1. 用户故事

1. **高中生在 `/ask` 提问**「v0=30, h=40 的平抛运动落点是多少？」→ 同一气泡里出现：文字推导 + 引用 + 内嵌 3D 平抛 R3F 卡 + v0/h 滑块。
2. 用户拖滑块 → 轨迹 / HUD 毫秒级更新（前端纯数学）。
3. 用户在卡片底部追问「加上阻力」→ 同一卡片**原地**重生成成带阻力的物理模型。
4. 用户问「2NaOH + H2SO4 反应」→ 出现化学反应卡：原子球棍模型 + 配平动画 + 电子转移箭头 + 浓度滑块。
5. 用户点击「分享演示」→ 生成短链，同学打开后看到当前参数；老师与学生同时在线时，老师拖滑块全场同步。
6. 用户回看历史会话 → 演示卡完整保留（首选快照；可选「重放流式动效」）。

---

## 2. 总体设计

### 2.1 一句话
**会话即沙箱**：generative compile 产物作为消息附件嵌入对话；R3F 把它"演得好看"；chemistry 把"能演的领域"扩到第三个；协同把"沙箱"变成共享空间。

### 2.2 关键设计约束（与 `agent.md` 一致）
- LLM 仅走 OpenAI-compatible 接口，按 `llm.models[]` 顺序重试；不引入 google/gemini/mimo 专用 provider。
- 不修改 `configs/config.yaml`（gitignored 本地密钥）；新增 key 走 `configs/config.example.yaml` + viper default。
- 前端口径用 v5/V5，全局登录入口 `AppLayout.tsx`。
- LLM 超时 10m；流式聚合走 `StreamResponseAggregator`。

### 2.3 数据流总览

```
                ┌──────────────────────────────────────────┐
   /ask UI ────►│  POST /api/v1/agent/chat (SSE)           │
                │  ChatRequest{ parent_package_id?, ... }  │
                └────────────────────┬─────────────────────┘
                                     ▼
                 ┌──── agent graph (Eino 风格手工编排) ────┐
                 │ intent → retrieval → answer_writer       │
                 │   └► (parent_package_id?) regenerate_clf  │
                 │ → demo_planner ───┐                      │
                 │                   ▼                      │
                 │   generative.Compile / Recompute /        │
                 │   Regenerate (含 chemistry)               │
                 │                   │                      │
                 │   SSE: content / citation / preview      │
                 │        partial → preview complete         │
                 │   (持久化到 agent_message_events)         │
                 └──────────────────────────────────────────┘
                                     ▼
              ┌─────────── ChatBubble ────────────────────┐
              │ MarkdownText + Citations                   │
              │ InteractiveDemoCard                        │
              │   ├─ physics → R3FPhysicsPreview (v7 视觉) │
              │   ├─ biology → SemanticBiologyRenderer    │
              │   └─ chemistry → R3FChemistryPreview ★新  │
              │ InteractionPlanPanel（滑块 / 按钮）         │
              │ CollabPresence ★新 + ShareButton ★新       │
              │ ReactionBar                                │
              └────────────────────────────────────────────┘

  分享 / 协同：/api/v1/share/packages/:token (HTTP) +
              /api/v1/collab/packages/:id/ws  (WebSocket hub)
```

---

## 3. 能力 A · 会话化内嵌演示（沿用既定 M1–M5）

> 本节为 v6 升级方案 §10/§11 决策（2026-05-16 已锁）的工程映射，不再赘述设计取舍，详见原 plan §1–§11。

### 3.1 后端

| 改动点 | 文件 | 摘要 |
|---|---|---|
| 类型 | `internal/agent/types.go` | `ChatRequest` 增 `ParentPackageID/RegenerateReason/InteractiveDemo`；新增 `PreviewPayload{PackageID, Status, Stage, Package, Error}`；`Message` 增 `PackageID *uuid.UUID` |
| Aggregator | `internal/agent/stream_aggregator.go` | preview 事件从"保留首个"改为"始终覆盖到最新 complete"；新增 `Events() []SSEEvent` 暴露给 WriteService |
| Graph | `internal/agent/graph/graph.go` | `runPrimaryFlow` 物理/生物/化学分支后插入 `demo_planner`；`parent_package_id` 命中时前置 `regenerate_classifier` |
| 节点 | `internal/agent/node/demo_planner.go`、`regenerate_classifier.go` | 新增；后者用 LLM JSON 输出 `{action, target_vars, reason}`，失败回退关键词规则 |
| Generative | `internal/modeling/generative/service.go` + `compiler.go` | 新增 `Recompute(ctx, packageID, overrides)` 和 `Regenerate(ctx, parentID, reason, ctx)` |
| 持久化 | `internal/repo/mysql/schema.go`、`models.go`、`agent_repo.go` | `agentMessageSchema` 加 nullable `PackageID`；新增 `agent_message_events` 表 + `MessageEventRepository` |
| HTTP | `internal/handler/http/router.go`、`generative_handler.go`、`agent_handler.go` | modeling 组新增 `:id/recompute`、`:id/regenerate`；agent 组新增 `messages/:id/replay`（SSE） |
| WriteService | `internal/agent/write_service.go` | `PersistConversation` 接受 `Events []SSEEvent` 与 `PackageID`，按 seq 落 `agent_message_events` |

### 3.2 前端

| 改动点 | 文件 | 摘要 |
|---|---|---|
| API 客户端 | `lib/api.ts` | 新增 `chatStream / getPackage / recomputePackage / regeneratePackage / replayMessage` |
| 公式求值 | `lib/formula/evaluator.ts` | 引入 **`expr-eval`** ~5KB gz；白名单函数；变量域来自 `simulation_logic.variables` |
| 组件 | `components/chat/{ChatBubble,InteractiveDemoCard,DemoSkeleton,DemoFallbackCard}.tsx` | 新增；按 subject 路由到 R3F / Biology / Chemistry 渲染器 |
| 页面 | `app/ask/page.tsx` | 改造为会话式 UI（消息列表 + ChatBubble），保留旧 search tab |

### 3.3 已确认决策（重复列出以便单点检索）
- 单一入口：复用 `POST /api/v1/agent/chat`
- 追问 regenerate：LLM 二分类 + 关键词兜底
- 前端 evaluator：`expr-eval`
- `message.package_id`：nullable 迁移
- SSE replay：本期实现，事件落 `agent_message_events`

---

## 4. 能力 B · R3F 场景视觉补完

### 4.1 现状盘点
v6 已落地骨架（每场景 44–97 行），缺：
- **材质/光照**：所有场景共用默认 standardMaterial + 单光源；缺乏 PBR、HDRI 环境光、阴影。
- **场景专属几何**：spring 用 `<Cylinder>` 替代真螺旋；orbit 行星无贴图；collision 球体无变形；projectile 缺地面格栅与发射体尾焰。
- **postFX**：`lib/postFX.tsx` 仅 34 行，未挂 Bloom/DoF；HUD 与场景缺辉光区分。
- **粒子 / 尾迹 / 命中爆点**：`R3FTrail.tsx` 仅基本 Trail，无渐隐尾、无命中粒子爆裂。
- **HUD 设计语言**：`R3FHud.tsx` 用 antd Tag，与学科色/glass 风格未对齐 v6 token。

### 4.2 实施清单

| Scene | 必交付视觉要素 | 关键依赖 |
|---|---|---|
| `OrbitScene` | 中心恒星（自发光 + Bloom）、行星 PBR 贴图（procedural noise 即可）、轨道线（半透明 Tube）、慢转星空环境贴图 | `drei/Sparkles`、`postprocessing/Bloom`、自研 `lib/materials/StarSurface.ts` |
| `ProjectileScene` | 地面网格（极简灰格）、发射器（红橙金属感）、弹体尾焰（粒子）、落点爆点（`<Sparkles>` × `setTimeout`）、矢量箭头（v / g 分解） | 复用 `FrameVector` + 新 `lib/effects/Burst.tsx` |
| `SpringScene` | 真螺旋几何（TubeGeometry + 螺线生成器）、固定壁面材质、振动残影（trail） | 新 `lib/geo/HelixCurve.ts` |
| `CollisionScene` | 两球软体形变（顶点 morph，命中瞬间 lerp 1.0→0.85）、动量箭头、火花粒子 | 新 `lib/effects/Impact.tsx` |
| `ForceScene` | 斜面木质材质、物块阴影、摩擦轨迹热力图（地面颜色随接触时长变红） | `drei/ContactShadows` |
| `MotionScene` | 一般运动 fallback：轨迹线 + 切向/法向加速度箭头 + 速度色阶 | 复用 |

### 4.3 公共升级
- `lib/R3FStage.tsx`：挂 `<Environment preset="city">`（drei 内置）+ 软阴影 + tone mapping。
- `lib/postFX.tsx`：默认开启 Bloom（intensity 0.6, luminanceThreshold 0.85），高级开关 DoF。
- `lib/R3FHud.tsx`：重写为 glass card，CSS 变量用 `--color-physics`；移动端折叠为抽屉。
- 性能阀门：`useDeviceTier()` hook 检测 `navigator.hardwareConcurrency / deviceMemory / prefers-reduced-motion`，低端机关 Bloom 与粒子。
- 降级链：R3F 加载失败 / WebGL 不可用 → `NativePhysicsPreview` → `PhysicsPreviewSandbox` → `DemoFallbackCard`。

### 4.4 不在 B 范围
- HDRI 自制资产采买（用 drei 预设即可）；
- 真物理摩擦学（沿用 Rapier 默认）；
- VR/AR 模式。

---

## 5. 能力 C · 化学建模域（chemistry）

### 5.1 学科建模能力定义

**目标场景**：高中化学课纲覆盖的反应类型（酸碱中和、氧化还原、置换、复分解、电离、水解）+ 简单有机（取代/加成/消去）。本期先做**无机水溶液 + 小分子气相**两类，覆盖课纲约 70%。

**输入**：自然语言反应描述（`2NaOH + H2SO4 → Na2SO4 + 2H2O`、`电解水`、`铁与硫酸铜反应`）。

**输出**（`ChemistryReactionPackage`，与 physics/biology 平行的 generative 子类型）：

```go
// internal/modeling/chemistry/types.go (新)
type ChemistryReactionPackage struct {
    ReactionType   string             // neutralization | redox | displacement | electrolysis | ...
    Equation       BalancedEquation   // 配平 + 反应物/生成物 + 化学计量数
    Conditions     Conditions          // 温度 / 催化剂 / 溶剂 / 通电 / Δ
    ElectronTransfer []ETransfer      // 仅 redox：from→to 元素、电子数
    EnergyProfile  *EnergyProfile     // 可选：反应坐标 + 活化能
    Species        []SpeciesModel     // 每个分子的 3D 几何（原子坐标 + 键）
    Animation      AnimationScript    // 帧脚本：碰撞→形变→键断→键成→产物分离
    InteractionPlan InteractionPlan   // 浓度/温度/压力滑块
}

type BalancedEquation struct {
    Reactants []SpeciesCoef  // [{species:"NaOH", coef:2}, ...]
    Products  []SpeciesCoef
    Arrow     string         // "→" | "⇌" | "↑" | "↓"
}

type SpeciesModel struct {
    Formula    string                 // "H2O"
    Atoms      []AtomXYZ              // 简单分子用内置 SMILES → 3D 表查；复杂走 LLM 输出 + 校验
    Bonds      []Bond
    Color      string                 // CPK 配色
}
```

### 5.2 后端结构

```
internal/modeling/chemistry/
  service.go        // Service interface { AnalyzeReaction(eq) → ChemistryReactionPackage; Balance(eq) ... }
  balancer/         // 化学方程式配平：线性代数（Gauss 消元）方案
  reaction/         // 反应类型分类（规则 + LLM 兜底）
  smiles/           // 内置小分子 SMILES → 3D 坐标表（H2O / CO2 / NaOH / H2SO4 / Cu / Fe / ... 高频 50 个）
  electron/         // 氧化还原电子转移推导（基于氧化数计算）
  animation/        // 动画脚本生成（帧序列：碰撞 → 键变化 → 分离）
```

**配平算法**：把方程式系数视为线性方程组，用 `gonum/mat` 求最小整数正解（高中范围 ≤ 6 个未知数，绰绰有余）。

**集成**：
- `generative.CompilerService` 注入 `chemistry.Service`；`Compile` 在 `domain=chemistry` 分支调用，输出包装在 `GenerativeModelPackage.SimulationLogic` 里（复用既有契约，避免新增 top-level 字段）。
- Agent 工具：`internal/agent/tool/chemistry_analyze_tool.go`；`graph.Builder` 加 `WithChemistryAnalyzeTool`。
- 路由：`POST /api/v1/modeling/chemistry/analyze`、`POST /api/v1/modeling/chemistry/balance`。
- `mode` 枚举：`internal/agent/types.go`、`dto.ChatReq.Mode` binding 加入 `chemistry`；intent_classifier 升级为三分类。
- 监控：`monitoring.PromptProfile` 新增 chemistry 专属 profile。

### 5.3 前端

```
web/snowy-web/components/chemistry/
  R3FChemistryPreview.tsx   // 入口：球棍模型 + 动画播放器
  scenes/
    NeutralizationScene.tsx  // 酸碱中和：H+ 与 OH- 结合成 H2O 动画
    RedoxScene.tsx           // 氧化还原：电子流箭头 + 半反应高亮
    ElectrolysisScene.tsx    // 电解：电极 + 离子迁移
    DefaultMolecularScene.tsx // 通用球棍 + 反应箭头
  lib/
    CpkColors.ts             // 元素 CPK 配色
    BallStick.tsx            // 球棍模型组件（原子 Sphere + 键 Cylinder）
    EnergyProfile.tsx        // 反应坐标曲线
    EquationDisplay.tsx      // 配平式渲染（带系数动画）
  hooks/
    useReactionAnimation.ts  // 帧脚本回放
```

- `InteractiveDemoCard` 按 `package.domain === 'chemistry'` 路由到 `R3FChemistryPreview`。
- 降级：3D 失败 → 2D 球棍 SVG → `EquationDisplay` 纯文字。

### 5.4 数据 / 契约
- `GenerativeModelPackage.Domain` 枚举加 `chemistry`（DB 列类型不变，已是 varchar）。
- 既有 `PackageRepository` 不改 schema，靠 `domain` 字段区分。
- 前端 `lib/types/generative.ts` 镜像 chemistry 类型。

### 5.5 不在 C 范围
- 量子化学计算 / 真分子轨道；
- 有机反应机理详细电子推动（仅展示总反应）；
- 反应热力学定量计算（仅定性箭头）。

---

## 6. 能力 D · 多用户协同与实时分享

### 6.1 两档能力

| 档位 | 用户故事 | 技术形态 |
|---|---|---|
| **D1 · 静态分享（必交付）** | 用户点"分享演示" → 生成短链 → 他人打开看到**当时**参数与 package 快照 | HTTP 短链 + package 不可变快照 |
| **D2 · 实时协同（必交付）** | 多用户同时在线于同一卡片，参数 / 视角 / HUD 实时同步；显示在场用户头像 | WebSocket hub + 简化 CRDT（last-write-wins on params） |

### 6.2 数据模型

```go
// internal/repo/mysql/schema.go (新增)
type PackageShareSchema struct {
    Token       string    `gorm:"primaryKey;type:varchar(32)"`  // 8-12 字 nanoid
    PackageID   uuid.UUID `gorm:"type:char(36);index"`
    SnapshotJSON string   `gorm:"type:longtext"`                // 含 overrides
    CreatedBy   uuid.UUID `gorm:"type:char(36)"`
    ExpiresAt   *time.Time
    Mode        string                                          // "view" | "collab"
    CreatedAt   time.Time
}

type CollabSessionSchema struct {  // 内存优先 + Redis 持久；DB 仅做审计
    ID          uuid.UUID
    PackageID   uuid.UUID
    HostUserID  uuid.UUID
    Participants string  // JSON: [{user_id, role, joined_at}]
    LastSyncAt  time.Time
}
```

短链存 DB（量小）；活跃协同状态存 Redis（key `collab:{pkg_id}` → params hash + presence set）。

### 6.3 HTTP / WS 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/share/packages` | body: `{package_id, overrides, mode, ttl?}` → `{token, url}` |
| GET | `/api/v1/share/:token` | 返回快照 + 是否可加入 collab |
| POST | `/api/v1/share/:token/join` | 申请 collab，返回 ws_url + role |
| WS | `/api/v1/collab/packages/:pkg_id/ws?token=...` | 双向消息：`{type: param_update / cursor / presence / regenerate_request / chat, ...}` |

### 6.4 协同语义
- **参数同步**：last-write-wins（按服务器接收时间），节流 30ms；服务端聚合后广播。
- **角色**：host（创建者）可锁参数 / 启动重生成；guest 默认可改参数，可被 host 禁言。
- **冲突**：simple；不引入 yjs/automerge（关系简单，参数 < 20 个）。
- **离开**：presence 5s 心跳，超时移除。
- **重生成**：仅 host 可触发；触发后广播 SSE 事件流（复用 `messages/:id/replay` 的事件 schema）。

### 6.5 实现位置

```
internal/collab/
  hub.go            // 内存 + Redis pub/sub 的房间管理；每房间一个 goroutine
  message.go        // 消息类型
  presence.go       // 心跳与在场列表
internal/handler/http/
  share_handler.go  // 短链 CRUD + join
  collab_handler.go // WS 升级 + 路由到 hub
internal/repo/mysql/share_repo.go
internal/repo/redis/collab_store.go
```

前端：
```
web/snowy-web/components/chat/
  ShareButton.tsx
  CollabPresence.tsx  // 头像气泡
  hooks/useCollabRoom.ts  // WS 客户端封装，向 InteractiveDemoCard 注入受控参数
```

`InteractiveDemoCard` 把"参数 source of truth"从本地 state 抽象为可注入 controller：单机用 `useLocalParams()`；协同用 `useCollabRoom()`。

### 6.6 安全
- 短链 token 32 位 nanoid，不可枚举。
- collab 加入需登录用户（沿用 `AppLayout.tsx` 全局登录）；匿名仅能 view 短链。
- WS 消息体大小 ≤ 8KB；server 强制 schema 校验，拒绝未知字段。
- host 可踢人；24h 不活跃自动归档协同房间。

### 6.7 不在 D 范围
- 语音 / 视频；
- 富文本协同编辑；
- 跨房间广播 / 教师全班推送（v8 候选）。

---

## 7. 跨能力契约影响（汇总）

| 文件 | A | B | C | D |
|---|:-:|:-:|:-:|:-:|
| `internal/agent/types.go` | ✓ | | ✓(mode) | |
| `internal/agent/graph/graph.go` | ✓ | | ✓ | |
| `internal/agent/node/{demo_planner,regenerate_classifier}.go` | ✓ | | | |
| `internal/agent/tool/chemistry_analyze_tool.go` | | | ✓ | |
| `internal/modeling/generative/service.go` | ✓ | | ✓ | |
| `internal/modeling/chemistry/*` | | | ✓ | |
| `internal/repo/mysql/{schema,models,agent_repo}.go` | ✓ | | | |
| `internal/repo/mysql/share_repo.go` | | | | ✓ |
| `internal/repo/redis/collab_store.go` | | | | ✓ |
| `internal/collab/*` | | | | ✓ |
| `internal/handler/http/{router,agent_handler,generative_handler,share_handler,collab_handler}.go` | ✓ | | ✓ | ✓ |
| `internal/app/app.go` | ✓ | | ✓ | ✓ |
| `web/snowy-web/lib/{api,types/generative,formula/evaluator}.ts` | ✓ | | ✓ | |
| `web/snowy-web/components/chat/*` | ✓ | | | ✓ |
| `web/snowy-web/components/physics/r3f/{scenes,lib}/*` | | ✓ | | |
| `web/snowy-web/components/chemistry/*` | | | ✓ | |
| `web/snowy-web/app/ask/page.tsx` | ✓ | | | |
| `configs/config.example.yaml` | ✓ | | | ✓(redis collab key prefix 可选) |
| `api/openapi/*.yaml` | ✓ | | ✓ | ✓ |

---

## 8. 里程碑（无时间估计，仅依赖关系）

```
M0  方案 review & 锁定（当前已完成 → 进入 M1）
│
├─ A 序列（会话化骨架）
│   M1  契约 & SSE 通道（types / aggregator / migration / generative Recompute&Regenerate / 事件持久化 / replay handler）
│   M2  Agent 节点（demo_planner + regenerate_classifier）+ HTTP 端点 + 集成测试
│   M3  前端 chatStream + ChatBubble + InteractiveDemoCard + expr-eval + DemoSkeleton/Fallback
│   M4  参数交互三层（本地 evaluator / recompute / regenerate）
│   M5  /ask 改造 + 端到端验证
│
├─ B 序列（R3F 视觉补完）  依赖 M3（InteractiveDemoCard 路由到 R3F）
│   M6  公共升级：R3FStage 环境光 + postFX + R3FHud 重写 + useDeviceTier
│   M7  六场景逐场景补齐（OrbitScene → ProjectileScene → SpringScene → CollisionScene → ForceScene → MotionScene）
│   M8  降级链联调（R3F→Native→Sandbox→Fallback）+ 移动端抽屉
│
├─ C 序列（chemistry 学科）  依赖 M1（契约）+ M2（agent 工具装配）
│   M9   后端：chemistry service / balancer / smiles 表 / reaction 分类 / animation 脚本
│   M10  generative + agent + HTTP 接入；mode 枚举扩展；intent_classifier 三分类
│   M11  前端 R3FChemistryPreview + scenes + 降级链
│   M12  端到端：5 个高频反应 happy path（中和 / 氧化还原 / 置换 / 电解 / 复分解）
│
└─ D 序列（协同 / 分享）  依赖 M5（演示卡参数已可控）
    M13  D1 静态分享：share schema + handler + 短链 + 前端 ShareButton
    M14  D2 协同：hub + WS + Redis presence + CollabPresence + useCollabRoom
    M15  权限 / 安全 / 限流；与 host 重生成联动；e2e 多 tab 验证

合并出口：M16  全链路 lint / build / make test-integration / 手动场景矩阵
```

> 并行机会：B 与 C 在 A 的 M3 完成后即可并行启动；D 的 M13 与 C 的 M9–M10 可并行。

---

## 9. SQL 任务跟踪

session 内已建 14 个 A 序列 todos + 17 条依赖。本 v7 文档定稿后追加：
- B 序列 3 个（`m6-r3f-stage-postfx`、`m7-r3f-scenes`、`m8-r3f-fallback-mobile`）
- C 序列 4 个（`m9-chem-backend`、`m10-chem-integration`、`m11-chem-frontend`、`m12-chem-e2e`）
- D 序列 3 个（`m13-share-static`、`m14-collab-realtime`、`m15-collab-security`）
- 合并 1 个（`m16-final-validation`，依赖 M5/M8/M12/M15）

依赖：M6→M3；M7→M6；M8→M7；M9→M1；M10→M9 & M2；M11→M10 & M3；M12→M11；M13→M5；M14→M13；M15→M14；M16→M5,M8,M12,M15。

---

## 10. 风险与对策

| # | 风险 | 概率 | 对策 |
|---|---|---|---|
| R1 | LLM 化学方程式产出错误（配错原子 / 不守恒） | 高 | 后端 balancer 强校验质量守恒、电荷守恒；不通过则 SSE `preview {status:"failed"}`，前端显示原始式 + "重新解析" |
| R2 | R3F 视觉升级在低端机帧率崩盘 | 中 | `useDeviceTier` 自动降级；Bloom/粒子可关；fallback 链兜底 |
| R3 | WebSocket 长连接放大服务器内存 | 中 | 单实例上限（默认 1000 房间 × 20 人）配置化；超过转拒绝 + 引导静态分享；Redis pub/sub 支持多实例水平扩展（v8 启用） |
| R4 | 协同最后写赢导致体验抖动 | 中 | 客户端 30ms 节流 + 服务端 60ms 合并广播；视觉用 lerp 平滑过渡而非硬切 |
| R5 | 化学 SMILES→3D 表覆盖率不够 | 中 | 内置 50 个高频分子；命中失败时回退 2D 球棍 + 化学式文本；后台异步累积 LLM 生成 + 人工校验入库 |
| R6 | preview 与 content SSE 顺序交错导致 UI 闪动 | 中 | graph 强约束：answer flush 完成后再 emit preview；前端按事件类型独立 append |
| R7 | 历史会话 package 已被清理 | 低 | message 带 `package_summary`；缺失展示"演示已过期，重新生成" |
| R8 | 短链被爬虫枚举 | 低 | 32 位 nanoid + 速率限制；可选私密短链需登录访问 |
| R9 | 前端 evaluator 安全（执行 LLM 表达式） | 中 | `expr-eval` 严格白名单；不用 `Function`/`eval`；变量域固定 |
| R10 | A/B/C/D 同期落地导致 PR 体量过大 | 高 | 按里程碑切分 PR；每个 M 单独可合并；feature flag（`features.chemistry / features.collab`）默认 off，逐步放开 |

---

## 11. 验证清单

### 11.1 自动化
- `go build ./...`、`go test ./...` 全绿
- `make test-integration`：发起 chat → 收到 content + preview(complete) → message 落库带 package_id → recompute 端点更新 outcomes → replay 端点回放事件
- `cd web/snowy-web && npm run lint && npm run build` 通过
- 新增：化学配平单测（10+ 反应方程式）、R3F scene 截图回归（Playwright 视觉对比，容差 5%）、collab WS 集成测（2 client 同步参数）

### 11.2 手动场景矩阵

| 场景 | 验收点 |
|---|---|
| 平抛运动 | R3F 渲染、v0/h 滑块毫秒响应、追问"加阻力"触发 regenerate |
| 单摆 | spring 场景视觉补完后，螺旋几何 + 衰减动画 |
| 光合作用 | 生物图谱 + 光强/CO₂ 滑块 → 即时曲线 |
| 中和反应 | 2NaOH + H2SO4，配平动画 + 球棍模型 + 离子流动 |
| 氧化还原 | Fe + CuSO4，电子转移箭头 + 半反应高亮 |
| 电解水 | 电极 + H2/O2 气泡 + 体积比 2:1 |
| WebGL 禁用 | 自动落到 Native → Sandbox → Fallback |
| 移动端 375px | 抽屉式参数 + 单栏堆叠 |
| 分享短链 | 生成 → 新隐身窗口打开 → 看到当时参数 |
| 实时协同 | 两 tab 加入同房间 → A 拖滑块 → B 30ms 内同步 → A 触发 regenerate → B 收到新 package |
| SSE replay | 历史会话点"重放动效" → preview 事件按 seq 顺序回放 |

---

## 12. 非本期范围（明确排除，避免范围漂移）

- 移动端原生 App 适配；
- 国际化文案；
- VR/AR 模式与 HDRI 自制资产；
- 量子化学 / 真分子轨道计算；
- 有机反应电子推动机理动画；
- 跨房间广播（教师全班推送）→ v8；
- 协同语音 / 视频 / 富文本编辑 → v8；
- 多模型 ensemble 仲裁机制升级 → v8；
- 演示卡片导出 GIF / MP4 → v8。

---

## 13. 已确认决策（截至 2026-05-16）

A 序列：
1. 单一入口：`POST /api/v1/agent/chat`
2. 追问 regenerate：LLM 二分类 + 关键词兜底
3. 前端 evaluator：`expr-eval`
4. `message.package_id`：nullable 迁移
5. SSE replay：本期实现，`agent_message_events` 表落库

B/C/D 序列：
6. R3F 视觉补完范围：六场景全覆盖 + 公共 stage/postFX/HUD 升级 + 低端机自动降级；不引入第三方 HDRI 资产
7. 化学学科本期覆盖范围：高中无机水溶液 + 小分子气相；50 个高频分子内置 SMILES 表；balancer 走 `gonum/mat` 线性代数
8. 化学契约：包装进现有 `GenerativeModelPackage`（`domain=chemistry` + `SimulationLogic` 内嵌），不新增 top-level 字段
9. 协同档位：必交付 D1（静态分享）+ D2（实时同步）；不引入 yjs/automerge，last-write-wins
10. 协同安全：登录用户才可加入 collab；短链 32 位 nanoid；feature flag `features.collab` 控制总开关
11. feature flag：`features.chemistry`、`features.collab` 默认 off，按里程碑灰度

---

## 14. 文档与代码追踪
- 本文档（v7）为本期 source of truth；过程记录于 session plan `~/.copilot/session-state/3d57cbdc-fd81-4071-8a40-9b4d07131320/plan.md`。
- SQL todos 表持续同步 M1–M16 状态；每完成一个 milestone 在 PR 描述中引用对应 §。
- v6 文档 (`snowy-v6-modeling-visual-upgrade.md`) 保留为视觉设计参考，不再独立推进；其未完成项被 v7 §4 吸收。
