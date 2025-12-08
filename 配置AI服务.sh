#!/bin/bash
# ============================================
# qwq AIOps - AI 服务配置向导
# ============================================

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo "========================================"
echo "  qwq AIOps - AI 服务配置"
echo "========================================"
echo ""

echo "qwq 是 AI 驱动的智能运维平台，需要配置 AI 服务。"
echo ""
echo "支持的 AI 服务："
echo "  1) OpenAI API - 需要 API Key，功能最强"
echo "  2) Ollama - 本地模型，完全免费，数据不出服务器"
echo "  3) 其他兼容 OpenAI API 的服务"
echo ""

read -p "请选择 AI 服务类型 [1-3]: " AI_CHOICE

# 备份配置文件
if [ -f docker-compose.yml ]; then
    cp docker-compose.yml docker-compose.yml.backup.$(date +%Y%m%d%H%M%S)
    echo -e "${GREEN}✓ 已备份配置文件${NC}"
else
    echo -e "${RED}✗ 找不到 docker-compose.yml 文件${NC}"
    exit 1
fi

case $AI_CHOICE in
    1)
        echo ""
        echo -e "${BLUE}配置 OpenAI API${NC}"
        echo ""
        read -p "请输入你的 OpenAI API Key: " OPENAI_KEY
        
        if [ -z "$OPENAI_KEY" ]; then
            echo -e "${RED}✗ API Key 不能为空${NC}"
            exit 1
        fi
        
        read -p "请输入 API 地址 [默认: https://api.openai.com/v1]: " OPENAI_URL
        OPENAI_URL=${OPENAI_URL:-https://api.openai.com/v1}
        
        read -p "请输入模型名称 [默认: gpt-3.5-turbo]: " OPENAI_MODEL
        OPENAI_MODEL=${OPENAI_MODEL:-gpt-3.5-turbo}
        
        # 先注释掉所有 AI 配置
        sed -i '/AI_PROVIDER=/s/^      - /      # - /' docker-compose.yml
        sed -i '/OPENAI_/s/^      - /      # - /' docker-compose.yml
        sed -i '/OLLAMA_/s/^      - /      # - /' docker-compose.yml
        
        # 启用 OpenAI 配置
        sed -i '/# 方式 1: OpenAI API/,/# - OPENAI_MODEL=/s/^      # - /      - /' docker-compose.yml
        sed -i "s|OPENAI_API_KEY=.*|OPENAI_API_KEY=$OPENAI_KEY|" docker-compose.yml
        sed -i "s|OPENAI_BASE_URL=.*|OPENAI_BASE_URL=$OPENAI_URL|" docker-compose.yml
        sed -i "s|OPENAI_MODEL=.*|OPENAI_MODEL=$OPENAI_MODEL|" docker-compose.yml
        
        echo -e "${GREEN}✓ OpenAI 配置完成${NC}"
        ;;
        
    2)
        echo ""
        echo -e "${BLUE}配置 Ollama${NC}"
        echo ""
        
        # 检查 Ollama 是否安装
        if command -v ollama &> /dev/null; then
            echo -e "${GREEN}✓ 检测到 Ollama 已安装${NC}"
            OLLAMA_HOST="http://host.docker.internal:11434"
        else
            echo -e "${YELLOW}⚠ 未检测到 Ollama${NC}"
            echo ""
            echo "请选择："
            echo "  1) 安装 Ollama（推荐）"
            echo "  2) 手动输入 Ollama 地址"
            echo ""
            read -p "请选择 [1-2]: " OLLAMA_CHOICE
            
            if [ "$OLLAMA_CHOICE" = "1" ]; then
                echo ""
                echo "正在安装 Ollama..."
                curl -fsSL https://ollama.com/install.sh | sh
                
                echo ""
                echo "下载推荐模型（qwen2.5:7b）..."
                ollama pull qwen2.5:7b
                
                OLLAMA_HOST="http://host.docker.internal:11434"
            else
                read -p "请输入 Ollama 服务地址 [默认: http://host.docker.internal:11434]: " OLLAMA_HOST
                OLLAMA_HOST=${OLLAMA_HOST:-http://host.docker.internal:11434}
            fi
        fi
        
        read -p "请输入模型名称 [默认: qwen2.5:7b]: " OLLAMA_MODEL
        OLLAMA_MODEL=${OLLAMA_MODEL:-qwen2.5:7b}
        
        # 测试连接
        echo ""
        echo "测试 Ollama 连接..."
        if curl -s --connect-timeout 5 "$OLLAMA_HOST/api/tags" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ Ollama 连接成功${NC}"
        else
            echo -e "${YELLOW}⚠ 无法连接到 Ollama${NC}"
            echo "请确保 Ollama 服务已启动"
            echo ""
            echo "启动 Ollama 服务："
            echo "  systemctl start ollama"
            echo "或"
            echo "  ollama serve"
            echo ""
            read -p "是否继续配置？[y/N]: " CONTINUE
            if [[ ! $CONTINUE =~ ^[Yy]$ ]]; then
                exit 1
            fi
        fi
        
        # 先注释掉所有 AI 配置
        sed -i '/AI_PROVIDER=/s/^      - /      # - /' docker-compose.yml
        sed -i '/OPENAI_/s/^      - /      # - /' docker-compose.yml
        sed -i '/OLLAMA_/s/^      - /      # - /' docker-compose.yml
        
        # 启用 Ollama 配置
        sed -i '/# 方式 2: Ollama 本地模型/,/# - OLLAMA_MODEL=/s/^      # - /      - /' docker-compose.yml
        sed -i "s|OLLAMA_HOST=.*|OLLAMA_HOST=$OLLAMA_HOST|" docker-compose.yml
        sed -i "s|OLLAMA_MODEL=.*|OLLAMA_MODEL=$OLLAMA_MODEL|" docker-compose.yml
        
        echo -e "${GREEN}✓ Ollama 配置完成${NC}"
        ;;
        
    3)
        echo ""
        echo -e "${BLUE}配置自定义 API${NC}"
        echo ""
        read -p "请输入 API 地址: " API_URL
        read -p "请输入 API Key: " API_KEY
        read -p "请输入模型名称: " API_MODEL
        
        if [ -z "$API_URL" ] || [ -z "$API_KEY" ]; then
            echo -e "${RED}✗ API 地址和 Key 不能为空${NC}"
            exit 1
        fi
        
        # 先注释掉所有 AI 配置
        sed -i '/AI_PROVIDER=/s/^      - /      # - /' docker-compose.yml
        sed -i '/OPENAI_/s/^      - /      # - /' docker-compose.yml
        sed -i '/OLLAMA_/s/^      - /      # - /' docker-compose.yml
        
        # 启用 OpenAI 配置（兼容模式）
        sed -i '/# 方式 1: OpenAI API/,/# - OPENAI_MODEL=/s/^      # - /      - /' docker-compose.yml
        sed -i "s|OPENAI_API_KEY=.*|OPENAI_API_KEY=$API_KEY|" docker-compose.yml
        sed -i "s|OPENAI_BASE_URL=.*|OPENAI_BASE_URL=$API_URL|" docker-compose.yml
        sed -i "s|OPENAI_MODEL=.*|OPENAI_MODEL=$API_MODEL|" docker-compose.yml
        
        echo -e "${GREEN}✓ 自定义 API 配置完成${NC}"
        ;;
        
    *)
        echo -e "${RED}✗ 无效选项${NC}"
        exit 1
        ;;
esac

echo ""
echo "========================================"
echo -e "${GREEN}  配置完成！${NC}"
echo "========================================"
echo ""
echo "下一步："
echo "  1. 重启服务: docker compose restart qwq"
echo "  2. 查看日志: docker compose logs -f qwq"
echo "  3. 访问系统: http://localhost:8081"
echo ""
echo "验证 AI 配置："
echo "  curl http://localhost:8081/api/health"
echo ""
