#!/bin/bash
# 快速诊断脚本

echo "=========================================="
echo "qwq AIOps 诊断工具"
echo "=========================================="
echo ""

echo "1. 检查容器状态"
docker compose ps
echo ""

echo "2. 检查服务日志（最后 20 行）"
docker compose logs qwq --tail 20
echo ""

echo "3. 测试 API 健康检查"
curl -s http://localhost:8081/api/health || echo "API 无响应"
echo ""

echo "4. 测试前端首页"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/)
echo "HTTP 状态码: $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ 前端可访问"
elif [ "$HTTP_CODE" = "401" ]; then
    echo "✓ 前端需要认证（正常）"
else
    echo "✗ 前端访问异常"
fi
echo ""

echo "5. 检查前端资源"
curl -s -o /dev/null -w "index.html: %{http_code}\n" http://localhost:8081/
curl -s -o /dev/null -w "assets/index.js: %{http_code}\n" http://localhost:8081/assets/index-*.js 2>/dev/null || echo "assets 文件检查失败"
echo ""

echo "6. 检查环境变量"
docker compose exec qwq env | grep -E "AI_PROVIDER|OLLAMA_HOST|DINGTALK_WEBHOOK|WEB_USER" || echo "环境变量未设置"
echo ""

echo "=========================================="
echo "诊断完成"
echo "=========================================="
