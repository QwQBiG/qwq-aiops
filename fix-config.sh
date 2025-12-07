#!/bin/bash
# ============================================
# qwq AIOps 平台 - 配置文件修复脚本
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
echo "  qwq AIOps 配置文件修复脚本"
echo "========================================"
echo ""

# 创建 config 目录
if [ ! -d "config" ]; then
    echo -e "${BLUE}[1/3] 创建 config 目录...${NC}"
    mkdir -p config
    echo -e "${GREEN}✓ config 目录创建成功${NC}"
else
    echo -e "${GREEN}[1/3] config 目录已存在${NC}"
fi

echo ""

# 创建 prometheus.yml
if [ ! -f "config/prometheus.yml" ]; then
    echo -e "${BLUE}[2/3] 创建 prometheus.yml...${NC}"
    cat > config/prometheus.yml <<'EOF'
# Prometheus 配置文件
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
EOF
    echo -e "${GREEN}✓ prometheus.yml 创建成功${NC}"
else
    echo -e "${GREEN}[2/3] prometheus.yml 已存在${NC}"
fi

echo ""

# 创建 mysql.cnf
if [ ! -f "config/mysql.cnf" ]; then
    echo -e "${BLUE}[3/3] 创建 mysql.cnf...${NC}"
    cat > config/mysql.cnf <<'EOF'
[mysqld]
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
default-authentication-plugin=mysql_native_password

[client]
default-character-set=utf8mb4
EOF
    echo -e "${GREEN}✓ mysql.cnf 创建成功${NC}"
else
    echo -e "${GREEN}[3/3] mysql.cnf 已存在${NC}"
fi

echo ""
echo "========================================"
echo -e "${GREEN}  配置文件修复完成！${NC}"
echo "========================================"
echo ""
echo "现在可以启动服务："
echo -e "  ${BLUE}docker-compose up -d${NC}"
echo ""
