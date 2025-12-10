#!/bin/bash

echo "============================================"
echo "qwq AIOps 平台 - 配置诊断工具"
echo "============================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 .env 文件
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}[警告]${NC} .env 配置文件不存在"
    echo ""
    read -p "是否要从 .env.example 创建 .env 文件？(Y/n): " CREATE_ENV
    if [ "$CREATE_ENV" = "Y" ] || [ "$CREATE_ENV" = "y" ] || [ -z "$CREATE_ENV" ]; then
        cp .env.example .env
        echo -e "${GREEN}[成功]${NC} .env 文件已创建，请编辑配置"
    fi
else
    echo -e "${GREEN}[成功]${NC} .env 配置文件存在"
fi
echo ""

# 检查必需的配置项
echo "检查必需的配置项..."
echo ""

# 检查 JWT_SECRET
if grep -q "^JWT_SECRET=" .env 2>/dev/null; then
    JWT_VALUE=$(grep "^JWT_SECRET=" .env | cut -d'=' -f2)
    if [ "$JWT_VALUE" = "change-this-to-a-random-secret-key-at-least-32-characters" ]; then
        echo -e "${YELLOW}[警告]${NC} JWT_SECRET 使用默认值，请修改为随机密钥"
    else
        echo -e "${GREEN}[成功]${NC} JWT_SECRET 已配置"
    fi
else
    echo -e "${RED}[错误]${NC} JWT_SECRET 未配置"
fi

# 检查 ENCRYPTION_KEY
if grep -q "^ENCRYPTION_KEY=" .env 2>/dev/null; then
    ENC_VALUE=$(grep "^ENCRYPTION_KEY=" .env | cut -d'=' -f2)
    if [ "$ENC_VALUE" = "change-this-to-32-byte-key-here" ]; then
        echo -e "${YELLOW}[警告]${NC} ENCRYPTION_KEY 使用默认值，请修改为随机密钥"
    else
        echo -e "${GREEN}[成功]${NC} ENCRYPTION_KEY 已配置"
    fi
else
    echo -e "${RED}[错误]${NC} ENCRYPTION_KEY 未配置"
fi

# 检查钉钉配置
if grep -q "^DINGTALK_WEBHOOK=" .env 2>/dev/null; then
    echo -e "${GREEN}[成功]${NC} DINGTALK_WEBHOOK 已配置"
else
    echo -e "${YELLOW}[提示]${NC} DINGTALK_WEBHOOK 未配置，钉钉通知功能不可用"
fi

# 检查 AI 配置
if grep -q "^AI_PROVIDER=" .env 2>/dev/null; then
    AI_PROVIDER=$(grep "^AI_PROVIDER=" .env | cut -d'=' -f2)
    echo -e "${GREEN}[成功]${NC} AI_PROVIDER 已配置: $AI_PROVIDER"
    
    if [ "$AI_PROVIDER" = "openai" ]; then
        if grep -q "^OPENAI_API_KEY=" .env 2>/dev/null; then
            echo -e "${GREEN}[成功]${NC} OPENAI_API_KEY 已配置"
        else
            echo -e "${RED}[错误]${NC} AI_PROVIDER=openai 但 OPENAI_API_KEY 未配置"
        fi
    elif [ "$AI_PROVIDER" = "ollama" ]; then
        if grep -q "^OLLAMA_HOST=" .env 2>/dev/null; then
            echo -e "${GREEN}[成功]${NC} OLLAMA_HOST 已配置"
        else
            echo -e "${YELLOW}[警告]${NC} AI_PROVIDER=ollama 但 OLLAMA_HOST 未配置"
        fi
    fi
else
    echo -e "${YELLOW}[提示]${NC} AI_PROVIDER 未配置，AI 功能不可用"
fi

# 检查 Docker
echo ""
echo "检查 Docker 环境..."
if command -v docker &> /dev/null; then
    if docker info &> /dev/null; then
        echo -e "${GREEN}[成功]${NC} Docker 运行正常"
    else
        echo -e "${YELLOW}[警告]${NC} Docker 已安装但无法连接，请检查 Docker 服务是否运行"
    fi
else
    echo -e "${YELLOW}[提示]${NC} Docker 未安装"
fi

echo ""
echo "============================================"
echo "诊断完成"
echo "============================================"
echo ""
echo "如需更详细的诊断，请运行: go run cmd/qwq/main.go --diagnose"
