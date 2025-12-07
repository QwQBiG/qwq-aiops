#!/bin/bash
# ============================================
# 批量替换 docker-compose 为 docker compose
# 解决 V1/V2 版本兼容问题
# ============================================

echo "开始替换 docker-compose 为 docker compose..."
echo ""

# 要替换的文件列表
files=(
    "README.md"
    "README_EN.md"
    "docs/deployment-guide.md"
    "PORT_CHANGE_GUIDE.md"
    "start.bat"
    "start.sh"
    "START_HERE.md"
    "fix-config.sh"
    "fix-config.bat"
    "一键部署说明.md"
    "配置文件问题修复.md"
    "网络问题已修复.md"
    "构建优化说明.md"
    "最新优化总结.md"
    "快速开始.md"
    "rebuild.sh"
    "rebuild.bat"
)

count=0
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        # 替换 docker-compose 为 docker compose
        sed -i 's/docker-compose/docker compose/g' "$file"
        echo "✓ 已更新: $file"
        ((count++))
    else
        echo "⚠ 文件不存在: $file"
    fi
done

echo ""
echo "=========================================="
echo "替换完成！共更新 $count 个文件"
echo "=========================================="
echo ""
echo "现在所有命令都使用 docker compose (V2)"
echo ""
