# Snowy Web 前端

面向高中生的 AIGC 学习平台 — 知识检索、物理建模、生物建模。

## 技术栈

- **框架**: Next.js 16 + TypeScript
- **UI**: Ant Design 6 + @ant-design/icons
- **状态管理**: Zustand
- **图表**: ECharts (echarts-for-react)
- **流程图**: React Flow (@xyflow/react)
- **部署**: 静态导出 + Nginx Docker 容器

## 页面结构

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | 首页 | 搜索入口、推荐卡片、快捷入口 |
| `/search` | 知识检索 | 搜索、筛选、结果、引用、关联问题 |
| `/physics` | 物理建模 | 题目解析、推导步骤、2D 图表、参数调节 |
| `/biology` | 生物建模 | 概念识别、过程拆解、结构图/流程图 |
| `/learning` | 学习中心 | 用户信息、历史记录、收藏内容 |

## 本地开发

```bash
# 安装依赖
npm install

# 启动开发服务器 (localhost:3000)
npm run dev

# 或从项目根目录
make web-dev
```

## 构建

```bash
# 静态导出到 out/
npm run build

# 或从项目根目录
make web-build
```

## Docker 部署

```bash
# 从项目根目录构建前端镜像
make docker-build-web

# 或使用 docker compose 启动全部服务
make docker-up
```

前端容器通过 Nginx 反向代理 `/api/` 到 `snowy-api:8080`，访问地址 `http://localhost:3001`。

## API 对接

API 客户端位于 `lib/api.ts`，统一处理：
- Token 注入 (Bearer JWT)
- 响应解包 (`{code, message, data, request_id}`)
- 错误映射

环境变量 `NEXT_PUBLIC_API_BASE` 可覆盖 API 基础路径（默认 `/api/v1`）。

