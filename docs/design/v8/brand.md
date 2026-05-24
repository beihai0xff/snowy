# Snowy 品牌使用指南

> 本文档定义 Snowy 品牌识别系统：mark（标记）、wordmark（横版标）、配色、尺寸、间距、误用边界。所有 UI、文档、社交分享卡片必须遵循本文档。任何破例使用需先与设计 owner 评审。

---

## 1. 品牌资产

### 1.1 文件清单

| 资产 | 路径 | 用途 |
|---|---|---|
| 彩色 Mark | [`web/snowy-web/public/brand/logo-mark.svg`](../../../web/snowy-web/public/brand/logo-mark.svg) | UI 默认 |
| 单色 Mark | [`web/snowy-web/public/brand/logo-mark-mono.svg`](../../../web/snowy-web/public/brand/logo-mark-mono.svg) | 跟随 `currentColor`，用于 hover、反白、暗主题 |
| 彩色 Wordmark | [`web/snowy-web/public/brand/logo-wordmark.svg`](../../../web/snowy-web/public/brand/logo-wordmark.svg) | 头图、营销页 |
| 单色 Wordmark | [`web/snowy-web/public/brand/logo-wordmark-mono.svg`](../../../web/snowy-web/public/brand/logo-wordmark-mono.svg) | 暗主题、单色印刷 |
| 矢量 favicon | [`web/snowy-web/app/icon.svg`](../../../web/snowy-web/app/icon.svg) | 现代浏览器 |
| PNG favicon 32 | [`web/snowy-web/app/icon.png`](../../../web/snowy-web/app/icon.png) | 老浏览器 fallback |
| Apple Touch | [`web/snowy-web/app/apple-icon.png`](../../../web/snowy-web/app/apple-icon.png) | iOS 180×180 |
| OG Image | [`web/snowy-web/public/brand/og-image.png`](../../../web/snowy-web/public/brand/og-image.png) | 社交分享 1200×630 |
| 文档共享副本 | [`docs/assets/brand/`](../../assets/brand/) | README、设计文档 |

### 1.2 React 组件

UI 中**不直接引用 SVG 文件**，统一用 [`<BrandMark />`](../../../web/snowy-web/components/common/BrandMark.tsx) 组件：

```tsx
import BrandMark from '@/components/common/BrandMark';

<BrandMark size={28} />
```

组件内部使用 `currentColor`，外层 CSS 控制颜色：

```css
.snowy-brand-mark { color: var(--color-primary); }
.snowy-brand:hover .snowy-brand-mark { color: var(--color-primary-hover); }
```

---

## 2. 视觉系统

### 2.1 几何

- viewBox **24×24**，6 轴对称（与雪花六重对称同构）
- **中心节点** r=2.5，固定 `(12, 12)`
- **中间节点** r=1.4，半径 6，6 个，沿 `[0°, 60°, 120°, 180°, 240°, 300°]`
- **端节点** r=0.9，半径 10
- **线段** stroke=1.3，round cap，共 6 条，从中心穿过中间节点到端节点
- 元素叠绘顺序：线 → 中间节点 → 端节点 → 中心节点（视觉重心保留在中心）

### 2.2 隐喻

节点-雪花混合：既保留 Snowy 的"雪花"原始联想（六轴对称），又嵌入"知识图谱节点"的现代语义（节点 + 连接）。**禁止理解为水分子结构、机械齿轮、装饰花纹**。

### 2.3 颜色

| 用法 | 颜色 | Token |
|---|---|---|
| 主用彩色 | `#2563EB` | `--color-primary` |
| Hover | `#1D4ED8` | `--color-primary-hover` |
| 反白 | `#FFFFFF` | — |
| 单色（继承） | `currentColor` | — |
| Wordmark 文字 | `#1C1917` | `--color-text` |
| 错误状态禁用色 | `#FCA5A5` | `--color-danger-soft`（仅警告占位） |

**禁止**使用未在此表中的颜色填充 mark，包括但不限于灰阶（`#666` 等）、品牌相邻色调（紫、青）、渐变色。

---

## 3. 尺寸与间距

### 3.1 最小尺寸

| 形态 | 最小展示尺寸 |
|---|---|
| Mark（仅 mark） | **16px** |
| Wordmark（横版） | **80px 宽** |

低于此阈值时几何细节（中间节点）会塌陷成视觉噪声，必须升级为 fallback：用 `<BrandMark size={16}>` 并配合 `--color-primary` 实底背景，或干脆不展示。

### 3.2 安全留白

mark 周围 ≥ **0.5 倍 mark 高度** 的留白（即 mark 28×28 时，四周不允许出现其他文本、按钮、图标，距离 ≥ 14px）。

```
┌─────────────────────────┐
│                         │
│       ░░░░░░░░░         │  ← 安全留白 0.5×mark
│       ░░ MARK ░░         │
│       ░░░░░░░░░         │
│                         │
└─────────────────────────┘
```

### 3.3 Wordmark 锁定

mark（24×24）+ **12px 间距** + "Snowy" 文字（Inter SemiBold 20px，letter-spacing -0.4，颜色 `--color-text`）。**禁止**调整间距、改变字号比例、替换字体。

---

## 4. 用法

### 4.1 应用 UI

- 顶部 header：`<BrandMark size={28} />` + `<span>Snowy</span>`，使用 `--color-primary`
- favicon：通过 Next.js 文件约定自动注入 `app/icon.svg` / `app/apple-icon.png`
- loading 占位：可短暂展示 `<BrandMark size={48} />` 作为 spinner 容器，**不允许**添加旋转动画到 mark 本身（mark 是品牌符号，不是 loader 资产）

### 4.2 文档

- README：使用 `docs/assets/brand/logo-wordmark.svg`，`<img height="48">` 嵌入
- 内部设计文档：mark 用 inline SVG，wordmark 用文件引用

### 4.3 社交卡片

- 所有外链分享必须落到 `og-image.png`（1200×630），由 `app/layout.tsx` 的 `metadata.openGraph` 与 `metadata.twitter` 注入
- **禁止**为社交平台单独 PS 不同版本

---

## 5. 错误用法

以下用法**全部禁止**，发现立即整改：

| 错误 | 说明 |
|---|---|
| 拉伸 / 压缩 | mark 必须 1:1 等比缩放，禁止改变宽高比 |
| 旋转 | mark 不可旋转（破坏 6 轴对称的轴线方向） |
| 改色 | 禁止使用 §2.3 之外的填充色 |
| 降透明度 | 禁止用 `opacity < 1` 作为"次要"层级，请改用单色 mark 配合低对比度容器 |
| 加阴影 / 发光 / 描边 | mark 自带视觉重量，禁止 `box-shadow`、`filter: drop-shadow`、外描边 |
| 替换内部几何 | 不允许把节点改成方块、星形、其他图标 |
| 嵌入小图标 | mark 内部禁止叠加 ✓ / ★ / 文字等装饰 |
| 用 emoji 替代 | ❄ / ❄️ 已弃用，所有占位必须替换为 inline SVG 或 `<BrandMark />` |
| 与其他 logo 拼接 | 禁止把 mark 和第三方 logo 直接相邻（如"Snowy × XX"），需用至少 2 倍 mark 宽度的分隔符 |
| 单色版改色 | 单色 mark 的 `currentColor` 必须继承品牌色系，不允许染成警告红、成功绿等语义色 |

---

## 6. 版本与归档

| 版本 | 日期 | 变更 |
|---|---|---|
| v1.0 | 2026-05-21 | 首版：6 轴节点-雪花混合 mark + wordmark + 单色版 + OG 卡片 |

后续如需调整几何，请在 [`logo-mark.svg`](../../../web/snowy-web/public/brand/logo-mark.svg) 与 [`BrandMark.tsx`](../../../web/snowy-web/components/common/BrandMark.tsx) 中同步改动，并更新本表。
