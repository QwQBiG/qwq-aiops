#!/bin/bash

# qwq AIOps 平台部署脚本
# 用途：快速部署 qwq 到生产环境

set -e

echo "========================================="
echo "  qwq AIOps 平台部署脚本"
echo "========================================="

# 配置变量
IMAGE_NAME="qwq-aiops"
IMAGE_TAG="${IMAGE_TAG:-latest}"
CONTAINER_NAME="qwq"
PORT="${PORT:-8899}"
DATA_DIR="${DATA_DIR:-./data}"

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装，请先安装 Docker"
    exit 1
fi

# 创建数据目录
echo "创建数据目录..."
mkdir -p "$DATA_DIR"

# 构建镜像
echo "构建 Docker 镜像..."
docker build -t "$IMAGE_NAME:$IMAGE_TAG" .

# 停止并删除旧容器
if docker ps -a | grep -q "$CONTAINER_NAME"; then
    echo "停止并删除旧容器..."
    docker stop "$CONTAINER_NAME" || true
    docker rm "$CONTAINER_NAME" || true
fi

# 启动新容器
echo "启动新容器..."
docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$PORT:8899" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "$DATA_DIR:/root/data" \
    -e TZ=Asia/Shanghai \
    "$IMAGE_NAME:$IMAGE_TAG"

# 等待服务启动
echo "等待服务启动..."
sleep 5

# 检查服务状态
if docker ps | grep -q "$CONTAINER_NAME"; then
    echo "========================================="
    echo "  部署成功！"
    echo "========================================="
    echo "访问地址: http://localhost:$PORT"
    echo "查看日志: docker logs -f $CONTAINER_NAME"
    echo "停止服务: docker stop $CONTAINER_NAME"
    echo "========================================="
else
    echo "错误: 容器启动失败"
    docker logs "$CONTAINER_NAME"
    exit 1
fi
