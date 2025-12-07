#!/bin/bash
# ============================================
# 手动拉取 Docker 镜像脚本
# 解决镜像拉取失败问题
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
echo "  手动拉取 Docker 镜像"
echo "========================================"
echo ""

# 需要拉取的镜像列表
IMAGES=(
    "node:18-alpine"
    "golang:1.23-alpine"
    "alpine:3.19"
    "mysql:8.0"
    "redis:7-alpine"
    "prom/prometheus:latest"
    "grafana/grafana:latest"
)

# 国内镜像源列表（优先使用学校网络友好的镜像源）
MIRRORS=(
    "docker.m.daocloud.io"
    "reg-mirror.qiniu.com"
    "docker.mirrors.ustc.edu.cn"
    "hub-mirror.c.163.com"
    "mirror.baidubce.com"
    "ccr.ccs.tencentyun.com"
)

echo -e "${BLUE}准备拉取 ${#IMAGES[@]} 个镜像...${NC}"
echo ""

# 拉取每个镜像
for IMAGE in "${IMAGES[@]}"; do
    echo -e "${BLUE}[拉取] $IMAGE${NC}"
    
    # 尝试直接拉取
    if docker pull "$IMAGE" 2>/dev/null; then
        echo -e "${GREEN}✓ $IMAGE 拉取成功${NC}"
        echo ""
        continue
    fi
    
    # 如果直接拉取失败，尝试使用镜像源
    SUCCESS=false
    for MIRROR in "${MIRRORS[@]}"; do
        echo -e "${YELLOW}  尝试镜像源: $MIRROR${NC}"
        
        # 构造镜像源地址
        MIRROR_IMAGE="$MIRROR/library/$IMAGE"
        
        if docker pull "$MIRROR_IMAGE" 2>/dev/null; then
            # 重新标记为原始镜像名
            docker tag "$MIRROR_IMAGE" "$IMAGE"
            docker rmi "$MIRROR_IMAGE" 2>/dev/null || true
            echo -e "${GREEN}✓ $IMAGE 拉取成功（通过 $MIRROR）${NC}"
            SUCCESS=true
            break
        fi
    done
    
    if [ "$SUCCESS" = false ]; then
        echo -e "${RED}✗ $IMAGE 拉取失败${NC}"
        echo -e "${YELLOW}  请检查网络连接或手动拉取此镜像${NC}"
    fi
    
    echo ""
done

echo ""
echo "========================================"
echo -e "${GREEN}镜像拉取完成！${NC}"
echo "========================================"
echo ""
echo "已拉取的镜像："
docker images | grep -E "node|golang|alpine|mysql|redis|prometheus|grafana" || true
echo ""
echo "下一步："
echo "  运行部署脚本: sudo ./一键部署.sh"
echo "  或直接构建: docker compose build"
echo ""
