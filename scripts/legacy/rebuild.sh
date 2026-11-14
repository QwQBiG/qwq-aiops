#!/bin/bash
# ============================================
# qwq AIOps 平台 - 重新构建脚本（Linux/macOS）
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "========================================"
echo "  qwq AIOps 平台重新构建脚本"
echo "========================================"
echo ""

# 检查 Docker 是否运行
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}[错误] Docker 未运行，请先启动 Docker${NC}"
    exit 1
fi

echo -e "${GREEN}[1/5] 检查 Docker 环境... OK${NC}"
echo ""

# 停止并删除容器
echo -e "${BLUE}[2/5] 停止并删除现有容器...${NC}"
docker-compose down
echo ""

# 清理 Docker 缓存
echo -e "${BLUE}[3/5] 清理 Docker 缓存...${NC}"
docker system prune -f
echo ""

# 重新构建（不使用缓存）
echo -e "${BLUE}[4/5] 重新构建镜像（不使用缓存）...${NC}"
echo -e "${YELLOW}[提示] 这可能需要 5-10 分钟，请耐心等待...${NC}"
echo ""
docker-compose build --no-cache --progress=plain

if [ $? -ne 0 ]; then
    echo ""
    echo -e "${RED}[错误] 构建失败${NC}"
    echo ""
    echo "可能的原因："
    echo "1. 网络问题 - 查看 NETWORK_FIX.md"
    echo "2. 磁盘空间不足 - 运行 docker system df 检查"
    echo "3. 依赖问题 - 查看上面的错误信息"
    echo ""
    exit 1
fi

echo ""
echo -e "${BLUE}[5/5] 启动服务...${NC}"
docker-compose up -d

if [ $? -ne 0 ]; then
    echo ""
    echo -e "${RED}[错误] 启动失败${NC}"
    exit 1
fi

echo ""
echo "========================================"
echo -e "${GREEN}  重新构建成功！${NC}"
echo "========================================"
echo ""
echo "访问地址："
echo -e "  前端界面: ${BLUE}http://localhost:8081${NC}"
echo -e "  API 文档: ${BLUE}http://localhost:8081/api/docs${NC}"
echo ""
echo "查看日志："
echo "  docker-compose logs -f qwq"
echo ""
echo "========================================"

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 健康检查
echo "正在检查服务状态..."
if curl -s http://localhost:8081/api/health >/dev/null 2>&1; then
    echo -e "${GREEN}[成功] 服务运行正常${NC}"
else
    echo -e "${YELLOW}[警告] 服务可能还在启动中${NC}"
    echo "运行: docker-compose logs -f qwq 查看日志"
fi

echo ""
