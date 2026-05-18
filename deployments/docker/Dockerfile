# ============================================================
# Snowy — Multi-stage Dockerfile
# 支持构建 snowy 目标二进制
#
# 构建参数:
#   TARGET   — 构建目标 (snowy)，默认 snowy
#   GOPROXY  — Go module proxy，默认使用国内代理并回落 direct
#
# 用法:
#   docker build --build-arg TARGET=snowy  -t snowy        .
# ============================================================

# ── Stage 1: Build ──────────────────────────────────────────
ARG GO_BUILDER_IMAGE=docker.m.daocloud.io/library/golang:1.25-alpine
ARG ALPINE_RUNTIME_IMAGE=docker.m.daocloud.io/library/alpine:3.21

FROM ${GO_BUILDER_IMAGE} AS builder

ARG TARGET=snowy
ARG GOPROXY=https://goproxy.cn,direct

ENV GOPROXY=${GOPROXY}

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
FROM ${ALPINE_RUNTIME_IMAGE}

ARG TARGET=snowy

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

# 复制配置模板；运行时通过 volume 挂载本地 configs/config.yaml
COPY configs/ ./configs/

# 默认 API 端口 8080
EXPOSE 8080

ENTRYPOINT ["./snowy-server"]
