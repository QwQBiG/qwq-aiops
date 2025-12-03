# 1. 编译阶段
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置国内代理 (虽然在 GitHub Actions 里可能用不到，但保留无妨)
ENV GOPROXY=https://goproxy.cn,direct

# 直接复制所有文件（包括源代码）
COPY . .

# 源代码都在
RUN go mod tidy

# 编译
RUN go build -o qwq main.go

# -------------------------------------------

# 2. 运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /root/

# 安装必要的 Linux 命令
RUN apk add --no-cache \
    bash \
    curl \
    grep \
    procps \
    coreutils \
    net-tools \
    iproute2 \
    tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制编译好的程序
COPY --from=builder /app/qwq .

# 暴露端口
EXPOSE 8899

# 启动命令
ENTRYPOINT ["./qwq"]
CMD ["web"]