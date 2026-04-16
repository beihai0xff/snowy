# ============================================================
# Snowy — Multi-stage Dockerfile
# 支持构建 api / worker 两个目标二进制
#
# 构建参数:
#   TARGET  — 构建目标 (api | worker)，默认 api
#
# 用法:
#   docker build --build-arg TARGET=api   -t snowy-api    .
#   docker build --build-arg TARGET=worker -t snowy-worker .
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG TARGET=api

# 安装编译依赖
RUN apk add --no-cache gcc musl-dev git ca-certificates tzdata

WORKDIR /src

# 先复制 go.mod / go.sum，利用 Docker layer cache
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码
COPY . .

# 编译目标二进制
# CGO_ENABLED=0 静态链接, 可在 scratch/distroless 中运行
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
              -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/snowy-${TARGET} \
    ./cmd/${TARGET}

# ── Stage 2: Runtime ────────────────────────────────────────
FROM alpine:3.21

ARG TARGET=api

# 时区 + CA 证书 (调用外部 LLM API 需要)
RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 非 root 运行
RUN addgroup -S snowy && adduser -S snowy -G snowy
USER snowy

WORKDIR /app

# 复制二进制
COPY --from=builder /out/snowy-${TARGET} ./snowy-server

# 复制默认配置（运行时可通过 volume 或 env 覆盖）
COPY configs/ ./configs/

# API 默认端口 8080, Worker 默认端口 8081
EXPOSE 8080 8081

ENTRYPOINT ["./snowy-server"]

