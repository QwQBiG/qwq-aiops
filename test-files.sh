#!/bin/bash
# 测试具体的文件访问

echo "=========================================="
echo "测试前端文件访问"
echo "=========================================="
echo ""

echo "1. 测试主要的 JS 文件"
echo "----------------------------------------"
files=(
    "assets/_plugin-vue_export-helper-DlAUqK2U.js"
    "assets/Dashboard-BrhmdLjO.js"
    "assets/Containers-CvrUOFh0.js"
    "assets/vendor-CtAA2Lrs.js"
    "assets/vue-vendor-DQ1NpETt.js"
    "assets/index-SrzHhsKe.js"
)

for file in "${files[@]}"; do
    echo "测试: $file"
    status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/$file")
    size=$(curl -s "http://localhost:8081/$file" | wc -c)
    echo "  状态码: $status"
    echo "  文件大小: $size bytes"
    if [ "$status" = "200" ] && [ "$size" -gt 100 ]; then
        echo "  ✓ 成功"
    else
        echo "  ✗ 失败"
        # 显示返回内容的前100个字符
        echo "  内容预览:"
        curl -s "http://localhost:8081/$file" | head -c 200
        echo ""
    fi
    echo ""
done

echo "2. 查看完整的服务日志（最后50行）"
echo "----------------------------------------"
docker compose logs qwq --tail 50

echo ""
echo "=========================================="
