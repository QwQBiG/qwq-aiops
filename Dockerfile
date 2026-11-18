# Build the frontend with the same entry point used locally and in CI.
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder

WORKDIR /app

COPY frontend/ ./frontend/
COPY scripts/build.mjs ./scripts/build.mjs
RUN node scripts/build.mjs --frontend-only

# --- Stage 2: 后端构建 ---
FROM --platform=$BUILDPLATFORM golang:1.26.11-alpine AS backend-builder

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

# Copy generated assets last so checkout placeholders cannot replace them.
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=frontend-builder /app/internal/server/dist ./internal/server/dist
# 编译 Go 程序
ARG TARGETARCH
RUN GOARCH=${TARGETARCH:-amd64} go build \
    -trimpath -ldflags="-w -s" \
    -o qwq \
    ./cmd/qwq

# 验证编译结果
RUN chmod +x qwq && ls -lh qwq

# --- Stage 3: Final Runtime Image ---
FROM alpine:3.24

LABEL description="qwq Ops workbench" \
      org.opencontainers.image.source="https://github.com/QwQBiG/qwq-aiops" \
      org.opencontainers.image.documentation="https://github.com/QwQBiG/qwq-aiops/blob/main/README.md" \
      org.opencontainers.image.licenses="Apache-2.0"

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
COPY LICENSE /app/LICENSE

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
