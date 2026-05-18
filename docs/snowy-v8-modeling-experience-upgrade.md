# Snowy v8 · 建模体验质感升级 + 人教版教学层方案

> 状态：草案，等待 review（2026-05-17）
> 范围：`web/snowy-web/components/physics/r3f/**`、`web/snowy-web/components/biology/illustrations/**`、`web/snowy-web/components/generative/**`、`web/snowy-web/app/modeling/**`、新增 `web/snowy-web/components/chemistry/**`、新增 `docs/curriculum/pep/**`
> 上游：
> - [`docs/snowy-v6-modeling-visual-upgrade.md`](snowy-v6-modeling-visual-upgrade.md) · 学科色 token、R3F 骨架、语义化生物图
> - [`docs/snowy-v7-conversational-modeling-platform.md`](snowy-v7-conversational-modeling-platform.md) · §4 R3F 视觉补完清单、§5 化学契约、§7 跨能力契约
> 走查证据：[`output/playwright/baseline-v8/`](../output/playwright/baseline-v8/) · 共 12 张截图，对应本期"丑基线"

---

## 0. 文档定位与边界

### 0.1 v8 = "v7 §4/§5 的落地细则 + 教学权威层"

| v8 章节 | 与 v7 关系 | 性质 |
|---|---|---|
| §1 真实体验问题清单 | 全新 | v8 首发：playwright 走查 12 张截图归因 |
| §2 物理 6 场景质感升级矩阵 | **v7 §4 落地细则** | v7 给清单，v8 给文件级 diff 方向 + 资源底线 |
| §3 生物 SVG 视觉增强（保留 2D） | v7 未覆盖 | 全新：用户决策——不做 3D 化 |
| §4 化学建模 UI 落地 | **v7 §5 前端落地** | v7 §5 给后端契约，v8 给 R3F 化学渲染器 |
| §5 人教版课纲映射 + 易错点库 | 全新 | 教学权威层，v7 未覆盖 |
| §6 性能降级与设备分层 | v7 §4.3 微提 | v8 给完整分级表 |
| §7 里程碑与验收（含视觉回归基线） | 全新 | v8 收口标准 |

### 0.2 明确不在 v8 范围（避免范围漂移）

| 排除项 | 归属 |
|---|---|
| 协同/分享 / WebSocket hub | v7 §6（暂缓） |
| SSE replay / `agent_message_events` | v7 §3.x（已部分落地） |
| LLM 路由与模型切换 | v5 |
| VR / AR 模式 | 后续版本 |
| 自制 HDRI / 真实贴图资产采买 | v8 仅用 drei 内置 `Environment preset` |

### 0.3 决策已锁（与用户对齐 2026-05-17）

| # | 决策项 | 决议 |
|---|---|---|
| 1 | 物理建模优化幅度 | **保留现有 6 场景** + 质感升级（材质 + 粒子 + HUD），不做大规模重建 |
| 2 | 生物建模优化幅度 | **保留 SVG 2D**，做渐变/微动效/即时回流增强，不做 3D 化 |
| 3 | 化学建模 | 全新落地（沿用 v7 §5 后端契约） |
| 4 | 教育层 | 引入**人教版课纲映射 + 易错点库**作为演示卡标配 |
| 5 | 资源底线 | drei 内置 `Environment preset` + procedural shader/noise，**不采买外部贴图** |
| 6 | 走查基线 | 入库 `output/playwright/baseline-v8/`，作为 v8 实施期视觉回归对照 |

---

## 1. 真实体验问题清单（playwright 走查归因）

走查方式：通过本地启动的 Playwright MCP + Chrome 内核访问 `http://localhost:3001/`，逐 Tab 体验 6 个学习场景，共抓 12 张截图入库。

### 1.1 P0 · 致命视觉 Bug（必须最先修）

| # | 现象 | 截图 | 量化数据 | 根因（文件） |
|---|---|---|---|---|
| **P0-1** | **3D 画布被压扁成扁条**，平抛球几乎只能左右看，看不到抛物线全貌 | [`07-physics-canvas-zoomed.png`](../output/playwright/baseline-v8/07-physics-canvas-zoomed.png) [`10b-physics-spring-canvas.png`](../output/playwright/baseline-v8/10b-physics-spring-canvas.png) | `canvas.height = 150 px`（shell 上限 `min-height:360px` 也守不住） | `web/snowy-web/components/physics/r3f/R3FPhysicsPreview.tsx:104` 的 `<Canvas>` 没有显式 `style={{height:'min(60vh, 540px)'}}`，被 `<Space direction="vertical">` 父级塌缩 |
| **P0-2** | 编译耗时 60s+ 无任何进度反馈，用户以为卡死 | [`02-modeling-empty.png`](../output/playwright/baseline-v8/02-modeling-empty.png) | 推演命令发出 → stage 卡在 `reasoning` ≥ 60s | `web/snowy-web/app/modeling/page.tsx:159` 处只做了 stage 切换，没有流式 token 进度 |
| **P0-3** | 教学层完全缺失：演示卡没有公式推导步骤、没有课本/考纲条目对应 | [`08-biology-genetics-real.png`](../output/playwright/baseline-v8/08-biology-genetics-real.png) | "AI 教练" 仅有 3–5 行散文式 summary，无知识点编号、无易错点 | `web/snowy-web/components/generative/GenerativePhysics3DCanvas.tsx`、`InteractionPlanPanel.tsx` 均不引用任何课纲数据源 |

### 1.2 P1 · 视觉粗糙点（影响"炫酷度"）

| # | 现象 | 截图 | 根因 |
|---|---|---|---|
| P1-1 | 3D 场景"塑料感"——所有物体都是 `meshStandardMaterial` + cyan emissive，金属/玻璃/光泽都长一样 | `06`、`10b` | 6 个 scene 文件均未用 `MeshTransmissionMaterial` / `Environment` / HDRI；`R3FStage.tsx:22-43` 只有一束 directionalLight + 2 束彩色 pointLight |
| P1-2 | 后处理（Bloom）开了但效果很弱，主体球的发光跟背景对比度低 | `06`、`10b` | `postFX.tsx:22` Bloom intensity 0.55，`luminanceThreshold 0.4`，未挂 `ToneMapping`/`Vignette` |
| P1-3 | 尾迹是一根纯色线段，没有渐隐、没有粗细变化、没有"速度感" | `06` | `R3FTrail.tsx:55-58` 用 `lineBasicMaterial`，固定 `opacity:0.85`，没有 gradient |
| P1-4 | 碰撞场景虽实现了 60 个点的爆点粒子，但粒子全是同色同大小 | （需新生成 collision package） | `CollisionScene.tsx:91-94` `pointsMaterial size:0.12` 固定，无随时间衰减、无颜色梯度 |
| P1-5 | 物理 HUD 是右上角一个小标签，数据只有 3–4 行，**缺关键的实时公式**（如 `F=ma`、`v=v₀+at`） | `06` | `R3FHud.tsx:22-67` 仅输出 `{label, value}` 二元组，未嵌入 LaTeX/MathJax |
| P1-6 | 生物 SVG 整体偏小学课本风：纯色 `#86efac → #16a34a` 线性渐变 + dashed 光线，无玻璃感、无微粒、无脉动 | [`09-biology-photosynthesis.png`](../output/playwright/baseline-v8/09-biology-photosynthesis.png) | `Photosynthesis.tsx:27-156` 用 SVG 直绘，仅 `radialGradient` + `linearGradient`，没有 `filter`、没有 `<animateMotion>`、没有 CSS keyframe 增强 |
| P1-7 | 遗传图谱是 ReactFlow 默认蓝白节点，零美术语言 | [`08-biology-genetics-real.png`](../output/playwright/baseline-v8/08-biology-genetics-real.png) | `GenerativeBiologyGraph.tsx:30-39` 用 antd 默认 `Node.style`，未引入 v6 token |

### 1.3 P2 · 教学权威与可用性

| # | 现象 | 根因 |
|---|---|---|
| P2-1 | **没有化学建模入口**——用户进 `/modeling` 只有"物理 / 生物"两个 Segmented 选项 | `modeling/page.tsx:290-294`、`api.ts:917-921` 已有 `chemistryAnalyze/Balance` API，但前端没接入 |
| P2-2 | 证据条只显示 `runtime_grounding` 类型，看不出对应人教版章节 | `modeling/page.tsx:402-417` 直接渲染 `source_type` 字符串，没有课纲映射查表 |
| P2-3 | "可信度 55%" 低置信度无任何提示如何排查 | `pkg.validation_report` 已有 4 个 valid 标志，但 UI 仅展示绿/红 Tag |
| P2-4 | 参数滑块在 P0-1 影响下基本不可用——用户拖动看不到 3D 反馈 | 与 P0-1 同源 |

### 1.4 现状盘点总结

| 维度 | 当前水位 | Gemini 演示参考水位 | 差距等级 |
|---|---|---|---|
| 3D 渲染分辨率 | 790×150（被压扁） | 等比 16:9 ≥ 540px | 🔴 致命 |
| PBR / 光照 | 1 directional + 2 point + 0 HDRI | HDRI + ContactShadows + multi-bounce | 🟠 重 |
| 后处理 | Bloom 弱 | Bloom + ToneMapping + Vignette + SSAO | 🟡 中 |
| 粒子/尾迹 | 单色 BufferGeometry | 渐隐 + 颜色梯度 + 速度映射 | 🟠 重 |
| HUD 信息密度 | 3–4 行键值 | 实时 LaTeX 公式 + 数值动画 | 🟠 重 |
| 教学权威层 | 无 | 课本章节定位 + 易错点弹幕 | 🔴 致命（教育属性缺失） |
| 化学建模 | 无 UI | （Gemini 有球棍 + 配平动画） | 🔴 致命 |
| 生物可视化 | 480×260 SVG | （Gemini 用 R3F 但教学场景下 2D 已够） | 🟡 中（保留 2D，增强材质） |
| 设备分层 | 仅 prefers-reduced-motion | tier-based: low/mid/high | 🟡 中 |

---

## 2. 物理 6 场景质感升级矩阵

> 决策：**保留现有 6 场景**（Motion / Projectile / Orbit / Spring / Collision / Force），只做材质 + 粒子 + HUD 的质感升级；不做场景拆分、不做骨架重写。

### 2.1 公共升级（一次改动惠及全部 6 场景）

#### 2.1.1 修复 P0-1 · Canvas 显式高度

**文件**：`web/snowy-web/components/physics/r3f/R3FPhysicsPreview.tsx`

```tsx
// L102-110 当前
<div className="snowy-r3f-shell" data-scene={sceneKind}>
  <Canvas className="snowy-r3f-canvas" ...>

// → 改成
<div
  className="snowy-r3f-shell"
  data-scene={sceneKind}
  style={{ height: 'clamp(420px, 60vh, 640px)' }}   // ★ 显式高度
>
  <Canvas
    className="snowy-r3f-canvas"
    style={{ width: '100%', height: '100%' }}        // ★ 撑满
    ...
  >
```

同时在 `r3f.css` 把 `.snowy-r3f-shell` 的 `min-height: 360px` 提到 `420px` 兜底；移动端 (`@media max-width:768px`) 设为 `clamp(320px, 56vh, 480px)`。

**验收**：刷新所有现有 package，canvas DOM height ≥ 420 px。

#### 2.1.2 升级 `R3FStage` · 加 HDRI 与软阴影

**文件**：`web/snowy-web/components/physics/r3f/lib/R3FStage.tsx`

| 改动 | 代码方向 |
|---|---|
| 引入 drei `<Environment preset="city">` | 不增加资源采买；orbit 用 `preset="night"`，collision 用 `preset="studio"`，spring/force/motion 用 `preset="apartment"`，projectile 用 `preset="park"` |
| 加 drei `<ContactShadows>` | `position={[0,-0.001,0]} opacity={0.5} blur={2.5} far={6}`；orbit 不挂 |
| 加 drei `<SoftShadows size={25} samples={16}>` | 仅 `quality !== 'eco'` 时挂载 |
| 改 ambient/directional 强度 | 让 HDRI 主导，directional 降到 0.6，ambient 降到 0.15；pointLight 学科色保留作"舞台灯" |
| 加 tone mapping | `<Canvas gl={{ toneMapping: THREE.ACESFilmicToneMapping, toneMappingExposure: 1.15 }}>` |

#### 2.1.3 升级 `postFX.tsx` · 三件套

| 效果 | 参数 | 触发档位 |
|---|---|---|
| `Bloom` | `intensity: 0.85, luminanceThreshold: 0.65, luminanceSmoothing: 0.2, mipmapBlur: true` | standard / high |
| `ToneMapping` | `mode: ACESFilmic` | always |
| `Vignette` | `eskil: false, offset: 0.4, darkness: 0.55` | standard / high |
| `DepthOfField` | 现有 | high only |
| `SSAO` | `radius: 0.08, intensity: 8` | **high only**（标准/省电不挂） |

#### 2.1.4 重写 `R3FTrail` · 渐隐 + 速度色阶

**文件**：`web/snowy-web/components/physics/r3f/lib/R3FTrail.tsx`

- 从 `lineBasicMaterial` 改成 **drei `<Trail>` 组件**（drei 10.x 内置）或 `LineGeometry + Line2`（meshline 风格）。
- 颜色按"速度幅值" gradient：`from #22d3ee (慢) → #f43f5e (快)`，每帧根据 `simRef.bodies.main.linvel()` 重算。
- Alpha 沿尾迹由头到尾 1.0 → 0，最大宽度 0.06，最小 0.005。

#### 2.1.5 重写 `R3FHud` · 加实时 LaTeX 公式

**文件**：`web/snowy-web/components/physics/r3f/lib/R3FHud.tsx` + 新增 `R3FFormulaCard.tsx`

| 改动 | 描述 |
|---|---|
| 引入 `katex` + `react-katex` | 已有 ant-design markdown 链路里若有则复用；否则新增（~70 KB gz） |
| HUD 上半部 = 现有键值 | 保留 |
| HUD 下半部 = **实时公式卡** | 例：`force` 场景显示 `\large F = ma = 2.0 \times 3.0 = 6.0\;\text{N}`，公式右侧数值绿色高亮，参数变化时 0.2s ease 过渡 |
| 学科色背景 | `--color-physics: oklch(0.62 0.16 220)`（v6 token），HUD 边框 1px 学科色 + glass blur |
| 移动端折叠 | `@media max-width:768px` HUD 折叠成右上角 ⓘ 按钮，点击展开为 bottom sheet |

#### 2.1.6 新增公共"教学层"侧栏（在 HUD 之外）

参见 §5 教学层。每个 3D 场景顶端浮动一条 **课纲徽章 chip**：
> `🎓 人教版 · 必修2 · §3.1 平抛运动`

### 2.2 场景级专项升级（6 场景差异化）

| Scene | 当前主问题 | v8 必交付要素 | 涉及文件 |
|---|---|---|---|
| **Orbit** | 行星只是发光小球，无大气、无环带、无星轨 | 1) 主星挂 `MeshDistortMaterial` + `Sparkles`<br>2) 行星用 `noise shader` 程序贴图（procedural）<br>3) 轨道线改 `Tube` + 渐变<br>4) 卫星尾巴 `Trail` 拉长到 600 点 | `scenes/OrbitScene.tsx` + 新 `lib/materials/StarSurface.ts` |
| **Projectile** | 单球 + cyan 尾迹，无发射器、无落点反馈 | 1) 起点加发射器（红橙金属圆柱 + 尖端发光）<br>2) 球过最高点时短暂闪一下（lerp emissive×2）<br>3) 落地触发 `Burst` 粒子（`Sparkles` + `setTimeout` 600ms）<br>4) 矢量箭头分解 `v_x / v_y`（双色） | `scenes/ProjectileScene.tsx` + 新 `lib/effects/Burst.tsx` |
| **Spring** | 螺旋已有，但缺墙面纹理、缺振动残影 | 1) 墙面改 `Brick` procedural shader<br>2) 球残影：每 6 帧抓一次 mass.position，叠 6 个透明球（alpha 渐降）<br>3) 弹簧拉伸时金属色温改红→蓝→紫表能量比 | `scenes/SpringScene.tsx` |
| **Collision** | 粒子爆点单色 | 1) 粒子按 `dt` 颜色 `#fde68a → #f97316 → 透明`<br>2) 粒子 size 用 `sizeAttenuation` 从 0.15→0.02<br>3) 加 "碰撞瞬间" 屏幕级镜头抖动 0.06s | `scenes/CollisionScene.tsx` |
| **Force** | 方块单色无阴影 | 1) 改用 `<RoundedBox>` (drei)<br>2) `<ContactShadows>` 默认开<br>3) 矢量箭头加 normal/friction/applied 3 色 4 箭头分解 | `scenes/ForceScene.tsx` + 复用 `FrameVector` |
| **Motion** | fallback 太简陋 | 1) 球升级到 `MeshTransmissionMaterial`<br>2) 切向 / 法向加速度箭头双色 | `scenes/MotionScene.tsx` |

### 2.3 验收（视觉回归）

| Check | 工具 | 通过线 |
|---|---|---|
| 6 场景 canvas DOM height | playwright | ≥ 420 px |
| 6 场景包含 HDRI | three 调试器 / `scene.environment !== null` | true |
| 后处理 EffectComposer 含 ≥3 个 effects | three.inspector | true |
| HUD 含 LaTeX 公式节点 | DOM 查询 `.katex` | 存在 |
| 视觉回归 | 与 [`baseline-v8/`](../output/playwright/baseline-v8/) 对比 | 6 张截图明显改观，人工评分 ≥ 4 / 5 |

---

## 3. 生物建模视觉增强（保留 2D SVG）

> 决策：不做 3D 化，**保留**现有 7 个 SVG 插画（`Photosynthesis / Respiration / Enzyme / Synapse / Genetics / Ecosystem / Membrane`），做"视觉档次升级"。

### 3.1 现状盘点

[`components/biology/illustrations/Photosynthesis.tsx`](../web/snowy-web/components/biology/illustrations/Photosynthesis.tsx) 等 7 文件：
- viewBox 都是 480×260（太小）
- 仅用 `linearGradient` / `radialGradient`，无 `filter`（缺玻璃 / 模糊 / 发光）
- 流动用 `strokeDasharray` 实现，效果生硬
- 颜色风格"小学课本"——纯色平涂 + 黑描边

### 3.2 全局升级清单

| 改动 | 文件 | 落地方向 |
|---|---|---|
| viewBox 扩到 720×420 | 7 个 illustration | 同步 css `.snowy-bio-illust__svg max-width:100%` |
| 引入 SVG `<filter>` 套装 | 新 `illustrations/filters.tsx` | `glow`（feGaussianBlur+feMerge）、`glass`（feBlend）、`grain`（feTurbulence） |
| 引入 CSS keyframes 库 | `illustrations.css` 新增 | `@keyframes flow-fast`、`pulse-soft`、`shimmer`、`drift`，按学科色 token 引用 |
| 主背景从纯绿渐变改"晨光感" | 7 个 illustration | `linear-gradient(160deg, #f0fdf4 0%, #ecfeff 60%, #fef3c7 100%)` + `feTurbulence` 噪声叠加 0.04 opacity |
| 所有箭头/光线 → 带 `marker-end` 渐变 + 流动微粒 | 7 个 illustration | 不再用 `dashed`，改用 `<circle>` 沿路径 `<animateMotion>` |
| **滑块即时回流**——所有数值变化 ≤ 50ms 反映到 SVG | `Photosynthesis.tsx:20` 改用 `useDeferredValue` + CSS variable | 滑块改变 `--light-intensity`，SVG 通过 `style="opacity: calc(0.3 + var(--light-intensity)/200)"` 直接响应，省 React diff |

### 3.3 7 张插画专项

| 插画 | 主要痛点 | v8 必交付增强 |
|---|---|---|
| **Photosynthesis** | 太阳是个圆 + dashed 光线 | 太阳挂 `glow` filter；光线改 `<circle>` 沿光路 `<animateMotion>` 形成"光子粒子流"；叶绿体内部加微泡（H₂O→O₂）；速率曲线带高亮"当前点"圆点 |
| **Respiration** | 线粒体造型呆 | 线粒体用 SVG path + glass filter；ATP 释放用 `pulse-soft` 关键帧；NADH 颜色加 shimmer |
| **Enzyme** | 锁钥模型生硬 | 加底物 fitting 动画（CSS transform `translateY` + `scale`）；中间体颜色变化（绿→黄→红）；产物释放粒子 |
| **Synapse** | 静态突触 | 神经递质囊泡按 `--firing` 变量沿轴突路径 motion；接收端 ion channel `glow` 闪烁 |
| **Genetics** | Punnett 方格无层次感 | F1/F2 用 glass card + soft shadow；显隐性用 颜色 + 大小阶梯；比例条带刻度数字 |
| **Ecosystem** | 食物链平铺 | 能量金字塔加 perspective transform；箭头加 `flow-fast` 微粒；10% 损耗用红色"被截断"动效 |
| **Membrane** | 磷脂双层呆板 | 双层磷脂头加 `shimmer`；蛋白通道脉动；扩散/主动运输不同箭头颜色与速率 |

### 3.4 生物 ReactFlow 概念图升级

[`GenerativeBiologyGraph.tsx:30-39`](../web/snowy-web/components/generative/GenerativeBiologyGraph.tsx)：
- 节点 style 改为引用 v6 token：`background: var(--color-biology-soft); color: var(--color-biology); border: 1px solid var(--color-biology-border); box-shadow: var(--shadow-card-1)`
- 边动画 `animated: true` 保留，但 stroke 改 `var(--color-biology)` 渐变
- 增加 `MiniMap` 与 `Background variant="dots"`
- 边类型按 `relation` 字段映射：`促进 → 绿色实线箭头`、`抑制 → 红色 ⊣`、`转化 → 双向虚线`

### 3.5 验收

- 7 张插画 viewBox ≥ 720×420
- 调动滑块 100% 数值能在 SVG 即时反映（playwright `evaluate` 拖滑块后断言 SVG style 变化）
- 与 [`09-biology-photosynthesis.png`](../output/playwright/baseline-v8/09-biology-photosynthesis.png) 对比，人工评分 ≥ 4 / 5

---

## 4. 化学建模 UI 落地

> 沿用 v7 §5 已设计的后端契约（`ChemistryReactionPackage`、`/api/v1/modeling/chemistry/balance|analyze`）。v8 任务：**把它接到前端**。

### 4.1 前端入口

| 改动 | 文件 | 内容 |
|---|---|---|
| Segmented 加"化学"档 | [`app/modeling/page.tsx:290-294`](../web/snowy-web/app/modeling/page.tsx) | `{ label: <Space><ExperimentOutlined />化学</Space>, value: 'chemistry' }` |
| 新增 chemistry 类型 | `modeling/page.tsx:42` | `type ModelingSubject = 'physics' \| 'biology' \| 'chemistry'` |
| 示例 chip | `subjectExamples` | `['2NaOH + H2SO4 → ?', '电解水', '铁与硫酸铜反应']` |
| 分支渲染 | `modeling/page.tsx:472-473` | `pkg.domain === 'chemistry' && <GenerativeChemistryCanvas pkg={pkg} values={values} />` |

### 4.2 新增组件树

```
web/snowy-web/components/chemistry/
├── GenerativeChemistryCanvas.tsx          // 顶层入口，按 reaction_type 路由
├── BallStickMolecule.tsx                  // R3F 球棍模型（原子 = 彩色球，键 = 圆柱）
├── ReactionStageBar.tsx                   // 反应物 → 中间态 → 产物 进度条
├── BalanceAnimation.tsx                   // 配平过程：系数从 ? 变成正确数字
├── ElectronTransferArrow.tsx              // 氧化还原电子流（R3F 箭头沿键移动）
└── chemistry.css
```

### 4.3 关键技术点

| 能力 | 实现 | 依赖 |
|---|---|---|
| 球棍模型 | R3F：原子 `<Sphere>` 用 CPK 配色（H 白、O 红、N 蓝、C 黑、Na 紫…），键用 `<Cylinder>` 在两原子间 lookAt | 复用现有 `@react-three/fiber` + drei |
| 分子布局 | 简单反应直接用后端给的 `atoms[].pos` 字段（v7 §5 契约定义），无 pos 则前端用 `dagre` 二维布局 fallback | 已有 dagre |
| 配平动画 | 系数 `?` → 正解的过程：每个分子整体 `scale 0.7→1.0` + 摩擦音效（可选） | 纯 css + react-spring |
| 电子转移 | `<Trail>` 组件 + 沿键路径 `useFrame` 插值；多电子用 instancedMesh | drei |
| 浓度/温度滑块 | 同物理：值 → 球的发光强度 / 反应速率 | 共享 `InteractionPlanPanel` |
| 反应类型路由 | 后端 `reaction_type ∈ { neutralization, redox, substitution, decomposition, electrolysis, organic_addition }` | v7 §5 已定义 |

### 4.4 验收

- `/modeling?type=chemistry&q=电解水` 能渲染出 H₂O 球棍 → H₂ + O₂ 分离 + 电子流动画
- 配平错误时（如 `H2 + O2 → H2O`）后端返回 `balance_error`，前端高亮系数 chip 红色 + 教练提示

---

## 5. 人教版课纲映射 + 易错点库（教学权威层）

> v8 新增。回答用户的"更专业、更具教育意义"需求。

### 5.1 数据契约设计

**新建目录**：`docs/curriculum/pep/`（PEP = People's Education Press，人教版）

```
docs/curriculum/pep/
├── README.md
├── schema.json                  // YAML / JSON schema 定义
├── physics/
│   ├── compulsory-1.yaml       // 必修1
│   ├── compulsory-2.yaml       // 必修2
│   └── elective-3-x.yaml       // 选择性必修
├── biology/
│   ├── compulsory-1.yaml
│   ├── compulsory-2.yaml
│   └── ...
├── chemistry/
│   └── ...
└── pitfalls/
    ├── physics.yaml             // 易错点库
    ├── biology.yaml
    └── chemistry.yaml
```

**条目 schema**（兼顾版权安全 —— 仅录章节编号 + 自有易错点文案，不复制课本原文）：

```yaml
# docs/curriculum/pep/physics/compulsory-2.yaml
version: pep-2019    # 课纲版次（公开信息）
subject: physics
book: compulsory-2
chapters:
  - id: ch5
    title: 抛体运动
    sections:
      - id: ch5.1
        title: 曲线运动
        knowledge_tags: [curved_motion, velocity_direction]
      - id: ch5.4
        title: 平抛运动
        knowledge_tags:
          - projectile_motion
          - horizontal_uniform
          - vertical_freefall
        learning_goals:
          - 理解平抛运动的两个分运动
          - 推导落点 x = v₀t, y = ½gt²
          - 计算落地速度与方向
        # 不存原文，只存编号 + 自有教学解释（无版权问题）
```

```yaml
# docs/curriculum/pep/pitfalls/physics.yaml
pitfalls:
  - id: phy.projectile.001
    knowledge_tags: [projectile_motion]
    pitfall: 误以为平抛运动水平方向有加速度
    correct: 水平方向只有惯性，加速度恒为 0；只有竖直方向受重力 g
    common_mistake: 写出 vx = v0 + at（错）
    correction_hint: vx 恒等于 v0；at 只对 vy 适用
    severity: high
    evidence_section: pep.physics.compulsory-2.ch5.4
  - id: phy.projectile.002
    knowledge_tags: [projectile_motion]
    pitfall: 把"水平射程"算成"轨迹长度"
    ...
```

### 5.2 前端集成

#### 5.2.1 课纲映射加载

新增 `web/snowy-web/lib/curriculum.ts`：

```ts
// 启动时通过 next.js static import 一次性载入 ~50KB YAML（解析后 JSON）
import physicsCh from '@/data/curriculum/physics-index.json';
// 提供：
export function lookupByTags(tags: string[]): CurriculumRef[];
export function lookupPitfalls(tags: string[]): Pitfall[];
```

后端 `evidence_refs` 已经带 `knowledge_tags`，前端用 tags 反查课纲编号。

#### 5.2.2 演示卡新增"教学侧栏"

新建组件 `web/snowy-web/components/learning/`：

| 组件 | 位置 | 功能 |
|---|---|---|
| **`CurriculumBadge.tsx`** | 演示卡顶端 chip 条 | 显示 `🎓 人教版·必修2·§5.4 平抛运动` + 点击展开"学习目标" |
| **`PitfallList.tsx`** | 右侧 AI 教练区"模型校验"下方 | 罗列匹配本题的 2-3 个易错点，红色 ⚠ + 一句话纠错 |
| **`KnowledgeMap.tsx`**（可选） | "完整证据"折叠区内 | mini 思维导图：当前知识点 ↔ 前置 / 后续节点 |

#### 5.2.3 与现有 UI 的关联点

- [`modeling/page.tsx:382-393`](../web/snowy-web/app/modeling/page.tsx) 证据条：在"基于 N 条证据"后追加 `<CurriculumBadge tags={pkg.learning_model.knowledge_tags} />`
- [`InteractionPlanPanel`](../web/snowy-web/components/generative/InteractionPlanPanel.tsx) 顶部插入 `<PitfallList tags={...} />`
- `/ask` 答案气泡如附带 demo，同样挂 `<CurriculumBadge>`

### 5.3 数据制作流程（无版权风险）

1. **章节编号**：来自人教版课本目录（公开信息），仅存 `compulsory-2.ch5.4` 这类编号 + 章节标题（不复制正文）。
2. **学习目标**：用 GPT-4 类工具基于课纲条目自动生成，人工 review；落在自有文件，受我们版权保护。
3. **易错点库**：自有教研 / 社区贡献（参考 [OpenCurriculum 风格](https://github.com/) 通用易错点列表 + 自有补充），无外部原文。
4. **持续更新**：YAML 文件入 git，PR 流程，社区可贡献。

### 5.4 验收

- 任一物理题（已知 knowledge_tags）打开演示卡，能看到课纲徽章 + ≥1 个易错点
- `docs/curriculum/pep/physics/compulsory-2.yaml` 至少覆盖必修2 全部章节编号 + 学习目标
- `docs/curriculum/pep/pitfalls/physics.yaml` 至少 30 个高频易错点

---

## 6. 性能降级与设备分层

### 6.1 设备分层 hook

新增 `web/snowy-web/lib/useDeviceTier.ts`：

```ts
export type DeviceTier = 'low' | 'mid' | 'high';

export function useDeviceTier(): DeviceTier {
  // 判定输入：
  //   - navigator.hardwareConcurrency (cores)
  //   - navigator.deviceMemory (GB, Chrome)
  //   - navigator.connection.effectiveType
  //   - prefers-reduced-motion
  //   - prefers-reduced-data
  //   - 实测帧率（首屏 2s 采样）

  // 输出表
  //   low:  cores≤2 || mem≤2 || effectiveType in {2g,slow-2g,3g} || reduced-motion || reduced-data
  //   high: cores≥8 && mem≥8 && effectiveType==4g
  //   mid:  其他
}
```

### 6.2 三档画质映射

| 能力 | low | mid | high |
|---|---|---|---|
| Canvas DPR | 1 | 1.5 | 2 |
| HDRI Environment | ❌ | ✅ preset | ✅ preset |
| ContactShadows | ❌ | ✅ blur=2 | ✅ blur=2.5 + SoftShadows |
| Bloom | ❌ | ✅ intensity 0.65 | ✅ intensity 0.95 |
| ToneMapping | ✅ | ✅ | ✅ |
| Vignette | ❌ | ✅ | ✅ |
| SSAO / DoF | ❌ | ❌ | ✅ |
| Trail max points | 60 | 300 | 600 |
| Particles | 0 | 30 | 90 |
| Procedural shader | 简版 | 标准 | 高频 |
| 化学球棍 instanced | 否 | 是 | 是 |
| 生物 SVG `<filter>` | 关 | 开 | 开 |

### 6.3 优雅降级链（保留 v6/v7 已有）

```
R3F WebGL2 ────► R3F WebGL1 ────► NativePhysicsPreview (Canvas2D) ────► PhysicsPreviewSandbox ────► DemoFallbackCard
   ↑               ↑                    ↑                                    ↑                          ↑
 高/中端          中端                 低端 / WebGL 不可用                    iframe 隔离                文本兜底
```

`R3FPhysicsPreview.tsx:96-98` 已有该 fallback 入口，v8 补完低端机判断（`tier === 'low' && reducedMotion` 时直接走 Native）。

### 6.4 验收

- 在 Chrome `DevTools → Rendering → Reduce motion + Throttling: Slow 4G` 下，3D 场景仍 ≥ 24 fps（playwright 测帧）
- `tier === 'low'` 时 EffectComposer 不挂载（DOM 查询 `.postprocessing` 不存在）

---

## 7. 里程碑与验收

### 7.1 里程碑（无时间估计，仅依赖关系）

| M | 名称 | 依赖 | 关键产物 |
|---|---|---|---|
| **M0** | 体验基线入库 | 已完成（[`baseline-v8/`](../output/playwright/baseline-v8/)） | 12 张截图 + 本文档 |
| **M1** | P0 修复 + 公共升级 | M0 | §2.1（Canvas 高度、HDRI、postFX 三件套、Trail 渐隐、HUD 公式卡） |
| **M2** | 物理 6 场景专项升级 | M1 | §2.2 各场景独立 PR |
| **M3** | 生物 7 插画视觉增强 | M1（filters / keyframes 库） | §3 |
| **M4** | 课纲 + 易错点库 v1 | M0 | `docs/curriculum/pep/` + `lib/curriculum.ts` + `CurriculumBadge` + `PitfallList` |
| **M5** | 化学 UI 落地 | M1、v7 §5 后端已就绪 | §4 整个 chemistry 子树 |
| **M6** | 设备分层 + 降级链 | M1 | `useDeviceTier` + 三档矩阵 |
| **M7** | 视觉回归基线 + 整体收口 | M1-M6 | 12 张 baseline-v9 对比图 + 人工评分 |

并行可能：M2、M3、M4、M5 可在 M1 后并行；M6 与 M2-M5 也可并行。

### 7.2 验收清单（v8 整体）

| 维度 | Check | 通过线 |
|---|---|---|
| 视觉 | 6 物理场景 canvas height | ≥ 420px |
| 视觉 | 6 物理场景含 HDRI 环境贴图 | true |
| 视觉 | postFX ≥ 3 effects | true |
| 视觉 | HUD 含实时 LaTeX 公式 | DOM `.katex` 存在 |
| 视觉 | 7 生物 SVG viewBox ≥ 720×420 | true |
| 视觉 | 视觉回归 | 12 张对比图，人工平均 ≥ 4 / 5 |
| 教学 | 任一物理 / 生物 / 化学题 demo 卡可见 `CurriculumBadge` | true |
| 教学 | 任一物理 / 生物题命中 ≥ 1 易错点 | true |
| 教学 | `docs/curriculum/pep/physics/compulsory-2.yaml` 章节齐 | 100% |
| 教学 | `pitfalls/physics.yaml` ≥ 30 条 | true |
| 化学 | `/modeling?type=chemistry` 可用 | 球棍模型 + 配平动画 + 电子流 三选一可见 |
| 性能 | low tier 下 ≥ 24 fps | playwright 测帧 |
| 性能 | low tier 不挂 EffectComposer | DOM 验证 |
| 性能 | WebGL 不可用 → fallback NativePhysicsPreview | true |
| 工程 | 不引入外部图像 / HDRI 资产 | git diff 检查 |
| 工程 | 课纲 yaml 仅含编号 + 自有文案 | 人工 review |

### 7.3 视觉回归（playwright 脚本草案）

新增 `web/snowy-web/scripts/visual-regression.ts`：

```ts
// 12 个固定 package_id（取自 baseline-v8 截图同源）
const PACKAGES = [
  { id: '2a8c6a7e-...', name: 'physics-projectile' },
  { id: '0f4d1cef-...', name: 'physics-spring' },
  // ... 其余 10 个
];
for (const p of PACKAGES) {
  await page.goto(`/modeling?package_id=${p.id}`);
  await page.waitFor({ time: 4 });
  await page.screenshot({ path: `output/playwright/baseline-v9/${p.name}.png`, fullPage: true });
}
```

CI 中跑 `pnpm vr:snapshot`，PR 评审者可视对比 v8 vs v9。

---

## 8. 风险与对策

| 风险 | 概率 | 影响 | 对策 |
|---|---|---|---|
| Canvas 加 HDRI / postFX 后低端机帧率塌方 | 中 | 中 | §6 设备分层兜底，low tier 不挂 |
| KaTeX 包体 +70KB | 低 | 低 | 已在 next.js dynamic import 之列，可按路由懒加载 |
| 课纲 yaml 体积膨胀 | 低 | 低 | 拆分 per book，按需 dynamic import |
| 易错点匹配不准（tags 落空） | 中 | 中 | 兜底文案"暂未匹配到精准易错点"，给出"通用易错点 top-3" |
| 化学球棍 atoms[].pos 缺失 | 中 | 中 | 用 dagre 二维 fallback 布局；写入后端补全 prompt |
| Bloom 把白底页面"漂白" | 低 | 低 | Bloom 只挂 Canvas 内，不影响外层 |

---

## 9. 与既有文档的关系

| 文档 | v8 关系 |
|---|---|
| [`snowy-v6-modeling-visual-upgrade.md`](snowy-v6-modeling-visual-upgrade.md) | 复用：学科色 token、R3F 骨架、语义化生物图 |
| [`snowy-v7-conversational-modeling-platform.md`](snowy-v7-conversational-modeling-platform.md) | **v7 §4 = v8 §2 的 "WHAT"，v8 §2 = v7 §4 的 "HOW"**；v7 §5 后端 = v8 §4 前端 |
| [`snowy-v5-redesign-blueprint.md`](snowy-v5-redesign-blueprint.md) | 不变更（LLM 路由、可靠性） |
| [`prd.md`](prd.md) | v8 §5 教学层强化了 PRD 中"高中生 / 课本对照"诉求 |

---

## 10. 附录 · 走查证据清单

| 文件 | 内容 | 对应章节 |
|---|---|---|
| [`01-home.png`](../output/playwright/baseline-v8/01-home.png) | 首页全貌 | §1.1 上下文 |
| [`02-modeling-empty.png`](../output/playwright/baseline-v8/02-modeling-empty.png) | 建模空态 + 编译进度卡 60s | §1.1 P0-2 |
| [`03-physics-projectile.png`](../output/playwright/baseline-v8/03-physics-projectile.png) | 物理推演中 | — |
| [`04-learning.png`](../output/playwright/baseline-v8/04-learning.png) | 我的学习 · 历史 | — |
| [`05-learning-models.png`](../output/playwright/baseline-v8/05-learning-models.png) | 我的学习 · 模型包 Tab | — |
| [`06-physics-projectile-real.png`](../output/playwright/baseline-v8/06-physics-projectile-real.png) | **平抛 3D 渲染（被压扁的 canvas）** | §1.1 P0-1 / §1.2 P1-1~3 |
| [`07-physics-canvas-zoomed.png`](../output/playwright/baseline-v8/07-physics-canvas-zoomed.png) | canvas 元素特写 790×150 | §1.1 P0-1 |
| [`08-biology-genetics-real.png`](../output/playwright/baseline-v8/08-biology-genetics-real.png) | 生物 · 遗传 ReactFlow | §1.2 P1-7 |
| [`09-biology-photosynthesis.png`](../output/playwright/baseline-v8/09-biology-photosynthesis.png) | 生物 · 光合作用 SVG | §1.2 P1-6 |
| [`10-physics-spring.png`](../output/playwright/baseline-v8/10-physics-spring.png) | 弹簧振子（仍 150px） | §1.1 P0-1 复现 |
| [`10b-physics-spring-canvas.png`](../output/playwright/baseline-v8/10b-physics-spring-canvas.png) | 弹簧场景 canvas 特写 | §1.1 P0-1 复现 |
| [`11-ask.png`](../output/playwright/baseline-v8/11-ask.png) | 提问页 | — |

---

## 11. 已确认决策（截至 2026-05-17）

| # | 决策 | 决议 |
|---|---|---|
| 1 | 是否实际跑浏览器走查 | ✅ 已跑（Playwright MCP + Chrome via brew） |
| 2 | 物理优化幅度 | ✅ 保留 6 场景，仅做质感升级 |
| 3 | 生物优化幅度 | ✅ 保留 SVG 2D，做视觉增强（不做 3D 化） |
| 4 | 是否引入课纲映射 + 易错点库 | ✅ 引入 |
| 5 | 是否采买外部 HDRI / 贴图 | ❌ 不采买，全用 drei preset + procedural |
| 6 | 课纲数据合规策略 | ✅ 只录章节编号 + 自有易错点文案（无版权风险） |
| 7 | 化学是否纳入本期 | ✅ 纳入（v7 §5 后端契约前提下） |
| 8 | 协同 / 分享是否纳入本期 | ❌ 不纳入（v7 §6 暂缓） |

---

> _Maintained by Snowy 教研 + 工程组 · v8 草案 2026-05-17 · 等待 review_

