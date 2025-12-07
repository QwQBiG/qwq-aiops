#!/bin/bash
# ============================================
# Docker 镜像源配置脚本（Linux）
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
echo "  Docker 镜像源配置脚本"
echo "========================================"
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}[提示] 需要 root 权限，尝试使用 sudo...${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# 备份现有配置
if [ -f /etc/docker/daemon.json ]; then
    echo -e "${BLUE}[1/4] 备份现有配置...${NC}"
    $SUDO cp /etc/docker/daemon.json /etc/docker/daemon.json.backup.$(date +%Y%m%d%H%M%S)
    echo -e "${GREEN}✓ 备份完成${NC}"
else
    echo -e "${BLUE}[1/4] 创建 Docker 配置目录...${NC}"
    $SUDO mkdir -p /etc/docker
    echo -e "${GREEN}✓ 目录创建完成${NC}"
fi

echo ""

# 创建新配置
echo -e "${BLUE}[2/4] 配置国内镜像源...${NC}"
$SUDO tee /etc/docker/daemon.json > /dev/null <<'EOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://reg-mirror.qiniu.com",
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://ccr.ccs.tencentyun.com"
  ],
  "dns": ["8.8.8.8", "8.8.4.4"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF
echo -e "${GREEN}✓ 配置完成${NC}"

echo ""

# 重启 Docker
echo -e "${BLUE}[3/4] 重启 Docker 服务...${NC}"
$SUDO systemctl daemon-reload
$SUDO systemctl restart docker
echo -e "${GREEN}✓ Docker 重启完成${NC}"

echo ""

# 验证配置
echo -e "${BLUE}[4/4] 验证配置...${NC}"
if docker info | grep -q "Registry Mirrors"; then
    echo -e "${GREEN}✓ 镜像源配置成功${NC}"
    echo ""
    echo "已配置的镜像源："
    docker info | grep -A 5 "Registry Mirrors"
else
    echo -e "${YELLOW}⚠ 无法验证镜像源配置${NC}"
fi

echo ""
echo "========================================"
echo -e "${GREEN}  配置完成！${NC}"
echo "========================================"
echo ""
echo "现在可以重新构建："
echo -e "  ${BLUE}docker-compose build --no-cache${NC}"
echo -e "  ${BLUE}docker-compose up -d${NC}"
echo ""
