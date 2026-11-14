#!/bin/bash
# ============================================
# qwq AIOps - 完整修复和重建脚本
# ============================================

set -e  # 遇到错误立即退出

echo "========================================"
echo "  qwq AIOps 完整修复和重建"
echo "========================================"
echo

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 步骤 1：检查环境
echo -e "${YELLOW}[1/6] 检查环境...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker 未安装${NC}"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo -e "${RED}错误: Node.js 未安装${NC}"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo -e "${RED}错误: npm 未安装${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 环境检查通过${NC}"
echo

# 步骤 2：清理旧的前端构建
echo -e "${YELLOW}[2/6] 清理旧的前端构建...${NC}"
if [ -d "frontend/dist" ]; then
    rm -rf frontend/dist
    echo -e "${GREEN}✓ 已清理旧构建${NC}"
else
    echo "无需清理"
fi
echo

# 步骤 3：重新构建前端
echo -e "${YELLOW}[3/6] 重新构建前端...${NC}"
cd frontend

# 检查 node_modules
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    npm install
fi

# 构建前端
echo "开始构建..."
npm run build

# 验证构建结果
if [ ! -f "dist/index.html" ]; then
    echo -e "${RED}错误: 前端构建失败，index.html 不存在${NC}"
    exit 1
fi

if [ ! -d "dist/assets" ]; then
    echo -e "${RED}错误: 前端构建失败，assets 目录不存在${NC}"
    exit 1
fi

# 检查关键文件
PLUGIN_FILE=$(ls dist/assets/_plugin-vue_export-helper-*.js 2>/dev/null | head -n 1)
if [ -z "$PLUGIN_FILE" ]; then
    echo -e "${RED}错误: 前端构建失败，plugin helper 文件不存在${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 前端构建成功${NC}"
echo "  - index.html: $(ls -lh dist/index.html | awk '{print $5}')"
echo "  - assets 文件数: $(find dist/assets -type f | wc -l)"
echo "  - plugin helper: $(basename $PLUGIN_FILE)"
echo

cd ..

# 步骤 4：停止现有容器
echo -e "${YELLOW}[4/6] 停止现有容器...${NC}"
docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
echo -e "${GREEN}✓ 容器已停止${NC}"
echo

# 步骤 5：重新构建 Docker 镜像（不使用缓存）
echo -e "${YELLOW}[5/6] 重新构建 Docker 镜像...${NC}"
echo "提示: 这可能需要 5-10 分钟，请耐心等待..."
echo

# 尝试使用 docker compose（新版本）
if docker compose version &> /dev/null; then
    docker compose build --no-cache --progress=plain
else
    docker-compose build --no-cache --progress=plain
fi

if [ $? -ne 0 ]; then
    echo -e "${RED}错误: Docker 镜像构建失败${NC}"
    echo
    echo "可能的原因："
    echo "1. 网络问题 - 检查 Docker 镜像源配置"
    echo "2. 磁盘空间不足 - 运行: docker system df"
    echo "3. 前端文件问题 - 检查 frontend/dist 目录"
    echo
    echo "调试命令："
    echo "  docker system df          # 检查磁盘使用"
    echo "  docker system prune -a    # 清理所有未使用的资源"
    echo "  docker images             # 查看镜像列表"
    exit 1
fi

echo -e "${GREEN}✓ Docker 镜像构建成功${NC}"
echo

# 步骤 6：启动服务
echo -e "${YELLOW}[6/6] 启动服务...${NC}"
if docker compose version &> /dev/null; then
    docker compose up -d
else
    docker-compose up -d
fi

if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 服务启动失败${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 服务启动成功${NC}"
echo

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 健康检查
echo "检查服务健康状态..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 服务运行正常${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${YELLOW}警告: 服务可能还在启动中${NC}"
        echo "运行以下命令查看日志："
        echo "  docker compose logs -f qwq"
    fi
    sleep 2
done

echo
echo "========================================"
echo -e "${GREEN}  修复和重建完成！${NC}"
echo "========================================"
echo
echo "访问地址："
echo "  前端界面: http://localhost:8081"
echo "  或: http://192.168.50.15:8081"
echo
echo "查看日志："
echo "  docker compose logs -f qwq"
echo
echo "验证前端资源："
echo "  curl -I http://localhost:8081/assets/_plugin-vue_export-helper-DlAUqK2U.js"
echo
echo "========================================"
