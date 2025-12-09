#!/bin/bash
# 在容器内测试 embed 是否工作

echo "=========================================="
echo "容器内 embed 测试"
echo "=========================================="
echo ""

echo "1. 进入容器检查文件系统"
echo "----------------------------------------"
echo "注意：embed 的文件不会出现在文件系统中"
echo "它们被编译进了二进制文件"
echo ""

echo "2. 检查二进制文件大小"
echo "----------------------------------------"
docker compose exec qwq ls -lh /app/qwq
echo ""
echo "如果二进制文件很小（<50MB），说明前端文件没有嵌入"
echo "如果很大（>100MB），说明前端文件已嵌入"
echo ""

echo "3. 使用 strings 检查二进制文件"
echo "----------------------------------------"
echo "检查是否包含前端文件名："
docker compose exec qwq sh -c "strings /app/qwq | grep -E '(index\.html|Dashboard.*\.js|vendor.*\.js)' | head -10"
echo ""

echo "4. 测试 Go 程序能否读取 embed 文件"
echo "----------------------------------------"
echo "查看服务启动日志："
docker compose logs qwq 2>&1 | grep -E "(前端资源|embed|dist)" | tail -10
echo ""

echo "=========================================="
