#!/bin/bash
# ============================================
# qwq AIOps 平台 - 一键部署脚本
# 自动配置 Docker 镜像源并构建启动
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
echo "  qwq AIOps 平台 - 一键部署"
echo "========================================"
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}[提示] 需要 root 权限，尝试使用 sudo...${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# 步骤 1: 配置 Docker 镜像源
echo -e "${BLUE}[1/6] 配置 Docker 国内镜像源...${NC}"

# 备份现有配置
if [ -f /etc/docker/daemon.json ]; then
    $SUDO cp /etc/docker/daemon.json /etc/docker/daemon.json.backup.$(date +%Y%m%d%H%M%S)
    echo -e "${GREEN}✓ 已备份现有配置${NC}"
else
    $SUDO mkdir -p /etc/docker
fi

# 创建新配置
$SUDO tee /etc/docker/daemon.json > /dev/null <<'EOF'
{
  "registry-mirrors": [
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

echo -e "${GREEN}✓ Docker 镜像源配置完成${NC}"
echo ""

# 步骤 2: 重启 Docker
echo -e "${BLUE}[2/6] 重启 Docker 服务...${NC}"
$SUDO systemctl daemon-reload
$SUDO systemctl restart docker
sleep 3
echo -e "${GREEN}✓ Docker 重启完成${NC}"
echo ""

# 步骤 3: 验证镜像源
echo -e "${BLUE}[3/6] 验证镜像源配置...${NC}"
if docker info | grep -q "Registry Mirrors"; then
    echo -e "${GREEN}✓ 镜像源配置成功${NC}"
    docker info | grep -A 4 "Registry Mirrors"
else
    echo -e "${YELLOW}⚠ 无法验证镜像源，但继续尝试构建${NC}"
fi
echo ""

# 步骤 4: 创建配置文件
echo -e "${BLUE}[4/6] 创建配置文件...${NC}"

# 创建 config 目录
mkdir -p config

# 创建 prometheus.yml
if [ ! -f config/prometheus.yml ]; then
    cat > config/prometheus.yml <<'PROM'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'qwq'
    static_configs:
      - targets: ['qwq:8899']
    metrics_path: '/metrics'
PROM
    echo -e "${GREEN}✓ prometheus.yml 创建完成${NC}"
else
    echo -e "${GREEN}✓ prometheus.yml 已存在${NC}"
fi

# 创建 mysql.cnf
if [ ! -f config/mysql.cnf ]; then
    cat > config/mysql.cnf <<'MYSQL'
[mysqld]
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
default-authentication-plugin=mysql_native_password

[client]
default-character-set=utf8mb4
MYSQL
    echo -e "${GREEN}✓ mysql.cnf 创建完成${NC}"
else
    echo -e "${GREEN}✓ mysql.cnf 已存在${NC}"
fi

echo ""

# 步骤 5: 停止现有容器
echo -e "${BLUE}[5/6] 停止现有容器...${NC}"
docker compose down 2>/dev/null || true
echo -e "${GREEN}✓ 现有容器已停止${NC}"
echo ""

# 步骤 6: 构建并启动
echo -e "${BLUE}[6/6] 构建并启动服务...${NC}"
echo -e "${YELLOW}[提示] 首次构建需要 6-10 分钟，请耐心等待...${NC}"
echo ""

# 构建
if docker compose build --no-cache; then
    echo ""
    echo -e "${GREEN}✓ 构建成功${NC}"
    echo ""
    
    # 启动
    echo -e "${BLUE}启动服务...${NC}"
    if docker compose up -d; then
        echo ""
        echo "========================================"
        echo -e "${GREEN}  部署成功！${NC}"
        echo "========================================"
        echo ""
        echo "访问地址："
        echo -e "  前端界面: ${BLUE}http://localhost:8081${NC}"
        echo -e "  API 文档: ${BLUE}http://localhost:8081/api/docs${NC}"
        echo -e "  Prometheus: ${BLUE}http://localhost:9090${NC}"
        echo -e "  Grafana: ${BLUE}http://localhost:3000${NC}"
        echo ""
        echo "默认账号："
        echo "  用户名: admin"
        echo "  密码: admin123"
        echo ""
        echo "查看日志："
        echo "  docker compose logs -f qwq"
        echo ""
        echo "查看服务状态："
        echo "  docker compose ps"
        echo ""
        echo "========================================"
        
        # 等待服务启动
        echo ""
        echo -e "${BLUE}等待服务启动...${NC}"
        sleep 10
        
        # 健康检查
        if curl -s http://localhost:8081/api/health >/dev/null 2>&1; then
            echo -e "${GREEN}✓ 服务运行正常${NC}"
        else
            echo -e "${YELLOW}⚠ 服务可能还在启动中，请稍后访问${NC}"
            echo "运行以下命令查看日志："
            echo "  docker compose logs -f qwq"
        fi
    else
        echo ""
        echo -e "${RED}✗ 启动失败${NC}"
        echo "查看错误日志："
        echo "  docker compose logs"
        exit 1
    fi
else
    echo ""
    echo -e "${RED}✗ 构建失败${NC}"
    echo ""
    echo "可能的原因："
    echo "1. 网络问题 - 检查网络连接"
    echo "2. 磁盘空间不足 - 运行: df -h"
    echo "3. 依赖问题 - 查看上面的错误信息"
    echo ""
    echo "尝试手动拉取镜像："
    echo "  docker pull node:18-alpine"
    echo "  docker pull golang:1.23-alpine"
    echo "  docker pull alpine:3.19"
    echo ""
    exit 1
fi

echo ""
