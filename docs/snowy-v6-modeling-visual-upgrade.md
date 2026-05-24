# Snowy v6 · 建模功能视觉与体验升级方案

> 状态：草案（待评审）
> 范围：`web/snowy-web`（前端） · 不改动后端 API 契约
> 配套：见 `docs/snowy-v4-redesign-blueprint.md` 游戏化学习链路、`docs/snowy-v5-redesign-blueprint.md` 多模型与可靠性、`docs/design/v6` 设计 tokens
> 说明：本文是历史视觉升级方案，若涉及检索增强或证据召回链路，均按规划能力理解；当前默认问答链路为 LLM 直答 + runtime grounding。

---

## 1. 背景

Snowy 当前已上线**知识问答 + 物理 / 3D 建模 + 生物建模**三大核心能力（见 `README.md` 与 `docs/prd.md`）。建模链路在工程上已具备：

- 物理：`components/physics/NativePhysicsPreview.tsx`（867 行，Canvas 2D 手写 3D 投影 + Rapier3D 物理）、`components/physics/PhysicsPreviewSandbox.tsx`（iframe 沙箱 fallback）、`components/generative/GenerativePhysicsCanvas.tsx`（SVG 折线 / 矢量箭头）。
- 生物：`components/biology/BiologyDiagram.tsx`、`components/generative/GenerativeBiologyGraph.tsx` 均基于 `@xyflow/react` 渲染彩色矩形节点。

但**视觉层"过于工程、过于扁平"**，与目标用户的体验期待存在显著差距。本方案沉淀针对该问题的设计与实施计划。

---

## 2. 用户画像与体验诊断

### 2.1 目标用户：高中生（15–18 岁）
- **视觉敏感**：成长于短视频 / 游戏 / 国漫氛围，对扁平、塑料感视觉容忍度低；
- **多端使用**：通勤 / 课间在手机和平板上学习，桌面学习集中在晚自习；
- **碎片化学习**：单次停留 5–15 分钟，依赖"即时反馈"与"看一眼就懂的图"维持注意力；
- **应试导向**：需要与课本对应的公式、矢量标注、变量关系，"好看"必须建立在"学得对"之上；
- **学科语义敏感**：他们记笔记会自己画细胞、画受力图；如果平台的图比他们自己画的还粗糙，留存与口碑会显著受损。

### 2.2 当前建模视觉的核心问题
1. **物理预览缺乏空间感**：`NativePhysicsPreview` 是手写 2D 投影 + 发光圆球，所有 sceneKind（orbit / spring / collision / projectile / force / motion）共用一套"星空背景 + 球 / 立方体 + 箭头"模板，缺乏材质、光照、景深，5 个场景看起来差不多。
2. **生成式物理画布过于扁平**：`GenerativePhysicsCanvas` 仅 SVG 折线 + 几条静态箭头，命中目标无任何反馈动效。
3. **生物图谱毫无形态语义**：光合作用、突触、遗传都被渲染为同一种 React Flow 矩形节点（仅靠 `nodeColorMap[node.type]` 区分颜色），无法承担"看一眼就懂"。
4. **设计语言割裂**：`globals.css` 是浅色暖中性 stone 色系，但 NativePhysics 强制深蓝宇宙渐变；首页 / modeling 页 / 物理预览 / 生物预览四种风格并存。
5. **状态体验缺位**：加载/空态多用基础 `Empty`，无骨架屏；移动端断点未单独打磨；缺少 `prefers-reduced-motion` 兼容。
6. **HUD 与参数面板"开发者审美"**：HUD 直接堆 `tag` 与小字，参数滑块缺学科色与即时数字反馈。

---

## 3. 设计原则

| # | 原则 | 含义 |
|---|---|---|
| 1 | **学科色统一** | 复用 `globals.css` 现有 token：物理 `--color-physics: #0891B2`、生物 `--color-biology: #16A34A`；建模预览的边光、滑块 track、HUD 强调色一律对齐学科色 |
| 2 | **语义化形态** | 生物节点按主题选 SVG 插画（叶绿体 ≠ 突触 ≠ DNA 链）；物理对象按 sceneKind 选 3D 模型与材质（恒星 / 弹簧螺旋 / 斜面 / 抛物） |
| 3 | **科技感但克制** | 适度 bloom / 粒子尾迹 / 矢量动画 / glass card；3D 预览舞台允许深色，其他区域保留浅色，避免整体过暗 |
| 4 | **即时反馈** | 参数滑块拖动时 3D 场景平滑 lerp，HUD 数值带 +/- 高亮 / 微抖；命中目标爆点 |
| 5 | **移动端友好** | 触屏旋转/缩放；按钮 ≥ 44px；HUD 与参数面板自动折叠为抽屉 |
| 6 | **渐进增强** | WebGL 不可用 / 设备性能弱 / 用户开"省电模式" / `prefers-reduced-motion` 时，回退现有 `NativePhysicsPreview` 与无动效路径 |
| 7 | **科学正确优先于美** | 任何视觉化不得改变物理量纲、矢量方向、变量含义；HUD 与公式标注与课本一致 |

---

## 4. 技术选型决策

| 维度 | 决策 | 备选 | 选定理由 |
|---|---|---|---|
| 物理 3D 渲染 | **`three` + `@react-three/fiber` + `@react-three/drei` + `@react-three/postprocessing`** | 继续用 Canvas 2D 手写投影 / 引入 Babylon.js | R3F 与 React 编程模型一致，drei 提供 OrbitControls/Trail/Html 等常用件；postprocessing 提供 Bloom/DoF；社区资源最丰富 |
| 物理仿真步进 | **保留 `@dimforge/rapier3d-compat`** | 切换 cannon-es / 自写积分 | Rapier 已在用，准确性与性能足够；只换"绘制"层 |
| 生物可视化 | **类型驱动的语义化 SVG 插画组件** + **增强 React Flow 兜底** | 全部走 React Flow / 全部走 SVG | 高频主题用定制插画拿"看一眼就懂"，长尾主题靠增强 ReactFlow 保证不开天窗 |
| 状态/CSS | **复用 antd 6 + globals.css 既有 token** | 引入 Tailwind / framer-motion | 控制依赖；现有 token 已足以承载学科色与动效 |
| 入场动效 | **CSS keyframes + Web Animations API** | framer-motion | 避免新增依赖，符合 reduced-motion 控制 |

---

## 5. 信息架构 / 渲染器选择策略

### 5.1 物理 sceneKind → R3F 场景
`components/physics/r3f/lib/sceneRegistry.ts` 按 `artifact.scene_type` + 现有 `sceneKind()` 路由：

| sceneKind | R3F 场景 | 关键视觉 |
|---|---|---|
| orbit | `OrbitScene` | 恒星 emissive + Bloom；行星 + `<Trail>`；轨道环 dashed line |
| projectile | `ProjectileScene` | 地形 plane PBR；抛体球；速度矢量箭头 + `<Html>` 标注；命中目标爆点 |
| spring | `SpringScene` | `TubeGeometry` 沿 helix 的螺旋弹簧；质点 + 振幅刻度 |
| collision | `CollisionScene` | 两刚体（金属 / 玻璃材质对比）+ 碰撞粒子爆点 |
| force | `ForceScene` | 斜面 + 物块 + 重力 / 支持力 / 摩擦力分解箭头（不同颜色） |
| motion (兜底) | `MotionScene` | 通用网格 + 质点 + 轨迹 |

### 5.2 生物 topic → SVG 插画
`components/biology/SemanticBiologyRenderer.tsx` 按 `spec.visualization_type` / `spec.topic` 路由：

| 主题关键词 | 插画组件 | 关键视觉 |
|---|---|---|
| 光合 / photosynthesis | `Photosynthesis` | 叶片 + 叶绿体剖面 + 光强箭头 + CO₂/O₂ 气泡；光强/CO₂/温度参数 → 实时速率曲线 |
| 呼吸 / respiration | `Respiration` | 线粒体剖面 + 糖酵解→TCA→电子传递链三段流程 + ATP 计数 |
| 酶 / enzyme | `Enzyme` | 底物 / 酶 / 产物 + 温度滑块 → 钟形活性曲线 |
| 突触 / synapse | `Synapse` | 突触前膜 → 神经递质囊泡 → 突触间隙 → 受体；动画演示单向传递 |
| 遗传 / genetics | `Genetics` | 染色体 + Punnett 方格 + 子代表现型比例柱状 |
| 生态 / ecosystem | `Ecosystem` | 食物网节点 + 能量流箭头（金字塔可选） |
| 膜 / membrane | `Membrane` | 磷脂双分子层 + 通道蛋白 + 浓度梯度箭头 |
| 未命中 | 增强 `GenerativeBiologyGraph` | 节点按类型选形状（胶囊 / 平行四边形 / 六边形 / 圆）+ 图标 + 渐变 marker 动画边 |

---

## 6. 文件级改动清单

### 6.1 新增
- `web/snowy-web/components/physics/r3f/R3FPhysicsPreview.tsx`
- `web/snowy-web/components/physics/r3f/scenes/{Orbit,Projectile,Spring,Collision,Force,Motion}Scene.tsx`
- `web/snowy-web/components/physics/r3f/lib/{sceneRegistry,cameraRig,postFX,hud}.tsx`
- `web/snowy-web/components/physics/r3f/hooks/useRapierSimulation.ts`
- `web/snowy-web/components/biology/illustrations/{Photosynthesis,Respiration,Enzyme,Synapse,Genetics,Ecosystem,Membrane}.tsx`
- `web/snowy-web/components/biology/SemanticBiologyRenderer.tsx`
- `web/snowy-web/components/common/{SubjectBadge,SkeletonPreview,SectionHeader}.tsx`

### 6.2 修改
- `web/snowy-web/package.json`：新增 `three` / `@react-three/fiber` / `@react-three/drei` / `@react-three/postprocessing`。
- `web/snowy-web/components/physics/NativePhysicsPreview.tsx`：保留为降级渲染器，从主链路解耦。
- `web/snowy-web/components/biology/BiologyDiagram.tsx`：节点形状 / 图标 / 边动效升级。
- `web/snowy-web/components/generative/GenerativeBiologyGraph.tsx`：先尝试 `SemanticBiologyRenderer`，未命中回退增强 React Flow。
- `web/snowy-web/components/generative/GenerativePhysicsCanvas.tsx`：学科 cyan 配色、`stroke-dashoffset` 入场、命中爆点动效。
- `web/snowy-web/app/globals.css`：新增 glass / 动画 / 移动端断点 token、`@keyframes`（浮入 / 脉冲 / glow / 命中爆点）。
- `web/snowy-web/app/page.tsx`：能力卡 hover 边光、推荐卡 stagger 入场 + 骨架屏。
- `web/snowy-web/app/modeling/page.tsx`：Stage 进度脉冲、参数滑块学科色 + 即时数字 chip、骨架屏、移动端单栏 + 抽屉参数。
- `web/snowy-web/app/physics/page.tsx` / `app/biology/page.tsx`：壳层引用新组件。

---

## 7. 性能与降级策略

- **代码分割**：R3F 场景按需 `next/dynamic`(`{ ssr: false }`) + `<Suspense>`，首屏不加载 Three.js；
- **目标 bundle 增量**：R3F 总 gz < ~150KB；
- **画质档位**：默认低质 Bloom；高画质（DoF + 高分辨阴影）通过 UI 开关启用；
- **WebGL 检测**：失败 → 回退 `NativePhysicsPreview`；
- **省电模式**：手动开关 + 自动检测（电池 API / `prefers-reduced-motion`）→ 关闭后处理 + 降低 simulation 步频；
- **SSR 兼容**：所有 R3F 与 Rapier 组件 `'use client'` + 动态导入，避免 Next.js 16 SSR 报错。

---

## 8. 验证策略

- `cd web/snowy-web && npm run lint` 保持零 warning；
- `npm run build` 通过；记录 bundle 大小增量；
- 手动验收用例：
  - 物理：抛体 / 弹簧 / 轨道 / 碰撞 / 斜面 / 通用运动 6 个 sceneKind 在 `/modeling` 与 `/physics` 各跑一遍，参数滑块实时驱动 3D；
  - 生物：光合 / 呼吸 / 酶 / 突触 / 遗传 / 生态 / 膜 7 主题命中专用插画并响应参数；未命中主题回退到增强 React Flow；
  - 双断点：桌面（1440px）+ 移动端（375px / 768px）视觉无错乱；
  - 强制禁用 WebGL（DevTools `--disable-webgl`）后自动回退至 `NativePhysicsPreview`；
  - `prefers-reduced-motion: reduce` 下无后处理、无入场动画、无脉冲；
- `make test-unit`（Go 后端）保证无回归（视觉改动不应影响后端，但兜底跑一次）。

---

## 9. 风险与权衡

| 风险 | 概率 | 缓解 |
|---|---|---|
| R3F + Rapier SSR 报错 | 中 | 全部 `dynamic({ ssr: false })`；CI 在 Node 环境跑 `next build` 验证 |
| Bundle 体积超预算 | 中 | `drei` 仅按需 import；`postprocessing` 只引入 Bloom；监控 `next build` 输出 |
| 移动端低端机性能差 | 中 | 默认低画质；提供"省电模式"；电池低于 20% 自动回退 |
| 插画美术质量不达预期 | 中 | 先做 2 个标杆（光合 + 突触）做内部评审，再展开剩余 5 个 |
| 旧 `NativePhysicsPreview` 被遗弃后回归 | 低 | 主链路通过 feature flag 切换；保留单元渲染入口便于灰度回滚 |

---

## 10. 里程碑（不含时间估计）

> 详细 todo 与依赖关系由 session 内 SQL 表跟踪，本节仅列高阶里程碑。

- **M1 · 基础设施**：依赖安装、`useRapierSimulation` 抽取、`R3FPhysicsPreview` 外壳与 WebGL 降级。
- **M2 · 物理 R3F 场景**：6 个 sceneKind 全部实现并接入 Bloom。
- **M3 · 生物插画**：7 个高频主题插画 + `SemanticBiologyRenderer` 路由 + 增强 ReactFlow 兜底。
- **M4 · UX 与页面**：globals.css 动效 / 学科色滑块 / 骨架屏 / 移动端抽屉 / 首页打磨。
- **M5 · 验证**：lint + build + 移动端 / reduced-motion / WebGL 降级人工验收 + `make test-unit`。

---

## 11. 不在本方案范围内

- 后端 prompt / Eino Graph / 资料召回链路改动；
- 新增建模主题（例如波动光学）；
- 国际化与暗色模式系统级切换（globals.css 已预留，但本次不实施）；
- 游戏化任务舱（见 `docs/snowy-v4-redesign-blueprint.md`，另行排期）。
