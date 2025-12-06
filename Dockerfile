# --- Stage 1: Build Frontend (Vue) ---
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend

# 设置 npm 镜像源加速
RUN npm config set registry https://registry.npmmirror.com

# 复制前端配置
COPY frontend/package.json frontend/package-lock.json* ./
# 安装依赖
RUN npm install

# 复制前端源码
COPY frontend/ .
# 编译生成 dist 目录
RUN npm run build

# --- Stage 2: Build Backend (Go) ---
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct

# 复制 Go 依赖
COPY go.mod go.sum ./
RUN go mod tidy

# 复制 Go 源码
COPY . .

COPY --from=frontend-builder /app/frontend/dist ./internal/server/dist

# 编译 Go 程序
RUN go build -o qwq ./cmd/qwq/main.go

# --- Stage 3: Final Image ---
FROM alpine:latest
WORKDIR /root/

# 安装运维工具
RUN apk add --no-cache bash curl grep procps coreutils net-tools iproute2 tzdata tini docker-cli

# 安装 kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/

ENV TZ=Asia/Shanghai

# 复制二进制文件
COPY --from=backend-builder /app/qwq .

EXPOSE 8899
ENTRYPOINT ["/sbin/tini", "--", "./qwq"]
CMD ["web"]