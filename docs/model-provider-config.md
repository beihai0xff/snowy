# Snowy 模型接入与本地私密配置

Snowy 只使用一个运行时配置文件：`configs/config.yaml`。该文件是本地私密文件，已被 `.gitignore` 忽略，不得提交到 Git。

## 初始化本地配置

```bash
cp configs/config.example.yaml configs/config.yaml
```

然后编辑 `configs/config.yaml` 中的 `llm.models[]`：

```yaml
llm:
  models:
    - provider: "openai"
      model_provider: "custom"
      model: "your-model-name"
      api_key: "replace-with-local-api-key"
      base_url: "https://api.openai.com/v1"
      timeout: 10m
      temperature: 0.2
      max_tokens: 131072
      max_retries: 2
      retry_interval: 1s
```

字段说明：

- `provider` 固定使用 `openai`，表示统一 OpenAI-compatible 调用实现。
- `model_provider` 仅用于网关透传和监控展示，可填写 `openai`、`xiaomi`、`custom` 等。
- `model` 是实际模型名。
- `base_url` 是 OpenAI-compatible API Base URL，服务会调用 `${base_url}/chat/completions`。
- `api_key` 是本地私密密钥，运行时作为 `Authorization: Bearer <api_key>` 发送。

## OpenAI / 小米 / 自定义网关示例

OpenAI:

```yaml
model_provider: "openai"
model: "gpt-4.1"
base_url: "https://api.openai.com/v1"
api_key: "sk-local-only"
```

小米或其他 OpenAI-compatible 网关：

```yaml
model_provider: "xiaomi"
model: "your-xiaomi-model"
base_url: "https://your-xiaomi-compatible-endpoint/v1"
api_key: "your-local-xiaomi-api-key"
```

自定义模型公司：

```yaml
model_provider: "custom"
model: "vendor-model-name"
base_url: "https://vendor.example.com/v1"
api_key: "vendor-local-api-key"
```

## Docker 与 binary 共用配置

Docker 和本地 binary 都读取同一个 `configs/config.yaml`：

```bash
go run ./cmd/snowy --config configs/config.yaml
make docker-run
```

为让同一个配置文件同时适配宿主机 binary 和 Docker container，数据库与 Redis 统一使用 `snowy-host.internal`：

```yaml
database:
  host: "snowy-host.internal"
redis:
  addr: "snowy-host.internal:6379"
```

Docker Compose 会自动把 `snowy-host.internal` 映射到宿主机网关。宿主机直接运行 binary 前，需要保证本机也能解析该域名：

```bash
sudo sh -c 'grep -q "snowy-host.internal" /etc/hosts || echo "127.0.0.1 snowy-host.internal" >> /etc/hosts'
```

## 安全规则

- 不要提交 `configs/config.yaml`。
- 不要把真实 `api_key`、JWT secret、OAuth secret、数据库密码写入仓库模板或文档。
- 用户提供的小米 `base_url` 和 `api_key` 只能写入本地 `configs/config.yaml`。
- 不要新增 `MIMO_API_KEY`、`XIAOMI_MIMO_API_KEY` 或厂商专用 SDK 分支；模型调用统一走 OpenAI-compatible `/chat/completions`。
