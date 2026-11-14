#!/bin/bash
# ============================================
# 前端资源诊断脚本
# ============================================

echo "========================================"
echo "  前端资源诊断"
echo "========================================"
echo

# 检查本地前端构建
echo "[1] 检查本地前端构建"
echo "---"
if [ -d "frontend/dist" ]; then
    echo "✓ frontend/dist 目录存在"
    echo "  - index.html: $(test -f frontend/dist/index.html && echo '存在' || echo '不存在')"
    echo "  - assets 目录: $(test -d frontend/dist/assets && echo '存在' || echo '不存在')"
    
    if [ -d "frontend/dist/assets" ]; then
        TOTAL_FILES=$(find frontend/dist/assets -type f | wc -l)
        echo "  - assets 文件总数: $TOTAL_FILES"
        
        PLUGIN_FILES=$(ls frontend/dist/assets/_plugin-vue_export-helper-*.js 2>/dev/null)
        if [ -n "$PLUGIN_FILES" ]; then
            echo "  - plugin helper 文件:"
            for file in $PLUGIN_FILES; do
                SIZE=$(ls -lh "$file" | awk '{print $5}')
                echo "    * $(basename $file) ($SIZE)"
            done
        else
            echo "  ✗ plugin helper 文件不存在"
        fi
    fi
else
    echo "✗ frontend/dist 目录不存在"
    echo "  需要运行: cd frontend && npm run build"
fi
echo

# 检查 Docker 容器
echo "[2] 检查 Docker 容器"
echo "---"
if docker ps | grep -q qwq; then
    echo "✓ qwq 容器正在运行"
    
    # 检查容器内的前端文件
    echo
    echo "容器内前端文件检查:"
    
    # 检查 embed 目录是否存在
    if docker exec qwq test -d /app/internal/server/dist 2>/dev/null; then
        echo "  ✓ /app/internal/server/dist 目录存在"
        
        # 列出文件
        CONTAINER_FILES=$(docker exec qwq find /app/internal/server/dist -type f 2>/dev/null | wc -l)
        echo "  - 文件总数: $CONTAINER_FILES"
        
        # 检查 index.html
        if docker exec qwq test -f /app/internal/server/dist/index.html 2>/dev/null; then
            echo "  ✓ index.html 存在"
        else
            echo "  ✗ index.html 不存在"
        fi
        
        # 检查 assets 目录
        if docker exec qwq test -d /app/internal/server/dist/assets 2>/dev/null; then
            ASSETS_COUNT=$(docker exec qwq find /app/internal/server/dist/assets -type f 2>/dev/null | wc -l)
            echo "  ✓ assets 目录存在 ($ASSETS_COUNT 个文件)"
            
            # 检查 plugin helper
            PLUGIN_IN_CONTAINER=$(docker exec qwq ls /app/internal/server/dist/assets/_plugin-vue_export-helper-*.js 2>/dev/null)
            if [ -n "$PLUGIN_IN_CONTAINER" ]; then
                echo "  ✓ plugin helper 文件存在:"
                docker exec qwq ls -lh /app/internal/server/dist/assets/_plugin-vue_export-helper-*.js 2>/dev/null | awk '{print "    * " $9 " (" $5 ")"}'
            else
                echo "  ✗ plugin helper 文件不存在"
            fi
        else
            echo "  ✗ assets 目录不存在"
        fi
    else
        echo "  ✗ /app/internal/server/dist 目录不存在"
        echo "  这意味着前端文件没有被 embed 到 Go 二进制文件中"
    fi
else
    echo "✗ qwq 容器未运行"
    echo "  运行: docker compose up -d"
fi
echo

# 检查 HTTP 访问
echo "[3] 检查 HTTP 访问"
echo "---"
if curl -s http://localhost:8081 > /dev/null 2>&1; then
    echo "✓ 服务可访问 (http://localhost:8081)"
    
    # 测试关键文件
    echo
    echo "测试关键文件访问:"
    
    # index.html
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/)
    if [ "$STATUS" = "200" ]; then
        echo "  ✓ index.html: 200 OK"
    else
        echo "  ✗ index.html: $STATUS"
    fi
    
    # 查找 plugin helper 文件名
    PLUGIN_NAME=$(curl -s http://localhost:8081/ | grep -o '_plugin-vue_export-helper-[^"]*\.js' | head -n 1)
    if [ -n "$PLUGIN_NAME" ]; then
        echo "  - 检测到 plugin helper: $PLUGIN_NAME"
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/assets/$PLUGIN_NAME")
        if [ "$STATUS" = "200" ]; then
            echo "  ✓ $PLUGIN_NAME: 200 OK"
        else
            echo "  ✗ $PLUGIN_NAME: $STATUS (这是问题所在！)"
        fi
    else
        echo "  ✗ 无法从 index.html 中检测到 plugin helper 文件名"
    fi
else
    echo "✗ 服务不可访问"
fi
echo

# 诊断结论
echo "========================================"
echo "  诊断结论"
echo "========================================"
echo

# 判断问题
if [ ! -d "frontend/dist" ]; then
    echo "问题: 前端未构建"
    echo "解决: cd frontend && npm run build"
elif ! docker ps | grep -q qwq; then
    echo "问题: Docker 容器未运行"
    echo "解决: docker compose up -d"
elif ! docker exec qwq test -d /app/internal/server/dist/assets 2>/dev/null; then
    echo "问题: 前端文件未被 embed 到 Docker 镜像"
    echo "解决: ./fix-and-rebuild.sh"
else
    PLUGIN_IN_CONTAINER=$(docker exec qwq ls /app/internal/server/dist/assets/_plugin-vue_export-helper-*.js 2>/dev/null)
    if [ -z "$PLUGIN_IN_CONTAINER" ]; then
        echo "问题: plugin helper 文件未被 embed"
        echo "解决: ./fix-and-rebuild.sh"
    else
        echo "✓ 所有检查通过"
        echo
        echo "如果前端仍然报错，请："
        echo "1. 清除浏览器缓存 (Ctrl+Shift+Delete)"
        echo "2. 使用无痕模式访问"
        echo "3. 检查浏览器控制台的具体错误信息"
    fi
fi

echo
