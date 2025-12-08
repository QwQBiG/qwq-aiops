#!/bin/bash
# ============================================
# qwq AIOps 平台 - 完整部署脚本
# 包含完整的检查、验证和错误处理
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 全局变量
ERROR_COUNT=0
WARNING_COUNT=0
BUILD_LOG="build.log"

# 打印标题
print_header() {
    echo ""
    echo "========================================"
    echo "  qwq AIOps 平台 - 完整部署脚本"
    echo "========================================"
    echo ""
}

# 步骤 1: 环境检查
check_environment() {
    echo -e "${BLUE}[1/7] 环境检查${NC}"
    echo ""
    
    # 检查 Docker
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}✗ Docker 未运行或未安装${NC}"
        echo "  请先安装并启动 Docker"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker 运行正常${NC}"
    
    # 检查 Docker Compose
    if docker compose version >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Docker Compose V2 可用${NC}"
    elif docker-compose --version >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠ 使用 Docker Compose V1，建议升级到 V2${NC}"
        WARNING_COUNT=$((WARNING_COUNT + 1))
    else
        echo -e "${RED}✗ Docker Compose 未安装${NC}"
        exit 1
    fi
    
    # 检查必要文件
    echo ""
    echo "检查必要文件..."
    local required_files=("docker-compose.yml" "Dockerfile" "go.mod" "go.sum")
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            echo -e "${RED}✗ $file 不存在${NC}"
            ERROR_COUNT=$((ERROR_COUNT + 1))
        else
            echo -e "${GREEN}✓ $file 存在${NC}"
        fi
    done
    
    # 检查前端目录
    if [ ! -d "frontend" ] || [ ! -f "frontend/package.json" ]; then
        echo -e "${RED}✗ frontend 目录或 package.json 不存在${NC}"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    else
        echo -e "${GREEN}✓ frontend 目录完整${NC}"
    fi
    
    # 检查后端入口
    if [ ! -f "cmd/qwq/main.go" ]; then
        echo -e "${RED}✗ cmd/qwq/main.go 不存在${NC}"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    else
        echo -e "${GREEN}✓ 后端入口文件存在${NC}"
    fi
    
    if [ $ERROR_COUNT -gt 0 ]; then
        echo ""
        echo -e "${RED}发现 $ERROR_COUNT 个错误，无法继续部署${NC}"
        exit 1
    fi
    
    # 检查端口占用
    echo ""
    echo "检查端口占用..."
    local ports=(8081 3308 6380 9091 3000)
    local port_conflict=false
    
    for port in "${ports[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1 || netstat -tuln 2>/dev/null | grep -q ":$port "; then
            echo -e "${YELLOW}⚠ 端口 $port 已被占用${NC}"
            port_conflict=true
            WARNING_COUNT=$((WARNING_COUNT + 1))
        else
            echo -e "${GREEN}✓ 端口 $port 可用${NC}"
        fi
    done
    
    if [ "$port_conflict" = true ]; then
        echo ""
        echo -e "${YELLOW}部分端口被占用，可能导致服务启动失败${NC}"
        read -p "是否继续？[y/N]: " continue_deploy
        if [[ ! $continue_deploy =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # 检查磁盘空间
    echo ""
    echo "检查磁盘空间..."
    local available=$(df -BG . | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$available" -lt 10 ]; then
        echo -e "${YELLOW}⚠ 可用磁盘空间不足 10GB (当前: ${available}GB)${NC}"
        WARNING_COUNT=$((WARNING_COUNT + 1))
    else
        echo -e "${GREEN}✓ 磁盘空间充足 (${available}GB)${NC}"
    fi
    
    # 测试网络连接
    echo ""
    echo "测试网络连接..."
    if curl -s -I --connect-timeout 5 https://goproxy.cn >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Go 代理可访问${NC}"
    else
        echo -e "${YELLOW}⚠ Go 代理访问失败，构建可能较慢${NC}"
        WARNING_COUNT=$((WARNING_COUNT + 1))
    fi
    
    if curl -s -I --connect-timeout 5 https://registry.npmmirror.com >/dev/null 2>&1; then
        echo -e "${GREEN}✓ npm 镜像可访问${NC}"
    else
        echo -e "${YELLOW}⚠ npm 镜像访问失败，构建可能较慢${NC}"
        WARNING_COUNT=$((WARNING_COUNT + 1))
    fi
    
    echo ""
    echo -e "${GREEN}环境检查完成${NC}"
    if [ $WARNING_COUNT -gt 0 ]; then
        echo -e "${YELLOW}发现 $WARNING_COUNT 个警告${NC}"
    fi
    echo ""
}

# 步骤 2: 配置检查和创建
check_configuration() {
    echo -e "${BLUE}[2/7] 配置检查${NC}"
    echo ""
    
    # 创建必要目录
    echo "创建必要目录..."
    for dir in config data logs backups; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            echo -e "${GREEN}✓ 创建 $dir 目录${NC}"
        else
            echo -e "${GREEN}✓ $dir 目录已存在${NC}"
        fi
    done
    
    # 检查并创建 .env 文件
    echo ""
    if [ ! -f .env ]; then
        echo -e "${YELLOW}⚠ .env 文件不存在，正在创建...${NC}"
        cp .env.example .env
        echo -e "${GREEN}✓ .env 文件已创建${NC}"
    else
        echo -e "${GREEN}✓ .env 文件已存在${NC}"
    fi
    
    # 验证 AI 配置
    echo ""
    echo "验证 AI 配置..."
    if grep -q "^AI_PROVIDER=" .env && ! grep -q "^AI_PROVIDER=$" .env && ! grep -q "^#AI_PROVIDER=" .env; then
        local ai_provider=$(grep "^AI_PROVIDER=" .env | cut -d'=' -f2)
        echo -e "${GREEN}✓ AI_PROVIDER 已配置: $ai_provider${NC}"
        
        # 验证具体配置
        if [ "$ai_provider" = "openai" ]; then
            if grep -q "^OPENAI_API_KEY=" .env && ! grep -q "^OPENAI_API_KEY=$" .env; then
                echo -e "${GREEN}✓ OpenAI API Key 已配置${NC}"
            else
                echo -e "${RED}✗ OpenAI API Key 未配置${NC}"
                echo "  请编辑 .env 文件，设置 OPENAI_API_KEY"
                ERROR_COUNT=$((ERROR_COUNT + 1))
            fi
        elif [ "$ai_provider" = "ollama" ]; then
            echo -e "${GREEN}✓ Ollama 配置已设置${NC}"
            # 检查 OLLAMA_HOST 配置
            if grep -q "^OLLAMA_HOST=" .env && ! grep -q "^OLLAMA_HOST=$" .env; then
                local ollama_host=$(grep "^OLLAMA_HOST=" .env | cut -d'=' -f2)
                echo -e "${GREEN}✓ Ollama 地址: $ollama_host${NC}"
            else
                echo -e "${YELLOW}⚠ OLLAMA_HOST 未配置，请在 .env 中设置${NC}"
                echo "  Docker 环境建议使用: http://宿主机IP:11434"
                WARNING_COUNT=$((WARNING_COUNT + 1))
            fi
        fi
    else
        echo -e "${YELLOW}⚠ AI_PROVIDER 未配置${NC}"
        echo ""
        echo "qwq 是 AI 驱动的平台，需要配置 AI 服务才能正常运行"
        echo ""
        echo "请选择 AI 服务类型："
        echo "  1) OpenAI API（需要 API Key）"
        echo "  2) Ollama 本地模型（免费）"
        echo "  3) 跳过配置（稍后手动配置）"
        echo ""
        read -p "请输入选项 [1-3]: " ai_choice
        
        case $ai_choice in
            1)
                read -p "请输入 OpenAI API Key: " api_key
                if [ -n "$api_key" ]; then
                    sed -i "s/^# AI_PROVIDER=.*/AI_PROVIDER=openai/" .env
                    sed -i "s/^# OPENAI_API_KEY=.*/OPENAI_API_KEY=$api_key/" .env
                    sed -i "s/^# OPENAI_BASE_URL=.*/OPENAI_BASE_URL=https:\/\/api.openai.com\/v1/" .env
                    sed -i "s/^# OPENAI_MODEL=.*/OPENAI_MODEL=gpt-3.5-turbo/" .env
                    echo -e "${GREEN}✓ OpenAI 配置已保存${NC}"
                fi
                ;;
            2)
                sed -i "s/^# AI_PROVIDER=.*/AI_PROVIDER=ollama/" .env
                sed -i "s/^# OLLAMA_HOST=.*/OLLAMA_HOST=http:\/\/host.docker.internal:11434/" .env
                sed -i "s/^# OLLAMA_MODEL=.*/OLLAMA_MODEL=qwen2.5:7b/" .env
                echo -e "${GREEN}✓ Ollama 配置已保存${NC}"
                ;;
            3)
                echo -e "${YELLOW}⚠ 跳过 AI 配置${NC}"
                echo "  服务启动后需要手动配置 .env 文件"
                WARNING_COUNT=$((WARNING_COUNT + 1))
                ;;
        esac
    fi
    
    # 创建 Prometheus 配置
    echo ""
    if [ ! -f config/prometheus.yml ]; then
        echo "创建 Prometheus 配置..."
        cat > config/prometheus.yml <<'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'qwq'
    static_configs:
      - targets: ['qwq:8080']
    metrics_path: '/metrics'
EOF
        echo -e "${GREEN}✓ prometheus.yml 已创建${NC}"
    else
        echo -e "${GREEN}✓ prometheus.yml 已存在${NC}"
    fi
    
    # 创建 MySQL 配置
    if [ ! -f config/mysql.cnf ]; then
        echo "创建 MySQL 配置..."
        cat > config/mysql.cnf <<'EOF'
[mysqld]
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
default-authentication-plugin=mysql_native_password

[client]
default-character-set=utf8mb4
EOF
        echo -e "${GREEN}✓ mysql.cnf 已创建${NC}"
    else
        echo -e "${GREEN}✓ mysql.cnf 已存在${NC}"
    fi
    
    if [ $ERROR_COUNT -gt 0 ]; then
        echo ""
        echo -e "${RED}配置检查失败，请修复错误后重试${NC}"
        exit 1
    fi
    
    echo ""
    echo -e "${GREEN}配置检查完成${NC}"
    echo ""
}

# 步骤 3: 停止现有服务
stop_existing_services() {
    echo -e "${BLUE}[3/7] 停止现有服务${NC}"
    echo ""
    
    if docker compose ps -q 2>/dev/null | grep -q .; then
        echo "停止现有容器..."
        docker compose down
        echo -e "${GREEN}✓ 现有容器已停止${NC}"
    else
        echo -e "${GREEN}✓ 没有运行中的容器${NC}"
    fi
    
    echo ""
}

# 步骤 4: 构建镜像
build_images() {
    echo -e "${BLUE}[4/7] 构建镜像${NC}"
    echo ""
    echo -e "${YELLOW}首次构建需要 6-10 分钟，请耐心等待...${NC}"
    echo "构建日志将保存到: $BUILD_LOG"
    echo ""
    
    # 构建并保存日志
    if docker compose build --no-cache 2>&1 | tee "$BUILD_LOG"; then
        echo ""
        echo -e "${GREEN}✓ 镜像构建成功${NC}"
    else
        echo ""
        echo -e "${RED}✗ 镜像构建失败${NC}"
        echo ""
        echo "构建日志已保存到: $BUILD_LOG"
        echo ""
        echo "常见问题排查："
        echo "1. 网络问题"
        echo "   - 检查网络连接"
        echo "   - 查看是否需要配置代理"
        echo ""
        echo "2. 依赖问题"
        echo "   - 查看 $BUILD_LOG 中的错误信息"
        echo "   - 检查 go.mod 和 package.json"
        echo ""
        echo "3. 磁盘空间"
        echo "   - 运行: df -h"
        echo "   - 清理: docker system prune -a"
        echo ""
        exit 1
    fi
    
    echo ""
}

# 步骤 5: 启动服务
start_services() {
    echo -e "${BLUE}[5/7] 启动服务${NC}"
    echo ""
    
    if docker compose up -d; then
        echo ""
        echo -e "${GREEN}✓ 服务启动成功${NC}"
    else
        echo ""
        echo -e "${RED}✗ 服务启动失败${NC}"
        echo ""
        echo "查看错误日志:"
        echo "  docker compose logs"
        exit 1
    fi
    
    echo ""
}

# 步骤 6: 健康检查
health_check() {
    echo -e "${BLUE}[6/7] 健康检查${NC}"
    echo ""
    
    echo "等待服务启动..."
    local max_wait=60
    local waited=0
    local health_ok=false
    
    while [ $waited -lt $max_wait ]; do
        if curl -s http://localhost:8081/api/health >/dev/null 2>&1; then
            health_ok=true
            echo -e "${GREEN}✓ 服务启动成功 (等待了 ${waited} 秒)${NC}"
            break
        fi
        sleep 5
        waited=$((waited + 5))
        echo "  等待中... (${waited}/${max_wait}秒)"
    done
    
    if [ "$health_ok" = false ]; then
        echo -e "${YELLOW}⚠ 服务启动超时${NC}"
        echo ""
        echo "请检查服务状态:"
        echo "  docker compose ps"
        echo ""
        echo "查看日志:"
        echo "  docker compose logs -f qwq"
        echo ""
        WARNING_COUNT=$((WARNING_COUNT + 1))
    fi
    
    # 检查容器状态
    echo ""
    echo "检查容器状态..."
    docker compose ps
    
    echo ""
}

# 步骤 7: 显示结果
show_results() {
    echo -e "${BLUE}[7/7] 部署完成${NC}"
    echo ""
    echo "========================================"
    echo -e "${GREEN}  部署成功！${NC}"
    echo "========================================"
    echo ""
    echo "访问地址："
    echo -e "  前端界面: ${BLUE}http://localhost:8081${NC}"
    echo -e "  API 文档: ${BLUE}http://localhost:8081/api/docs${NC}"
    echo -e "  健康检查: ${BLUE}http://localhost:8081/api/health${NC}"
    echo -e "  Prometheus: ${BLUE}http://localhost:9091${NC}"
    echo -e "  Grafana: ${BLUE}http://localhost:3000${NC}"
    echo ""
    echo "默认账号："
    echo "  用户名: admin"
    echo "  密码: admin123"
    echo ""
    echo "常用命令："
    echo "  查看日志: docker compose logs -f qwq"
    echo "  查看状态: docker compose ps"
    echo "  停止服务: docker compose down"
    echo "  重启服务: docker compose restart"
    echo ""
    
    if [ $WARNING_COUNT -gt 0 ]; then
        echo -e "${YELLOW}注意: 发现 $WARNING_COUNT 个警告，请检查上面的输出${NC}"
        echo ""
    fi
    
    echo "========================================"
    echo ""
}

# 主函数
main() {
    print_header
    check_environment
    check_configuration
    stop_existing_services
    build_images
    start_services
    health_check
    show_results
}

# 执行主函数
main
