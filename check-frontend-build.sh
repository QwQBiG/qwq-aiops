#!/bin/bash
# 检查前端构建产物

echo "=========================================="
echo "检查前端构建产物"
echo "=========================================="
echo ""

echo "1. 检查本地 frontend/dist 目录"
echo "----------------------------------------"
if [ -d "frontend/dist" ]; then
    echo "✓ frontend/dist 目录存在"
    echo ""
    echo "assets 目录下的文件："
    ls -lh frontend/dist/assets/ | grep "_plugin-vue_export-helper" || echo "未找到 _plugin-vue_export-helper 文件"
    echo ""
    echo "所有 JS 文件："
    ls frontend/dist/assets/*.js | wc -l
    echo "个 JS 文件"
else
    echo "✗ frontend/dist 目录不存在"
fi

echo ""
echo "2. 检查二进制文件中嵌入的文件名"
echo "----------------------------------------"
echo "搜索 _plugin-vue_export-helper 相关文件："
docker compose exec qwq sh -c "strings /app/qwq | grep -i 'plugin.*vue.*export.*helper'" | head -10

echo ""
echo "3. 重新构建前端（本地测试）"
echo "----------------------------------------"
echo "进入前端目录并构建..."
cd frontend
npm run build 2>&1 | tail -20
cd ..

echo ""
echo "4. 再次检查构建产物"
echo "----------------------------------------"
ls -lh frontend/dist/assets/ | grep "_plugin-vue_export-helper" || echo "未找到 _plugin-vue_export-helper 文件"

echo ""
echo "=========================================="
echo "结论："
echo "如果本地构建后仍然没有这个文件，说明："
echo "1. Vite 配置可能有问题"
echo "2. 或者这个文件被合并到其他文件中了"
echo "3. 需要检查 frontend/vite.config.js 的 manualChunks 配置"
echo "=========================================="
