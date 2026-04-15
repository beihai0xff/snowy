// Package pkg 聚合项目内部公用基础包，供各业务域和基础设施层共享。
//
// 子包划分：
//   - common      — 统一错误码、日志工具、响应封装、上下文辅助
//   - config      — 配置加载与结构定义（Viper）
//   - middleware   — HTTP 中间件（鉴权、限流、CORS、日志、Recovery、RequestID）
package pkg
