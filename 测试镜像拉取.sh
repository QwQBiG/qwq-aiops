#!/bin/bash
# ============================================
# 测试 Docker 镜像拉取
# 快速验证网络和镜像源配置
# ============================================

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo "========================================"
echo "  测试 Docker 镜像拉取"
echo "========================================"
echo ""

# 测试镜像
TEST_IMAGE="alpine:latest"

echo -e "${BLUE}[1/4] 检查 Docker 服务状态...${NC}"
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}✗ Docker 服务未运行${NC}"
    echo "请先启动 Docker 服务："
    echo "  sudo systemctl start docker"
    exit 1
fi
echo -e "${GREEN}✓ Docker 服务正常${NC}"
echo ""

echo -e "${BLUE}[2/4] 检查镜像源配置...${NC}"
if docker info | grep -q "Registry Mirrors"; then
    echo -e "${GREEN}✓ 镜像源已配置${NC}"
    docker info | grep -A 4 "Registry Mirrors"
else
    echo -e "${YELLOW}⚠ 未配置镜像源${NC}"
    echo "建议运行: sudo ./fix-docker-mirror.sh"
fi
echo ""

echo -e "${BLUE}[3/4] 测试网络连接...${NC}"

# 测试 Docker Hub
echo -n "  Docker Hub: "
if curl -s --connect-timeout 5 https://registry-1.docker.io/v2/ >/dev/null 2>&1; then
    echo -e "${GREEN}可访问${NC}"
else
    echo -e "${RED}无法访问${NC}"
fi

# 测试镜像源
echo -n "  DaoCloud 镜像源: "
if curl -s --connect-timeout 5 https://docker.m.daocloud.io/v2/ >/dev/null 2>&1; then
    echo -e "${GREEN}可访问${NC}"
else
    echo -e "${RED}无法访问${NC}"
fi

echo -n "  七牛云镜像源: "
if curl -s --connect-timeout 5 https://reg-mirror.qiniu.com/v2/ >/dev/null 2>&1; then
    echo -e "${GREEN}可访问${NC}"
else
    echo -e "${RED}无法访问${NC}"
fi

echo -n "  USTC 镜像源: "
if curl -s --connect-timeout 5 https://docker.mirrors.ustc.edu.cn/v2/ >/dev/null 2>&1; then
    echo -e "${GREEN}可访问${NC}"
else
    echo -e "${RED}无法访问${NC}"
fi

echo -n "  网易镜像源: "
if curl -s --connect-timeout 5 https://hub-mirror.c.163.com/v2/ >/dev/null 2>&1; then
    echo -e

echo ""

echo -e "${BLUE}[4/4] 测试拉取镜像...${NC}"
echo "  尝试拉取: $TEST_IMAGE"

if docker pull "$TEST_IMAGE" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ 镜像拉取成功${NC}"
    
    # 清理测试镜像
    docker rmi "$TEST_IMAGE" >/dev/null 2>&1
    
    echo ""
    echo "========================================"
    echo -e "${GREEN}  测试通过！${NC}"
    echo "========================================"
    echo ""
    echo "您的 Docker 配置正常，可以开始部署："
    echo "  sudo ./一键部署.sh"
    echo ""
else
    echo -e "${RED}✗ 镜像拉取失败${NC}"
    echo ""
    echo "========================================"
    echo -e "${RED}  测试失败${NC}"
    echo "========================================"
    echo ""
    echo "建议的解决方案："
    echo ""
    echo "1. 配置镜像源："
    echo "   sudo ./fix-docker-mirror.sh"
    echo ""
    echo "2. 手动拉取镜像："
    echo "   ./手动拉取镜像.sh"
    echo ""
    echo "3. 查看详细文档："
    echo "   cat 镜像拉取失败解决方案.md"
    echo ""
    exit 1
fi
