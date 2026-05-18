# 人教版课纲映射数据

本目录存放人教版（PEP）高中物理 / 生物 / 化学课纲条目映射表与易错点库。

## 合规与版权

| 项 | 策略 |
|---|---|
| **课本原文** | ❌ 不复制 |
| **章节编号 + 章节标题** | ✅ 录入（公开信息） |
| **学习目标** | ✅ 录入自有教研文案 |
| **易错点 / 纠错提示** | ✅ 录入自有文案 |
| **典型例题** | ❌ 不直接复制；可改写概括知识点 |

## 目录结构

```
pep/
├── physics/{compulsory-1,compulsory-2,elective-3-x}.yaml
├── biology/{compulsory-1,compulsory-2,elective-x}.yaml
├── chemistry/{compulsory-1,compulsory-2,elective-x}.yaml
└── pitfalls/{physics,biology,chemistry}.yaml
```

## Schema

见 [schema.json](./schema.json)；前端通过 `web/snowy-web/lib/curriculum.ts` 加载（dynamic import per subject）。

## 维护流程

1. PR 修改 yaml；
2. CI 用 schema.json 校验；
3. 前端启动期解析为 JSON，运行时按 `knowledge_tags` O(1) 反查。

## 版本说明

- `version: pep-2019` — 2019 版人教版课纲（普通高中）。
- 后续若有新版课纲，新增 yaml 文件并保留旧版本号兼容。

