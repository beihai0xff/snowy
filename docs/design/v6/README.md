# Snowy v6 UI 重设计稿

> 这是 Snowy 前端 UI 的**第 6 版**重设计草案。当前阶段只交付**可交互的 HTML/CSS 静态原型**与配套**设计规范**，作为后续 Next.js + AntD 实现的视觉锚点。本目录不修改任何现有前端代码。

---

## 0. 如何看原型

```bash
# 直接用浏览器打开
open docs/design/v6/prototype/index.html
# 或在 IDE 里用 Live Preview 插件打开
```

入口页 `prototype/index.html` 提供导航。每个页面右下角有「📱 移动预览」按钮，一键切到 414 宽视图测响应式；顶栏右侧「☾ 深色」按钮切换深色主题，验证对比度。

---

## 1. 设计哲学（决定一切）

**一句话定调**：Snowy 不是科幻舰桥，是**学生书桌上一盏开着的台灯**——克制、明亮、随手能用，把光打在内容上而不是装饰上。

四条不可妥协的原则：

| # | 原则 | 落地方式 |
|---|---|---|
| 1 | **内容前置，装饰退后** | 去掉扫描线、轨道环、衬线英文 kicker、霓虹网格。首屏只剩搜索框、最近学习、三大能力入口。 |
| 2 | **可读性大于科技感** | 正文 16px、行高 1.7；浅色为默认 + 深色可切换；关闭所有 backdrop-filter blur。 |
| 3 | **认知负担最小化** | 术语回归人话：「指挥舱→首页」「知识星图→提问」「科学建模舱→推演」「任务档案→我的学习」；「AI 监控」从主导航**移除**。 |
| 4 | **手机也好用** | 高中生大量用手机。桌面 / 移动同等优先级，所有原型双视图交付。 |

---

## 2. 新信息架构

主导航从 v5 的 5 项压缩到 4 项：

```
首页 · 提问 · 推演 · 我的学习
```

- `/monitoring` 重命名 `/admin/llm`，从主导航移除，仅运营 / 教师可见
- `/physics` `/biology` 历史路由合并到 `/modeling`（实现阶段保留 redirect）

```mermaid
graph LR
  Home[首页] --> Ask[提问 /ask]
  Home --> Lab[推演 /modeling]
  Home --> My[我的学习 /learning]
  Ask -->|生成模型| Lab
  Lab -->|收藏| My
  Ask -->|收藏| My
  Admin[/admin/llm] -.独立入口.-> Monitor[AI 调用监控]
```

---

## 3. 与 v5 的核心差异

| 维度 | v5 现状 | v6 新设计 |
|---|---|---|
| 调性 | 深空科幻 / 任务舱 / 玻璃拟态 | 学习工具 / 浅色为主 / 中性书面感 |
| 主色 | 青 `#22d3ee` + 紫 `#818cf8` | 学术蓝 `#2563EB` |
| 中文字体 | Baskerville / Songti 衬线 | PingFang SC / Noto Sans SC 黑体 |
| 装饰 | 扫描线、轨道环、雷达图、网格 | 一律删除 |
| 圆角 | 18~32px | 4 / 8 / 12 / 16 收敛 |
| 阴影 | 26~110px 大阴影 | 1~12px 单层 |
| 主导航 | 5 项（含 AI 监控） | 4 项（AI 监控移到 /admin） |
| 术语 | 指挥舱 / 任务舱 / 星图 / 任务卡 | 首页 / 提问 / 推演 / 我的学习 |
| 建模页 | 三栏，画布只占 50% | 两栏 + 顶部折叠条，画布占 70% |
| 监控页 | 与全站脱节的 `#fafafa` 浅灰 | 同色板浅色，与全站统一 |
| 移动端 | 1180px 以下退化 | 双视图同等设计 |

详见 [`compare.md`](./compare.md)。

---

## 4. 设计令牌（速览）

完整定义见 [`tokens.css`](./tokens.css)。

### 色板

| 角色 | 浅色 | 深色 |
|---|---|---|
| 页面背景 | `#FAFAF9` | `#0C0A09` |
| 卡片背景 | `#FFFFFF` | `#1C1917` |
| 边框 | `#E7E5E4` | `#292524` |
| 主文本 | `#1C1917` | `#FAFAF9` |
| 次文本 | `#57534E` | `#D6D3D1` |
| 主色 | `#2563EB` | `#60A5FA` |
| 物理 | `#0891B2` 青 |
| 生物 | `#16A34A` 绿 |
| 化学 | `#D97706` 琥珀 |
| 数学 | `#9333EA` 紫 |
| 证据高亮 | `#FEF3C7` 背景 + `#92400E` 文字 |

### 字体

```css
--font-sans:    "PingFang SC", "Noto Sans SC", "Microsoft YaHei", Inter, "Segoe UI", sans-serif;
--font-display: Inter, "PingFang SC", "Noto Sans SC", sans-serif;
--font-mono:    "JetBrains Mono", Menlo, Consolas, monospace;
```

公式继续用 KaTeX，无变化。

### 字号阶梯

| Token | 值 | 用法 |
|---|---|---|
| `--text-xs` | 12px | Caption / 时间戳 |
| `--text-sm` | 14px | 辅助文本 / 按钮 |
| `--text-base` | 16px | 正文 |
| `--text-lg` | 18px | 副标题 |
| `--text-xl` | 20px | H3 |
| `--text-2xl` | 24px | H2 |
| `--text-3xl` | 32px | H1 |
| `--text-4xl` | 40px | Hero（移动 32） |
| `--text-5xl` | 56px | Hero 桌面 |

### 圆角

`4 / 6 / 8 / 12 / 16 / 9999` —— 卡片 12，按钮 8，标签 4，圆形 9999。

### 阴影

```css
--shadow-xs: 0 1px 2px rgba(28, 25, 23, 0.04);
--shadow-sm: 0 1px 3px rgba(28, 25, 23, 0.06), 0 1px 2px rgba(28, 25, 23, 0.04);
--shadow-md: 0 4px 12px rgba(28, 25, 23, 0.08), 0 2px 4px rgba(28, 25, 23, 0.04);
--shadow-lg: 0 12px 32px rgba(28, 25, 23, 0.10), 0 4px 8px rgba(28, 25, 23, 0.06);
```

### 间距栅格

4 的倍数：`4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64 / 80`。桌面外边距 32，移动 16。

---

## 5. 页面索引

| 路由 | 原型 | 重点变化 |
|---|---|---|
| `/` | [`home.html`](./prototype/home.html) | 砍掉轨道图 / 能力雷达 / 学习链路 / Mission Cockpit kicker；首屏 = 搜索框 + 三能力卡 + 最近学习 + 今日推荐 |
| `/ask`（原 `/search`） | [`ask.html`](./prototype/ask.html) | 论文式答案排版；引用脚注 [1][2] 内联；公式卡 / 易错点用手风琴折叠 |
| `/modeling` | [`modeling.html`](./prototype/modeling.html) | 画布占 70%；证据从左栏移到顶部折叠条；AI 教练右栏；物理 / 生物两态 |
| `/learning` | [`learning.html`](./prototype/learning.html) | 顶部统计卡 + 4 Tab（最近·收藏·答案·模型包）+ 表格化列表；含未登录态 |
| `/admin/llm` | [`admin-llm.html`](./prototype/admin-llm.html) | 从主导航移除；与全站同色板；4 统计卡 + 3 Tab |
| 组件参考 | [`components.html`](./prototype/components.html) | 按钮 / 输入 / 标签 / 卡片 / 空状态 / 加载态全集 |

---

## 6. 工程做法（HTML 原型）

- **纯 HTML / CSS / 极少 vanilla JS**，不引入框架
- 所有页面共用 [`tokens.css`](./tokens.css) + [`components.css`](./components.css) + [`prototype/shell.js`](./prototype/shell.js)
- 真实文本（PRD 案例：平抛运动 / 光合作用 / 牛顿第二定律），不写 lorem ipsum
- 数据用静态 mock，每个 HTML 末尾的 `<!-- -->` 注释贴对应 API 示例 JSON
- 桌面基线 1440 宽，移动基线 375 宽
- 主题切换 / 移动预览 / 导航高亮通过 [`prototype/shell.js`](./prototype/shell.js) 实现，localStorage 持久化

---

## 7. 不在本稿范围（明确划线）

| 项 | 说明 |
|---|---|
| ❌ React / Next.js 代码 | 本稿只交付 HTML 原型与规范 |
| ❌ 后端 API 改动 | 完全沿用 [`web/snowy-web/lib/api.ts`](../../../web/snowy-web/lib/api.ts) 现有契约 |
| ❌ 复杂动效 | 仅基础 hover / transition；进入 / 退出动画在实现阶段定 |
| ❌ i18n / 多语言 | 先专注中文体验 |
| ❌ 完整 a11y 审计 | 色对比度遵守 WCAG AA，但不做完整无障碍认证 |

---

## 8. Review → 实现的衔接

设计稿 review 通过后，**实现阶段**可以这样落地（不在本稿范围，仅做衔接预案）：

1. 把 [`tokens.css`](./tokens.css) 的 CSS 变量迁移到 `web/snowy-web/app/globals.css`，替换现有 `--snowy-*` 变量
2. AntD 主题适配：在 [`AppLayout.tsx`](../../../web/snowy-web/components/layout/AppLayout.tsx) 的 `ConfigProvider` 把 `algorithm` 从 `darkAlgorithm` 改为默认（亮色），`token` 颜色对齐新色板
3. 重写 5 个页面（按本稿原型一一对应）
4. 路由迁移：`/search` → `/ask`（保留 redirect），`/monitoring` → `/admin/llm`
5. 删除当前 `app/globals.css` 的扫描线 / 轨道 / 玻璃拟态相关 CSS（约 700 行可瘦身到 200 行内）

---

## 9. 变更记录

| 日期 | 版本 | 变更 |
|---|---|---|
| 2026-05-15 | v6 draft 1 | 初版交付：tokens + 5 页原型 + 组件清单 + compare 文档 |
