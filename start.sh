#!/bin/bash
# ============================================
# qwq AIOps 平台 - Linux/macOS 启动脚本
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
echo "  qwq AIOps 平台启动脚本"
echo "========================================"
echo ""

# 检查 Docker 是否运行
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}[错误] Docker 未运行，请先启动 Docker${NC}"
    exit 1
fi

echo -e "${GREEN}[1/4] 检查 Docker 环境... OK${NC}"
echo ""

# 检查是否存在 .env 文件
if [ ! -f .env ]; then
    echo -e "${YELLOW}[提示] 未找到 .env 文件，正在创建...${NC}"
    cp .env.example .env
    echo -e "${YELLOW}[提示] 请编辑 .env 文件配置 AI API Key${NC}"
    echo ""
fi

echo -e "${GREEN}[2/4] 检查配置文件... OK${NC}"
echo ""

# 停止现有容器
echo -e "${BLUE}[3/4] 停止现有容器...${NC}"
docker-compose down
echo ""

# 构建并启动
echo -e "${BLUE}[4/4] 构建并启动服务（首次运行需要 5-10 分钟）...${NC}"
echo ""
docker-compose up -d --build

if [ $? -ne 0 ]; then
    echo ""
    echo -e "${RED}[错误] 启动失败，请查看错误信息${NC}"
    exit 1
fi

echo ""
echo "========================================"
echo -e "${GREEN}  启动成功！${NC}"
echo "========================================"
echo ""
echo "访问地址："
echo -e "  前端界面: ${BLUE}http://localhost:8081${NC}"
echo -e "  API 文档: ${BLUE}http://localhost:8081/api/docs${NC}"
echo -e "  健康检查: ${BLUE}http://localhost:8081/api/health${NC}"
echo ""
echo "默认账号："
echo "  用户名: admin"
echo "  密码: admin123"
echo ""
echo "查看日志："
echo "  docker-compose logs -f qwq"
echo ""
echo "停止服务："
echo "  docker-compose down"
echo ""
echo "========================================"

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 健康检查
echo "正在检查服务状态..."
if curl -s http://localhost:8081/api/health >/dev/null 2>&1; then
    echo -e "${GREEN}[成功] 服务运行正常${NC}"
    echo ""
    
    # 尝试在浏览器中打开（仅 macOS 和某些 Linux）
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "是否在浏览器中打开？(y/n)"
        read -r open
        if [[ "$open" == "y" || "$open" == "Y" ]]; then
            open http://localhost:8081
        fi
    elif command -v xdg-open >/dev/null 2>&1; then
        echo "是否在浏览器中打开？(y/n)"
        read -r open
        if [[ "$open" == "y" || "$open" == "Y" ]]; then
            xdg-open http://localhost:8081
        fi
    fi
else
    echo -e "${YELLOW}[警告] 服务可能还在启动中，请稍后访问${NC}"
    echo "或运行: docker-compose logs -f qwq 查看日志"
fi

echo ""
