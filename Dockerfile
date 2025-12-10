# ============================================
# qwq AIOps Platform - 修复版 Dockerfile
# 解决前端 404 问题的完整方案
# ============================================

# --- Stage 1: 前端构建 ---
FROM node:18-alpine AS frontend-builder

WORKDIR /app

# 使用国内镜像源加速
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    npm config set registry https://registry.npmmirror.com

# 复制前端项目文件
COPY frontend/package*.json ./frontend/
WORKDIR /app/frontend

# 安装依赖（包含开发依赖，因为构建需要）
RUN npm ci

# 复制前端源码并构建
COPY frontend/ .
RUN npm run build

# 验证前端构建结果
RUN echo "=== 前端构建验证 ===" && \
    ls -lh dist/ && \
    echo "Assets 文件:" && \
    ls -lh dist/assets/ | head -5 && \
    echo "Plugin 文件:" && \
    find dist -name "*plugin*" -type f

# --- Stage 2: 后端构建 ---
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# 使用国内镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates

# Go 环境配置
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on
ENV CGO_ENABLED=0

# 下载 Go 依赖
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 先创建目标目录结构
RUN mkdir -p internal/server/dist

# 从前端构建阶段复制构建产物到 Go embed 路径
COPY --from=frontend-builder /app/frontend/dist ./internal/server/dist

# 复制 Go 源码
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# 最终验证：确保 Go embed 能找到文件
RUN echo "=== Go Embed 路径验证 ===" && \
    echo "当前目录: $(pwd)" && \
    echo "internal/server/dist 内容:" && \
    ls -la ./internal/server/dist/ && \
    echo "Assets 目录:" && \
    ls -la ./internal/server/dist/assets/ | head -10 && \
    echo "Plugin 文件:" && \
    find ./internal/server/dist -name "*plugin*" -type f && \
    echo "文件总数: $(find ./internal/server/dist -type f | wc -l)" && \
    echo "=== 验证关键文件 ===" && \
    test -f ./internal/server/dist/assets/_plugin-vue_export-helper-DlAUqK2U.js && echo "✓ Plugin helper 文件存在" || echo "✗ Plugin helper 文件不存在" && \
    echo "文件大小: $(ls -lh ./internal/server/dist/assets/_plugin-vue_export-helper-DlAUqK2U.js 2>/dev/null || echo '文件不存在')"
# 编译 Go 程序
ARG TARGETARCH
RUN GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-w -s -X main.Version=1.0.0" \
    -o qwq \
    ./cmd/qwq/main.go

# 验证编译结果
RUN chmod +x qwq && ls -lh qwq

# --- Stage 3: Final Runtime Image ---
FROM alpine:3.19

LABEL maintainer="qwq AIOps Team" \
      version="1.0.0" \
      description="qwq AIOps - AI-Powered Intelligent Operations Platform" \
      org.opencontainers.image.source="https://github.com/QwQBiG/qwq-aiops" \
      org.opencontainers.image.documentation="https://github.com/QwQBiG/qwq-aiops/blob/main/README.md" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# 使用国内 Alpine 镜像源加速
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

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

# 安装 kubectl（可选，默认不安装以加快构建速度）
# 如需安装，构建时添加参数：--build-arg INSTALL_KUBECTL=true
ARG INSTALL_KUBECTL=false
ARG TARGETARCH
RUN if [ "$INSTALL_KUBECTL" = "true" ]; then \
    echo "正在安装 kubectl..." && \
    KUBECTL_VERSION=$(wget -qO- https://dl.k8s.io/release/stable.txt) && \
    ARCH=${TARGETARCH:-amd64} && \
    # 使用国内镜像加速（如果可用）
    (wget -q "https://kubernetes.oss-cn-hangzhou.aliyuncs.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" -O /usr/local/bin/kubectl || \
     wget -q "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl" -O /usr/local/bin/kubectl) && \
    chmod +x /usr/local/bin/kubectl && \
    echo "kubectl 安装完成"; \
    else \
    echo "跳过 kubectl 安装（加快构建速度）"; \
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