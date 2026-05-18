# Snowy v8 建模体验升级执行报告

> 日期：2026-05-18
> 对应方案：[`docs/snowy-v8-modeling-experience-upgrade.md`](./snowy-v8-modeling-experience-upgrade.md)
> 状态：第一阶段已落地（前端视觉/教学/化学入口），等待 review 与进一步教研数据补充。

## 1. 已完成事项

### 1.1 物理 R3F 质感与可用性

| 项 | 状态 | 关键文件 |
|---|---:|---|
| 修复 3D canvas 被压扁（150px → `clamp(420px, 60vh, 640px)`） | ✅ | `web/snowy-web/components/physics/r3f/R3FPhysicsPreview.tsx` |
| 设备分层 `useDeviceTier`（low/mid/high） | ✅ | `web/snowy-web/lib/useDeviceTier.ts` |
| 程序化舞台灯光 + ContactShadows + SoftShadows（不依赖远程 HDR） | ✅ | `components/physics/r3f/lib/R3FStage.tsx` |
| Bloom/Vignette/DoF 后处理三档 | ✅ | `components/physics/r3f/lib/postFX.tsx` |
| 渐变尾迹 `R3FTrail` | ✅ | `components/physics/r3f/lib/R3FTrail.tsx` |
| HUD 实时 LaTeX 公式卡（KaTeX） | ✅ | `components/physics/r3f/lib/R3FHud.tsx` |
| 6 场景 quality 透传与升级 | ✅ | `components/physics/r3f/scenes/*` |

> 重要调整：原方案提到 `drei <Environment preset>`，实测会访问 `raw.githack.com` 下载 HDR，内网/离线会失败。因此实现改为本地程序化灯光，满足“不引入外部贴图资产”的稳定运行要求。

### 1.2 生物 2D SVG 增强

| 项 | 状态 | 关键文件 |
|---|---:|---|
| 光合作用 SVG 扩到 720×420 宽屏 | ✅ | `components/biology/illustrations/Photosynthesis.tsx` |
| glow/glass/grain SVG filter | ✅ | `Photosynthesis.tsx` |
| 光子粒子流 `<animateMotion>` | ✅ | `Photosynthesis.tsx` |
| 全局插画玻璃感背景、指标卡、动效 | ✅ | `components/biology/illustrations/illustrations.css` |
| ReactFlow 生物概念图节点/边视觉升级 + MiniMap | ✅ | `components/generative/GenerativeBiologyGraph.tsx` |

### 1.3 化学建模入口与球棍画布

| 项 | 状态 | 关键文件 |
|---|---:|---|
| `/modeling` 增加“化学” Tab | ✅ | `app/modeling/page.tsx` |
| `ModelingSubject` 扩展 `chemistry` | ✅ | `app/modeling/page.tsx`、`lib/api.ts` |
| 新增 `GenerativeChemistryCanvas` | ✅ | `components/chemistry/GenerativeChemistryCanvas.tsx` |
| 球棍模型 + 配平结果 + 电子流动画 | ✅ | `GenerativeChemistryCanvas.tsx` |
| 化学收藏类型支持 | ✅ | `lib/api.ts` |

### 1.4 人教版课纲映射 + 易错点库

| 项 | 状态 | 关键文件 |
|---|---:|---|
| PEP 目录与 schema | ✅ | `docs/curriculum/pep/README.md`、`schema.json` |
| 物理必修 1/2 课纲映射 | ✅ | `docs/curriculum/pep/physics/*.yaml` |
| 生物必修 1/2 课纲映射 | ✅ | `docs/curriculum/pep/biology/*.yaml` |
| 化学必修 1 课纲映射 | ✅ | `docs/curriculum/pep/chemistry/compulsory-1.yaml` |
| 物理/生物/化学易错点库 | ✅ | `docs/curriculum/pep/pitfalls/*.yaml` |
| YAML → JSON 构建脚本 | ✅ | `web/snowy-web/scripts/build-curriculum.cjs` |
| 前端查询层 + 中文 tag alias | ✅ | `web/snowy-web/lib/curriculum.ts` |
| `CurriculumBadge` + `PitfallList` | ✅ | `components/learning/*` |
| 演示卡接入课纲徽章与易错点 | ✅ | `app/modeling/page.tsx`、`InteractionPlanPanel.tsx` |

## 2. 验证结果

### 2.1 自动化检查

```bash
cd web/snowy-web
npm run curriculum:build
npm run lint -- --max-warnings=0
npx tsc --noEmit --pretty false
npm run build
```

结果：

| 命令 | 结果 |
|---|---:|
| `npm run curriculum:build` | ✅ 生成 physics/biology/chemistry JSON |
| `npm run lint -- --max-warnings=0` | ✅ 0 errors / 0 warnings |
| `npx tsc --noEmit` | ✅ 0 errors |
| `npm run build` | ✅ Next.js 生产构建通过 |

### 2.2 Playwright 回归

| 场景 | 结果 |
|---|---|
| 物理历史 package（平抛） | ✅ canvas 实测约 `710×459px`，不再是 150px；课纲徽章/易错点卡均出现 |
| 化学 mock package | ✅ `snowy-chemistry-canvas` 实测约 `790×469px`；球棍模型、课纲徽章、易错点卡均出现 |
| 化学 Tab | ✅ `/modeling` 空态可见“物理 / 生物 / 化学”三段选择 |

回归截图：

- [`output/playwright/baseline-v8/13-after-physics-v8.png`](../output/playwright/baseline-v8/13-after-physics-v8.png)
- [`output/playwright/baseline-v8/14-after-chemistry-v8.png`](../output/playwright/baseline-v8/14-after-chemistry-v8.png)

## 3. 当前未完成 / 后续建议

| 项 | 原因 | 建议 |
|---|---|---|
| 化学后端完整 `ChemistryReactionPackage` | 当前前端先做题干识别与本地 fallback 画布 | 下一阶段与 v7 §5 后端契约对齐 atoms/bonds/charges/electron_flow 字段 |
| 生物 7 张插画全部专项重绘 | 当前完成全局 CSS + 光合作用重点升级 | 后续逐个改 `Respiration/Enzyme/Synapse/Genetics/Ecosystem/Membrane` |
| 物理每个场景的高级粒子/软体形变 | 已做 canvas、高光、尾迹、HUD、场景基础质感 | 后续继续补 `Impact`、`AfterImage` 等专项组件 |
| 课纲覆盖完整度 | 已覆盖高频章节，不是全量教材目录 | 后续由教研持续补齐 yaml |

## 4. Review 重点

1. 是否认可移除远程 HDR、改本地程序化灯光的取舍？
2. `CurriculumBadge` / `PitfallList` 是否放在当前交互位置合适？
3. 化学 fallback 画布是否作为 v8 首版可接受？
4. 是否继续按“先补全生物 7 插画 → 再做化学后端契约 → 再做物理场景粒子细化”的顺序推进？

