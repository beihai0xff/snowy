# Snowy Web 前端

面向高中生的 Snowy v5 AI 科学任务舱 — 知识点直答、物理仿真与生物可视化统一建模。

## 技术栈

- **框架**: Next.js 16 + TypeScript
- **UI**: Ant Design 6 + @ant-design/icons
- **状态管理**: Zustand
- **浏览器渲染**: iframe Sandbox + 受控 React / Canvas / SVG 运行时
- **流程图**: React Flow (@xyflow/react)
- **部署**: 静态导出 + Nginx Docker 容器

## 页面结构

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | 指挥舱 | 搜索即建模、能力雷达、学习链路、任务卡 |
| `/search` | 知识星图 | 知识点直答、答案摘要、参考信息、公式卡、易错点、建模跳转 |
| `/modeling` | 科学建模舱 | 参考信息 / 模型画布 / AI 教练三栏工作台 |
| `/physics` | 物理建模 | 题目解析、推导步骤、浏览器渲染、参数调节 |
| `/biology` | 生物建模 | 概念识别、过程拆解、结构图/流程图 |
| `/learning` | 任务档案 | 历史记录、收藏内容、下一步入口 |

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

# 或先从项目根目录启动后端依赖（会等待 MySQL healthy 并自动迁移）
make docker-up
```

前端容器通过 Nginx 反向代理 `/api/` 到 `snowy:8080`，访问地址 `http://localhost:3001`。

## API 对接

API 客户端位于 `lib/api.ts`，统一处理：
- Token 注入 (Bearer JWT)
- 响应解包 (`{code, message, data, request_id}`)
- 错误映射

环境变量 `NEXT_PUBLIC_API_BASE` 可覆盖 API 基础路径（默认 `/api/v1`）。
