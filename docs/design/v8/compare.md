# Snowy v5 vs v6 · 对比与迁移指南

> 本文逐项列出 v5 现状与 v6 重设计的差异，作为后续实现阶段的迁移路线图。**本文不修改代码，仅作 review 与决策参考。**

---

## 1. 设计哲学对比

| 维度 | v5 | v6 |
|---|---|---|
| 心智锚点 | AI Science Mission Cockpit · 任务舱 | 学习工具 · 学生书桌上的台灯 |
| 视觉气质 | 深空 · 玻璃拟态 · 霓虹 · 扫描线 | 浅色 · 中性 · 克制 · 可读 |
| 目标用户感知 | 「炫的科技产品」 | 「随手用的学习工具」 |
| 文案风格 | 指挥舱 / 星图 / 任务 / 命中 / 校验灯 | 首页 / 提问 / 推演 / 我的学习 |
| 装饰元素 | 轨道环 / 能力雷达 / 学习链路图 / 网格 / 扫描线 | 全部删除 |
| 主导航 | 5 项（含 AI 监控） | 4 项（AI 监控移到 /admin） |
| 默认主题 | 强制深色 | 浅色（深色可切换） |

---

## 2. 视觉令牌对比

### 2.1 颜色

| 角色 | v5 | v6 |
|---|---|---|
| 页面背景 | `#020617` 深蓝黑 | `#FAFAF9` 暖灰白 |
| 卡片背景 | `rgba(9,18,34,0.72)` + backdrop-blur | `#FFFFFF` 纯白 |
| 主色 | `#22D3EE` 青 + `#818CF8` 紫 双主色 | `#2563EB` 学术蓝（单一主色） |
| 物理 | `#34D399` 翠绿 | `#0891B2` 青（更克制） |
| 生物 | `#FBBF24` 琥珀 | `#16A34A` 绿（符合直觉） |
| 错误 | `#FB7185` 粉红 | `#DC2626` 深红 |
| 主文本 | `#E5F2FF` 偏蓝白 | `#1C1917` 中性近黑 |
| 次文本 | `#91A8C3` 蓝灰 | `#57534E` 暖灰 |

### 2.2 字体

| 用途 | v5 | v6 |
|---|---|---|
| 中文标题 | `Baskerville, Songti SC` 衬线 | `PingFang SC, Noto Sans SC` 黑体 |
| 中文正文 | `Avenir Next, PingFang SC` | `PingFang SC, Noto Sans SC` |
| 英文 | `Avenir Next` | `Inter` |
| 公式 | KaTeX_Main | KaTeX_Main（保留） |
| 代码 | 无明确定义 | `JetBrains Mono` |

### 2.3 几何

| 维度 | v5 | v6 |
|---|---|---|
| 圆角 | 18 / 20 / 22 / 26 / 32（无章法） | 4 / 6 / 8 / 12 / 16 / 9999（标准阶梯） |
| 阴影 | `0 26~110px rgba(0,0,0,0.42)` 大阴影 | `0 1~12px rgba(28,25,23,0.04~0.10)` 克制 |
| 边框 | `rgba(125,211,252,0.16)` 半透明蓝 | `#E7E5E4` 实色暖灰 |
| 间距 | 杂用 8/14/16/18/22/28… | 严格 4 倍数：4/8/12/16/24/32/48/64 |
| backdrop-filter | 多处 blur(18~22px) | **完全禁用** |

---

## 3. 信息架构对比

### 3.1 主导航

| 序 | v5 路由 | v5 名称 | v6 路由 | v6 名称 |
|---|---|---|---|---|
| 1 | `/` | 指挥舱 | `/` | 首页 |
| 2 | `/search` | 知识星图 | `/ask` | 提问 |
| 3 | `/modeling` | 科学建模舱 | `/modeling` | 推演 |
| 4 | `/learning` | 任务档案 | `/learning` | 我的学习 |
| 5 | `/monitoring` | AI 监控 | `/admin/llm` | AI 调用监控（从主导航移除） |

### 3.2 路由迁移建议

```text
v5                    →   v6
/                     →   /
/search?q=...         →   /ask?q=...           （服务端 301）
/physics?q=...        →   /modeling?type=physics&q=...
/biology?q=...        →   /modeling?type=biology&q=...
/modeling             →   /modeling
/learning             →   /learning
/monitoring           →   /admin/llm            （需鉴权）
```

---

## 4. 页面级差异

### 4.1 首页

| 区块 | v5 | v6 |
|---|---|---|
| Hero kicker | "Snowy V5 Mission Cockpit" | 删除 |
| 主标题 | "AI 科学任务舱，把问题变成模型" | "问一个高中题，看见答案的来路" |
| 副标题 | 长段任务舱说明 | 一句话功能描述 |
| 副视觉 | 右侧轨道图（snowy-orbit + 4 个 node） | **删除整块** |
| 能力卡 | 3 张 + accent 圆形装饰 | 3 张 + 实色图标，无装饰 |
| 学习链路 | 7 步链路图（snowy-chain） | **删除** |
| 能力雷达 | 4 个 stat 卡 | **删除** |
| 今日任务卡 | 6 张 mission card | 改为 3 张推荐卡，文案降级 |
| 新增 | — | 「继续上次的学习」横向滚动列表 |

### 4.2 提问页（v5 `/search` → v6 `/ask`）

| 区块 | v5 | v6 |
|---|---|---|
| 页面标题 | "知识星图" | "提问" |
| 顶部 kicker | "Evidence Star Map" | 删除 |
| 搜索条 | 大卡片包裹 + 筛选器 inline | sticky 顶部窄条 + 筛选折叠 |
| 答案展示 | MarkdownText + 多 Card 嵌套 | 论文式正文 + 内联角标 [1][2] |
| 证据列表 | List 平铺 | 编号卡片，匹配内联角标 |
| 公式卡 | 一个并列 Card | 手风琴折叠（默认展开第 1 个） |
| 易错点 | 一个并列 Card | 折叠进手风琴 |
| 题型映射 | 一个并列 Card | 折叠进手风琴 |
| 参考信息视觉 | Tag + Text | `#FEF3C7` 黄色教辅高亮 + 角标 |
| 右辅栏 | 「下一步任务」+「学习建议队列」+「相关问题星轨」 | 「下一步」+「相关问题」+「提问技巧」 |

### 4.3 推演页（`/modeling`）

| 区块 | v5 | v6 |
|---|---|---|
| 顶部输入区 | Card-strong + 多行 + Tag + 学科 Segmented + 多按钮 | 单行工具栏：学科切换 + 输入 + 生成按钮 |
| 阶段进度 | 顶部 stage-strip（5 个 pill） | 保留但下移到工具栏下方一行 |
| 主体布局 | **三栏**：证据 + 画布 + AI 教练 | **两栏**：画布(70%) + AI 教练(30%) |
| 问题与证据 | 左栏独立 Card | 折叠条（默认收起），点击展开 |
| 模型画布 | 中栏占 ~50% 宽度 | 左栏占 ~70% 宽度，画布优先 |
| 反馈结果 | 嵌在 InteractionPlanPanel 内 | 浮动在画布底部，4 列关键数值 |
| AI 教练 | 右栏多个独立 Card 堆叠（5+） | 右栏 4 个紧凑 Card：推理摘要 / 参数 / 校验灯 / 微练习 |
| 校验报告 | 单独 ValidationReportPanel Card | 合并为「校验灯」4 格小卡 |

### 4.4 我的学习页（`/learning`）

| 区块 | v5 | v6 |
|---|---|---|
| 页面标题 | "学习中心" | "我的学习" |
| 用户卡 | Card 内含 Space + 多按钮 | 头像 + 邮箱 + 操作三列布局 |
| 统计卡 | 无 | 新增 4 张：本周提问 / 模型包 / 收藏 / 知识点 |
| Tab 数量 | 5（历史 / 答案 / 收藏 / 反馈 / 模型包） | 4（最近 / 收藏 / 答案 / 模型包），反馈合并 |
| 列表样式 | AntD List.Item | 标准表格 + 可点击行 |
| 未登录引导 | 表单 inline 在卡片内 | 单独 CTA 引导卡 + 一行表单 |

### 4.5 AI 监控页（v5 `/monitoring` → v6 `/admin/llm`）

| 区块 | v5 | v6 |
|---|---|---|
| 入口 | 主导航第 5 项 | **从主导航移除**，仅 `/admin/llm` 可达 |
| 调性 | 与全站脱节，背景 `#fafafa` | 与全站同色板 |
| KPI | 4 张 AntD Statistic | 4 张自定义大数字 + 迷你条形 |
| 模型配置 | Row + Col + 多 Descriptions | 2 列 Provider 卡，更清晰 |
| 聚合指标 | 单列 GroupMetrics Card | 双列：按链路 + 按厂商 |
| Tab | 无（一直滚动） | 3 Tab：调用记录 / Prompt Profile / 模型配置 |

---

## 5. 组件级差异

| 组件 | v5 | v6 |
|---|---|---|
| Button | AntD `<Button>` + 全局样式覆盖 | `.btn` + 修饰类（5 种类型 × 3 尺寸） |
| Card | `.snowy-glass` / `.snowy-glass-strong` 玻璃拟态 | `.card` 实色背景 + 1px 边框 |
| Tag | AntD `<Tag color="cyan/green/gold">` | `.tag` + 学科色 / 语义色修饰 |
| Search | AntD `<Search>` 大号 | `.search` 自定义组合 |
| Tabs | AntD `<Tabs>` | `.tabs` 下划线式 + 计数徽标 |
| List | AntD `<List>` | `.table` 表格化（学习页）+ `.list` 简单列表 |
| Alert | AntD `<Alert>` | `.alert` + 4 种语义色 |
| Empty | AntD `<Empty>` | `.empty` 自绘 |
| Progress | AntD `<Progress>` | `.metric-bar` 自定义 |
| 折叠 | AntD `<Collapse>` | 原生 `<details>` + `.accordion` |
| 滑杆 | 内嵌 `<input type="range">` 无样式 | `.param__slider` 统一样式 |

---

## 6. 文案 / 术语对照表

| v5 | v6 | 理由 |
|---|---|---|
| 指挥舱 | 首页 | 学生不需要"指挥"心智 |
| 知识星图 | 提问 | 直白行为词 > 隐喻 |
| 科学建模舱 | 推演 | 短 + 准确表达"调参看变化" |
| 任务档案 | 我的学习 | 学生熟悉 |
| AI 监控 | AI 调用监控 / 移到后台 | 学生不该看 |
| Mission Cockpit | （删除） | 不必要 |
| Evidence Star Map | （删除） | 不必要 |
| 任务卡 | 推荐 / 试试 | 不再强调游戏化 |
| 命中目标区 | 命中目标 | 删除"区"字 |
| 微练习队列 | 微练习 | 删除"队列" |
| 校验灯 | 校验 | 4 格小卡显示，名称简化 |
| 可信度 | 可信度（保留） | 学生能理解，且是关键指标 |
| 模型包 | 模型包（保留） | 已是产品概念 |
| 推理摘要 | AI 推理摘要 | 加 AI 前缀，更清晰 |
| 再推理 | 再推理（保留） | 简练且明确 |
| 学习链路 | （删除整块） | 没人会按 7 步走 |
| 能力雷达 | （删除） | 装饰 > 信息 |

---

## 7. 工程迁移路线图（非本稿范围，仅做衔接预案）

> 以下步骤**不在本设计稿阶段实施**，待 review 通过、用户决定进入实现阶段后再退出 plan mode 执行。

### Phase 0 · 视觉令牌迁移（0.5 天）

- [ ] 把 [`docs/design/v6/tokens.css`](tokens.css) 的变量定义合并到 [`web/snowy-web/app/globals.css`](../../../web/snowy-web/app/globals.css) 顶部
- [ ] 删除 v5 的 `--snowy-*` 变量与所有 `body::before / body::after` 装饰 CSS
- [ ] AntD `ConfigProvider` 主题（[`AppLayout.tsx`](../../../web/snowy-web/components/layout/AppLayout.tsx)）从 `darkAlgorithm` 改为默认；`colorPrimary` 改 `#2563EB`，`colorBgContainer` 改 `#FFFFFF`，`borderRadius: 12`

### Phase 1 · 顶栏 / 导航重写（0.5 天）

- [ ] [`AppLayout.tsx`](../../../web/snowy-web/components/layout/AppLayout.tsx) 菜单项重命名 + 移除 AI 监控
- [ ] 新增 `/admin` 鉴权 layout，把 `/monitoring` 移到 `/admin/llm`（保留 redirect）

### Phase 2 · 首页重写（1 天）

- [ ] 删除 [`app/page.tsx`](../../../web/snowy-web/app/page.tsx) 的 `snowy-orbit` / `snowy-chain` / `radarStats` / `capabilityCards` 装饰区块
- [ ] 新增「继续上次的学习」横向滚动（拉取 `api.getHistory()`）
- [ ] 文案对齐 v6

### Phase 3 · 提问页（1 天）

- [ ] [`app/search/page.tsx`](../../../web/snowy-web/app/search/page.tsx) → `app/ask/page.tsx`（保留 redirect）
- [ ] 答案区改为论文式排版，引用渲染为内联角标
- [ ] 公式卡 / 易错点 / 题型映射 改 `<Collapse>`
- [ ] 教辅黄色 `evidence-mark` 应用到关键短语

### Phase 4 · 推演页（1.5 天）

- [ ] [`app/modeling/page.tsx`](../../../web/snowy-web/app/modeling/page.tsx) 改两栏 + 顶部折叠条
- [ ] 提取 `EvidenceFoldBar` 组件
- [ ] 重写 `InteractionPlanPanel` + `ValidationReportPanel` 视觉
- [ ] 物理画布占 70%，参数刷新走本地重算

### Phase 5 · 学习中心 + 监控（1 天）

- [ ] [`app/learning/page.tsx`](../../../web/snowy-web/app/learning/page.tsx) Tab 数量 5→4，列表改 Table
- [ ] [`app/monitoring/page.tsx`](../../../web/snowy-web/app/monitoring/page.tsx) → `app/admin/llm/page.tsx`，视觉与全站对齐

### Phase 6 · 移动端优化 + 验证（0.5 天）

- [ ] 所有页面在 375 / 414 / 768 / 1280 / 1440 视口逐一回归
- [ ] 浅色 / 深色双主题验证

**合计：约 5~6 天**（不含测试和回归）

---

## 8. 风险与取舍

### 8.1 已知取舍

| 决策 | 牺牲 | 收益 |
|---|---|---|
| 砍掉所有炫技装饰 | 失去"科技感"卖点 | 阅读门槛降低；老师 / 家长接受度提升 |
| 默认浅色 | 失去"夜间学习"卖点（深色仍可切换） | 印刷感、可信感强；和教辅、试卷感觉一致 |
| AI 监控移出主导航 | 学生失去查看模型透明度的入口 | 学生认知负担降低；不属于学生关心的信息 |
| 删除「学习链路」7 步图 | 失去对产品理念的视觉宣传 | 节省一屏空间；学生看一次就够了 |
| 删除「能力雷达」 | — | 纯装饰，无信息价值 |

### 8.2 需要二轮 review 决策的点

1. **「继续上次的学习」的数据源**：v5 `api.getHistory()` 返回的是 raw history item，需要后端补全 `summary` 字段才能直接显示，或前端做截断。
2. **「教辅黄色高亮」如何在答案 Markdown 中标记**：建议后端在 `answer` 字段里用 `<mark>` 标签或自定义 markdown 扩展，前端用 CSS 渲染。需要 PE 调整。
3. **学科色「生物=绿」是否会与「成功=绿」混淆**：v6 我们用同一个色值是为了减少色板复杂度。如果发现混淆，备选方案是把"生物"换成 `#15803D` 深绿与 success 区分。
4. **「我的学习」未登录态默认显示哪个 view**：v6 原型默认显示未登录态（便于看引导卡）。实际实现应根据 token 自动判断。

---

## 9. 后续工作

- 等待用户对 v6 设计稿的反馈
- 收集具体页面 / 区块的修改意见
- 反馈点 #1~#4 需要明确决策
- 若所有反馈处理完且用户确认，则退出 plan mode 进入实现阶段
