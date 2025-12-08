# ============================================================================
# 多阶段构建 Dockerfile for IAM Contracts
# ============================================================================

# ============================================================================
# 构建阶段
# ============================================================================
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git make

# 设置 Go 环境变量
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 复制 go mod 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN go build \
    -ldflags "-s -w \
        -X main.Version=${VERSION} \
        -X main.BuildTime=${BUILD_TIME} \
        -X main.GitCommit=${GIT_COMMIT}" \
    -o apiserver \
    ./cmd/apiserver/

# ============================================================================
# 运行阶段
# ============================================================================
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/apiserver /app/apiserver

# 复制配置文件
COPY --chown=app:app configs /app/configs

# 创建应用运行所需目录（日志、持久化密钥等）
RUN mkdir -p /app/logs /app/data/keys /var/log/iam-contracts && \
    chown -R app:app /app /var/log/iam-contracts

# 预创建 TLS 目录，避免按文件挂载失败
RUN mkdir -p /etc/iam-contracts/ssl /etc/iam-contracts/grpc/ca /etc/iam-contracts/grpc/server && \
    chmod 755 /etc/iam-contracts /etc/iam-contracts/ssl /etc/iam-contracts/grpc /etc/iam-contracts/grpc/ca /etc/iam-contracts/grpc/server

# 切换到非 root 用户
USER app

# 暴露端口
EXPOSE 9080 9444

# 健康检查 (检查实际监听的端口 9080)
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:9080/healthz || exit 1

# 启动命令
ENTRYPOINT ["/app/apiserver"]
CMD ["--config=/app/configs/apiserver.prod.yaml"]
