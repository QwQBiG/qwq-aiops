#!/bin/bash
# 验证前端文件是否正确嵌入

echo "=========================================="
echo "验证前端文件嵌入情况"
echo "=========================================="
echo ""

echo "1. 检查构建日志中的前端文件验证"
echo "----------------------------------------"
docker compose logs qwq 2>&1 | grep -A 5 "前端资源验证" || echo "未找到验证日志"
echo ""

echo "2. 尝试访问具体的 assets 文件"
echo "----------------------------------------"
echo "测试 _plugin-vue_export-helper-DlAUqK2U.js:"
curl -I http://localhost:8081/assets/_plugin-vue_export-helper-DlAUqK2U.js 2>&1 | head -5
echo ""

echo "测试 Dashboard-BrhmdLjO.js:"
curl -I http://localhost:8081/assets/Dashboard-BrhmdLjO.js 2>&1 | head -5
echo ""

echo "3. 列出所有可访问的 assets 文件"
echo "----------------------------------------"
echo "尝试访问 assets 目录（可能返回 404 或目录列表）:"
curl -s http://localhost:8081/assets/ | head -20
echo ""

echo "4. 检查 index.html 中引用的文件"
echo "----------------------------------------"
curl -s http://localhost:8081/ | grep -o 'src="[^"]*"' | head -10
echo ""

echo "=========================================="
echo "建议："
echo "1. 如果所有 assets 文件都 404，说明 embed 失败"
echo "2. 检查构建日志中的 '前端资源验证' 部分"
echo "3. 可能需要在本地构建测试"
echo "=========================================="
