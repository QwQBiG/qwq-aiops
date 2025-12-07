# ============================================
# qwq AIOps Platform - Multi-stage Dockerfile
# Version: 1.0.0
# ============================================

# --- Stage 1: Build Frontend (Vue 3) ---
FROM node:18-alpine AS frontend-builder

LABEL stage=frontend-builder
LABEL maintainer="qwq AIOps Team"

WORKDIR /app/frontend

# 复制前端依赖配置和锁文件
COPY frontend/package*.json ./

# 安装依赖（使用 npm ci 更快更可靠）
RUN npm ci

# 复制前端源码
COPY frontend/ .

# 构建生产版本
RUN npm run build

# 清理不需要的文件
RUN rm -rf node_modules src public

# --- Stage 2: Build Backend (Go) ---
FROM golang:1.23-alpine AS backend-builder

LABEL stage=backend-builder
LABEL maintainer="qwq AIOps Team"

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates

# 复制 Go 模块文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download && go mod verify

# 复制源码
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# 从前端构建阶段复制编译好的文件
COPY --from=frontend-builder /app/frontend/dist ./internal/server/dist

# 编译 Go 程序（优化编译参数）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o qwq \
    ./cmd/qwq/main.go

# 验证二进制文件
RUN chmod +x qwq && ./qwq --version || true

# --- Stage 3: Final Runtime Image ---
FROM alpine:3.19

LABEL maintainer="qwq AIOps Team" \
      version="1.0.0" \
      description="qwq AIOps - AI-Powered Intelligent Operations Platform" \
      org.opencontainers.image.source="https://github.com/QwQBiG/qwq-aiops" \
      org.opencontainers.image.documentation="https://github.com/QwQBiG/qwq-aiops/blob/main/README.md" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# 安装运行时依赖和运维工具
RUN apk add --no-cache \
    # 基础工具
    bash \
    curl \
    wget \
    ca-certificates \
    tzdata \
    tini \
    # 系统监控工具
    procps \
    coreutils \
    grep \
    net-tools \
    iproute2 \
    # Docker 客户端
    docker-cli \
    && rm -rf /var/cache/apk/*

# 安装 kubectl
ARG INSTALL_KUBECTL=true
RUN if [ "$INSTALL_KUBECTL" = "true" ]; then \
    KUBECTL_VERSION=$(wget -qO- https://dl.k8s.io/release/stable.txt) && \
    wget -q "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl; \
    fi

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建非 root 用户（安全最佳实践）
RUN addgroup -g 1000 qwq && \
    adduser -D -u 1000 -G qwq qwq && \
    mkdir -p /app/data /app/logs /app/backups && \
    chown -R qwq:qwq /app

# 复制编译好的二进制文件
COPY --from=backend-builder --chown=qwq:qwq /app/qwq /app/qwq

# 切换到非 root 用户
USER qwq

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 使用 tini 作为 init 进程（处理僵尸进程）
ENTRYPOINT ["/sbin/tini", "--"]

# 默认启动命令
CMD ["/app/qwq", "web"]