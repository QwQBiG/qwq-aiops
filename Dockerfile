# 1. 编译阶段
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置国内代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制所有文件
COPY . .

# 下载依赖
RUN go mod tidy

# 指定路径
RUN go build -o qwq ./cmd/qwq/main.go

# -------------------------------------------

# 2. 运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 安装必要的 Linux 命令 (包含 tini 防止僵尸进程)
RUN apk add --no-cache \
    bash \
    curl \
    grep \
    procps \
    coreutils \
    net-tools \
    iproute2 \
    tzdata \
    tini

# 设置时区
ENV TZ=Asia/Shanghai

# 复制编译好的程序
COPY --from=builder /app/qwq .

# 暴露端口
EXPOSE 8899

# 启动命令 (使用 tini 管理进程)
ENTRYPOINT ["/sbin/tini", "--", "./qwq"]
CMD ["web"]