#!/bin/bash
# ============================================
# qwq AIOps 平台 - Ubuntu 部署脚本
# 专门为 Ubuntu 服务器优化的部署脚本
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo "========================================"
echo "  qwq AIOps 平台 - Ubuntu 部署"
echo "========================================"
echo ""

# 检查是否为 root 用户
if [ "$EUID" -eq 0 ]; then 
   echo -e "${YELLOW}警告: 不建议使用 root 用户运行${NC}"
   echo "建议使用普通用户，并确保用户在 docker 组中"
   read -p "是否继续？[y/N]: " continue_as_root
   if [[ ! $continue_as_root =~ ^[Yy]$ ]]; then
       exit 1
   fi
fi

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker 未安装${NC}"
    echo "安装 Docker:"
    echo "  curl -fsSL https://get.docker.com | sh"
    echo "  sudo usermod -aG docker $USER"
    exit 1
fi

# 检查 Docker Compose
if ! docker compose version &> /dev/null; then
    echo -e "${RED}Docker Compose V2 未安装${NC}"
    echo "安装 Docker Compose:"
    echo "  sudo apt-get update"
    echo "  sudo apt-get install docker-compose-plugin"
    exit 1
fi

# 检查 Docker 权限
if ! docker ps &> /dev/null; then
    echo -e "${RED}无法访问 Docker${NC}"
    echo "请确保："
    echo "  1. Docker 服务正在运行: sudo systemctl start docker"
    echo "  2. 用户在 docker 组中: sudo usermod -aG docker $USER"
    echo "  3. 重新登录或执行: newgrp docker"
    exit 1
fi

echo -e "${GREEN}✓ Docker 环境检查通过${NC}"
echo ""

# 检查必要文件
echo "检查项目文件..."
required_files=("docker-compose.yml" "Dockerfile" "go.mod")
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}✗ $file 不存在${NC}"
        exit 1
    fi
done
echo -e "${GREEN}✓ 项目文件完整${NC}"
echo ""

# 检查端口占用
echo "检查端口占用..."
ports=(8081 3308 6380 9091 3000)
port_conflict=false

for port in "${ports[@]}"; do
    if ss -tuln | grep -q ":$port "; then
        echo -e "${YELLOW}⚠ 端口 $port 已被占用${NC}"
        port_conflict=true
    fi
done

if [ "$port_conflict" = true ]; then
    echo ""
    echo -e "${YELLOW}部分端口被占用，可能导致服务启动失败${NC}"
    read -p "是否继续？[y/N]: " continue_deploy
    if [[ ! $continue_deploy =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 创建必要目录
echo ""
echo "创建数据目录..."
mkdir -p data logs backups config
echo -e "${GREEN}✓ 目录创建完成${NC}"

# 检查 .env 文件
if [ ! -f .env ]; then
    echo ""
    echo -e "${YELLOW}⚠ .env 文件不存在${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}✓ 已从 .env.example 创建 .env${NC}"
        echo -e "${YELLOW}请编辑 .env 文件配置 AI 服务${NC}"
    else
        echo -e "${RED}✗ .env.example 也不存在${NC}"
        exit 1
    fi
fi

# 停止现有服务
echo ""
echo "停止现有服务..."
docker compose down 2>/dev/null || true
echo -e "${GREEN}✓ 现有服务已停止${NC}"

# 构建镜像
echo ""
echo -e "${BLUE}开始构建镜像...${NC}"
echo "这可能需要几分钟时间，请耐心等待..."
echo ""

if docker compose build --no-cache; then
    echo ""
    echo -e "${GREEN}✓ 镜像构建成功${NC}"
else
    echo ""
    echo -e "${RED}✗ 镜像构建失败${NC}"
    echo ""
    echo "常见问题："
    echo "  1. 网络问题 - 检查网络连接"
    echo "  2. 磁盘空间 - 运行: df -h"
    echo "  3. 查看详细错误: docker compose build"
    exit 1
fi

# 启动服务
echo ""
echo -e "${BLUE}启动服务...${NC}"
if docker compose up -d; then
    echo ""
    echo -e "${GREEN}✓ 服务启动成功${NC}"
else
    echo ""
    echo -e "${RED}✗ 服务启动失败${NC}"
    echo "查看日志: docker compose logs"
    exit 1
fi

# 等待服务就绪
echo ""
echo "等待服务就绪..."
sleep 10

# 检查服务状态
echo ""
echo "服务状态:"
docker compose ps

echo ""
echo "========================================"
echo -e "${GREEN}  部署完成！${NC}"
echo "========================================"
echo ""
echo "访问地址："
echo -e "  前端界面: ${BLUE}http://$(hostname -I | awk '{print $1}'):8081${NC}"
echo -e "  本地访问: ${BLUE}http://localhost:8081${NC}"
echo ""
echo "常用命令："
echo "  查看日志: docker compose logs -f qwq"
echo "  查看状态: docker compose ps"
echo "  停止服务: docker compose down"
echo "  重启服务: docker compose restart qwq"
echo ""
echo "========================================"
echo ""

