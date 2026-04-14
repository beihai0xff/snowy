// Package repo 是基础设施层（Infrastructure Layer），统一封装所有外部依赖的适配实现。
//
// 子包划分：
//   - mysql       — MySQL 连接池 + Repository 实现（实现 user.Repository、agent.*Repository 等 DDD Port）
//   - redis       — Redis 连接 + 缓存（Cache）、限流（RateLimiter）、会话存储（SessionStore）
//   - llm         — LLM 供应商统一接口 + 适配实现（OpenAI、Gemini）
//   - embedding   — Embedding 供应商统一接口 + 适配实现（OpenAI text-embedding）
//   - opensearch  — OpenSearch 搜索引擎适配（实现 search.Repository、content/indexer.Indexer）
//   - storage     — 对象存储统一接口 + 适配实现（MinIO / S3）
//
// 上层业务（domain service）通过 interface 依赖本层能力，不直接耦合具体实现。
package repo
