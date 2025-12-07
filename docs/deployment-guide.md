# qwq AIOps 平台部署指南

> **版本**: v1.0.0  
> **更新日期**: 2025-12-07  
> **状态**: ✅ 生产就绪

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [生产环境部署](#生产环境部署)
- [AI 配置说明](#ai-配置说明)
- [配置说明](#配置说明)
- [监控和维护](#监控和维护)
- [故障排查](#故障排查)
- [性能优化](#性能优化)

## 系统要求

### 最低配置

- **CPU**: 2核
- **内存**: 4GB
- **磁盘**: 20GB
- **操作系统**: Linux (Ubuntu 20.04+, CentOS 7+, Debian 10+) / macOS / Windows
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### 推荐配置（生产环境）

- **CPU**: 4核+
- **内存**: 8GB+
- **磁盘**: 50GB+ SSD
- **操作系统**: Ubuntu 22.04 LTS
- **Docker**: 最新稳定版
- **Docker Compose**: 最新稳定版
- **网络**: 公网 IP（用于 SSL 证书申请）

### AI 功能额外要求

如果需要使用 AI 智能运维功能，还需要：

- **云端 API**：OpenAI API Key 或硅基流动 API Key
- **本地模型**：Ollama + DeepSeek/Qwen 模型（推荐 8GB+ 内存）

## 快速开始

### 方式一：使用部署脚本（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 配置环境变量（可选）
cp .env.example .env
# 编辑 .env 文件，配置 AI 模型 API Key

# 3. 运行部署脚本
chmod +x deploy.sh
./deploy.sh

# 4. 访问系统
# 前端界面: http://localhost:8080
# API 文档: http://localhost:8080/api/docs
# 默认账号: admin / admin123
```

### 方式二：使用 Docker Compose

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 配置环境变量（可选）
cp .env.example .env

# 3. 启动所有服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f

# 6. 访问系统
# 前端界面: http://localhost:8080
# API 文档: http://localhost:8080/api/docs
```

### 方式三：手动 Docker 部署

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 构建镜像
docker build -t qwq-aiops:v1.0.0 .

# 3. 运行容器
docker run -d \
  --name qwq \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 8899:8899 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/data:/root/data \
  -e AI_PROVIDER=openai \
  -e AI_API_KEY=your-api-key-here \
  qwq-aiops:v1.0.0

# 4. 查看日志
docker logs -f qwq

# 5. 访问系统
# 前端界面: http://localhost:8080
```

### 方式四：本地开发部署

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 后端编译运行
go build -o qwq cmd/qwq/main.go
./qwq

# 3. 前端开发（新终端）
cd frontend
npm install
npm run dev

# 4. 访问系统
# 前端开发服务器: http://localhost:5173
# 后端 API: http://localhost:8899
```

## AI 配置说明

qwq 的核心优势在于 AI 智能运维功能。您可以选择云端 API 或本地模型。

### 云端 API 配置（推荐新手）

#### OpenAI API

```bash
# 在 .env 文件中配置
AI_PROVIDER=openai
AI_API_KEY=sk-xxxxxxxxxxxxx
AI_MODEL=gpt-4
AI_BASE_URL=https://api.openai.com/v1  # 可选，使用代理时配置
```

#### 硅基流动 API（国内推荐）

```bash
# 在 .env 文件中配置
AI_PROVIDER=siliconflow
AI_API_KEY=sk-xxxxxxxxxxxxx
AI_MODEL=deepseek-chat
AI_BASE_URL=https://api.siliconflow.cn/v1
```

#### Azure OpenAI

```bash
# 在 .env 文件中配置
AI_PROVIDER=azure
AI_API_KEY=your-azure-key
AI_MODEL=gpt-4
AI_BASE_URL=https://your-resource.openai.azure.com
AI_API_VERSION=2024-02-15-preview
```

### 本地模型配置（推荐企业）

#### 使用 Ollama（完全私有化）

```bash
# 1. 安装 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. 下载模型（推荐 DeepSeek）
ollama pull deepseek-coder:6.7b
# 或者使用 Qwen
ollama pull qwen2.5:7b

# 3. 启动 Ollama 服务
ollama serve

# 4. 配置 qwq
# 在 .env 文件中配置
AI_PROVIDER=ollama
AI_BASE_URL=http://localhost:11434
AI_MODEL=deepseek-coder:6.7b
```

#### 使用本地 API

```bash
# 如果您有自己部署的服务
AI_PROVIDER=openai
AI_BASE_URL=http://your-server:8000/v1
AI_API_KEY=your-local-key
AI_MODEL=deepseek-chat
```

### AI 功能说明

配置完成后，您可以使用以下 AI 功能：

1. **自然语言运维**：通过对话完成运维任务
2. **智能应用推荐**：根据场景推荐应用组合
3. **架构优化建议**：分析 Docker Compose 配置并提供优化建议
4. **SQL 查询优化**：分析慢查询并提供索引建议
5. **智能告警降噪**：减少告警风暴
6. **容量规划建议**：基于历史数据预测资源需求

### 测试 AI 配置

```bash
# 启动服务后，测试 AI 功能
curl -X POST http://localhost:8899/api/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "帮我查看系统负载"
  }'
```

## 生产环境部署

### 1. 环境准备

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 启用 Docker 服务
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. 配置文件准备

创建配置目录：

```bash
mkdir -p config data
```

创建 `.env` 配置文件：

```bash
# 服务配置
PORT=8899
MODE=production
TZ=Asia/Shanghai

# 数据库配置
DB_TYPE=sqlite
DB_PATH=/root/data/qwq.db

# 安全配置
JWT_SECRET=your-secret-key-change-me-in-production
SESSION_TIMEOUT=3600

# AI 配置（必须配置）
AI_PROVIDER=openai
AI_API_KEY=your-openai-api-key
AI_MODEL=gpt-4
AI_BASE_URL=https://api.openai.com/v1

# 监控配置
MONITORING_ENABLED=true
PROMETHEUS_PORT=9090

# 日志配置
LOG_LEVEL=info
LOG_FILE=/root/data/logs/qwq.log

# 集群配置（可选）
CLUSTER_ENABLED=false
CLUSTER_NODES=node1:8899,node2:8899

# 备份配置（可选）
BACKUP_ENABLED=true
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30
```

或者使用 YAML 配置文件 `config/app.yaml`：

```yaml
server:
  port: 8899
  mode: production
  frontend_port: 8080

database:
  type: sqlite
  path: /root/data/qwq.db
  # 或使用 PostgreSQL
  # type: postgres
  # host: localhost
  # port: 5432
  # database: qwq
  # username: qwq
  # password: your-password

security:
  jwt_secret: "your-secret-key-change-me"
  session_timeout: 3600
  enable_rbac: true
  enable_audit: true

ai:
  provider: openai
  api_key: "your-openai-api-key"
  model: gpt-4
  base_url: "https://api.openai.com/v1"
  timeout: 60
  max_tokens: 2000

monitoring:
  enabled: true
  prometheus_port: 9090
  metrics_interval: 60
  alert_enabled: true

logging:
  level: info
  file: /root/data/logs/qwq.log
  max_size: 100
  max_backups: 10
  max_age: 30

cluster:
  enabled: false
  node_id: node1
  nodes:
    - node1:8899
    - node2:8899
  health_check_interval: 10

backup:
  enabled: true
  schedule: "0 2 * * *"
  retention_days: 30
  storage:
    type: local
    path: /root/data/backups
```

### 3. 使用 Docker Compose 部署

编辑 `docker-compose.yml` 根据需要启用或禁用服务，然后：

```bash
# 启动服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 重启服务
docker-compose restart
```

### 4. 配置反向代理（Nginx）

创建 `/etc/nginx/sites-available/qwq`：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL 证书配置
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 前端静态资源
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 后端 API
    location /api {
        proxy_pass http://localhost:8899;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持（实时日志、终端等）
    location /ws {
        proxy_pass http://localhost:8899;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 文件上传大小限制
    client_max_body_size 100M;
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/qwq /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 5. 配置 SSL 证书（Let's Encrypt）

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx -y

# 申请证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 | 必填 |
|--------|------|--------|------|
| `PORT` | 后端 API 端口 | 8899 | 否 |
| `FRONTEND_PORT` | 前端服务端口 | 8080 | 否 |
| `MODE` | 运行模式 (development/production) | production | 否 |
| `DB_TYPE` | 数据库类型 (sqlite/postgres) | sqlite | 否 |
| `DB_PATH` | SQLite 数据库路径 | /root/data/qwq.db | 否 |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | info | 否 |
| `JWT_SECRET` | JWT 密钥（生产环境必须修改） | - | 是 |
| `AI_PROVIDER` | AI 提供商 (openai/ollama/azure) | openai | 是 |
| `AI_API_KEY` | AI API 密钥 | - | 是* |
| `AI_MODEL` | AI 模型名称 | gpt-4 | 否 |
| `AI_BASE_URL` | AI API 地址 | - | 否 |
| `TZ` | 时区 | Asia/Shanghai | 否 |
| `CLUSTER_ENABLED` | 是否启用集群 | false | 否 |
| `MONITORING_ENABLED` | 是否启用监控 | true | 否 |
| `BACKUP_ENABLED` | 是否启用自动备份 | true | 否 |

> **注意**：使用 Ollama 本地模型时，`AI_API_KEY` 可以不填

### 数据持久化

重要数据目录：

- `/root/data/qwq.db` - SQLite 数据库（存储所有业务数据）
- `/root/data/logs/` - 日志文件
- `/root/data/backups/` - 备份文件
- `/root/config/` - 配置文件
- `/var/run/docker.sock` - Docker Socket（容器管理必需）

确保这些目录已挂载到宿主机：

```bash
docker run \
  -v $(pwd)/data:/root/data \
  -v $(pwd)/config:/root/config \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ...
```

### 端口说明

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | 前端 Web UI | 用户访问的主界面 |
| 8899 | 后端 API | RESTful API 服务 |
| 9090 | Prometheus | 监控指标（可选） |
| 3000 | Grafana | 监控面板（可选） |

## 监控和维护

### 健康检查

```bash
# 检查后端服务状态
curl http://localhost:8899/api/health

# 检查前端服务状态
curl http://localhost:8080

# 查看系统指标
curl http://localhost:8899/api/monitoring/metrics

# 查看 AI 服务状态
curl http://localhost:8899/api/ai/status

# 完整的健康检查
curl http://localhost:8899/api/health/full
```

预期响应：

```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime": "2h30m15s",
  "services": {
    "database": "healthy",
    "docker": "healthy",
    "ai": "healthy",
    "monitoring": "healthy"
  }
}
```

### 日志管理

```bash
# 查看实时日志
docker logs -f qwq

# 查看最近100行日志
docker logs --tail 100 qwq

# 导出日志
docker logs qwq > qwq.log
```

### 备份和恢复

#### 备份

```bash
# 停止服务
docker-compose stop qwq

# 备份数据
tar -czf qwq-backup-$(date +%Y%m%d).tar.gz data/

# 启动服务
docker-compose start qwq
```

#### 恢复

```bash
# 停止服务
docker-compose stop qwq

# 恢复数据
tar -xzf qwq-backup-20240101.tar.gz

# 启动服务
docker-compose start qwq
```

### 更新升级

```bash
# 拉取最新代码
git pull

# 重新构建镜像
docker-compose build

# 重启服务（零停机）
docker-compose up -d --no-deps --build qwq
```

## 故障排查

### 常见问题

#### 1. 容器无法启动

```bash
# 查看容器日志
docker logs qwq

# 检查端口占用
sudo netstat -tlnp | grep 8899

# 检查 Docker 状态
sudo systemctl status docker
```

#### 2. 无法访问 Docker API

确保 Docker socket 已正确挂载：

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock ...
```

#### 3. 数据库连接失败

检查数据库文件权限：

```bash
ls -la data/qwq.db
chmod 644 data/qwq.db
```

#### 4. AI 功能不可用

检查 AI API 配置：

```bash
# 查看环境变量
docker exec qwq env | grep AI

# 检查 AI 服务状态
curl http://localhost:8899/api/ai/status

# 测试 OpenAI API 连接
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.openai.com/v1/models

# 测试 Ollama 连接（如果使用本地模型）
curl http://localhost:11434/api/tags
```

常见 AI 问题：

- **API Key 无效**：检查 `.env` 文件中的 `AI_API_KEY` 是否正确
- **网络连接失败**：检查是否需要配置代理，设置 `AI_BASE_URL`
- **模型不存在**：确认 `AI_MODEL` 配置的模型名称正确
- **Ollama 无法连接**：确保 Ollama 服务已启动，端口 11434 可访问

#### 5. 前端无法访问

```bash
# 检查前端服务是否运行
curl http://localhost:8080

# 检查 Docker 容器状态
docker ps | grep qwq

# 查看前端日志
docker logs qwq | grep frontend

# 检查端口占用
netstat -tlnp | grep 8080
```

#### 6. 容器管理功能异常

```bash
# 检查 Docker Socket 挂载
docker inspect qwq | grep docker.sock

# 测试 Docker API 访问
docker exec qwq docker ps

# 检查 Docker 权限
ls -la /var/run/docker.sock
```

### 性能优化

#### 1. 数据库优化

```bash
# 定期清理日志
docker exec qwq sqlite3 /root/data/qwq.db "DELETE FROM logs WHERE created_at < datetime('now', '-30 days');"

# 优化数据库
docker exec qwq sqlite3 /root/data/qwq.db "VACUUM;"
```

#### 2. 资源限制

在 `docker-compose.yml` 中添加资源限制：

```yaml
services:
  qwq:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

#### 3. 日志轮转

配置 Docker 日志驱动：

```yaml
services:
  qwq:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 性能优化

### 系统级优化

```bash
# 1. 调整文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 2. 优化内核参数
cat >> /etc/sysctl.conf <<EOF
net.core.somaxconn = 1024
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.ip_local_port_range = 10000 65000
EOF
sysctl -p

# 3. 启用 Docker 日志轮转
cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF
systemctl restart docker
```

### 应用级优化

在 `docker-compose.yml` 中配置资源限制：

```yaml
services:
  qwq:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
    environment:
      - GOMAXPROCS=4
      - GOMEMLIMIT=6GiB
```

### 数据库优化

```bash
# SQLite 优化
docker exec qwq sqlite3 /root/data/qwq.db <<EOF
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=10000;
PRAGMA temp_store=MEMORY;
EOF

# 定期清理旧数据
docker exec qwq sqlite3 /root/data/qwq.db "DELETE FROM logs WHERE created_at < datetime('now', '-30 days');"

# 优化数据库
docker exec qwq sqlite3 /root/data/qwq.db "VACUUM;"
```

### 缓存优化

如果使用 Redis 缓存：

```yaml
services:
  redis:
    image: redis:7-alpine
    command: redis-server --maxmemory 2gb --maxmemory-policy allkeys-lru
    volumes:
      - redis-data:/data
```

### 监控性能指标

```bash
# 查看容器资源使用
docker stats qwq

# 查看详细性能指标
curl http://localhost:8899/api/monitoring/metrics | jq

# 查看 API 响应时间
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8899/api/health
```

创建 `curl-format.txt`：

```
time_namelookup:  %{time_namelookup}\n
time_connect:  %{time_connect}\n
time_appconnect:  %{time_appconnect}\n
time_pretransfer:  %{time_pretransfer}\n
time_redirect:  %{time_redirect}\n
time_starttransfer:  %{time_starttransfer}\n
----------\n
time_total:  %{time_total}\n
```

## 安全建议

### 基础安全

1. **修改默认密码**：首次登录后立即修改管理员密码
2. **启用 HTTPS**：生产环境必须使用 HTTPS
3. **配置防火墙**：只开放必要的端口（80, 443, 8080, 8899）
4. **定期备份**：设置自动备份任务，异地存储
5. **更新系统**：定期更新系统和 Docker
6. **监控日志**：启用日志监控和告警
7. **限制访问**：使用 IP 白名单或 VPN

### 高级安全

```bash
# 1. 配置防火墙规则
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 22/tcp
ufw enable

# 2. 启用 fail2ban
apt install fail2ban -y
systemctl enable fail2ban
systemctl start fail2ban

# 3. 配置 Docker 安全
# 限制容器权限
docker run --security-opt=no-new-privileges:true ...

# 4. 定期安全扫描
docker scan qwq-aiops:v1.0.0
```

### 数据安全

```bash
# 1. 加密敏感配置
# 使用 Docker Secrets 或环境变量加密工具

# 2. 定期备份
# 设置自动备份脚本
cat > /etc/cron.daily/qwq-backup <<'EOF'
#!/bin/bash
docker exec qwq /usr/local/bin/backup.sh
EOF
chmod +x /etc/cron.daily/qwq-backup

# 3. 备份验证
# 定期测试备份恢复流程
```

## 集群部署（高可用）

### 架构说明

qwq 支持多节点集群部署，提供高可用性和负载均衡。

```
                    ┌─────────────┐
                    │   Nginx LB  │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
   │ Node 1  │       │ Node 2  │       │ Node 3  │
   │ qwq:8899│       │ qwq:8899│       │ qwq:8899│
   └────┬────┘       └────┬────┘       └────┬────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    ┌──────▼──────┐
                    │  PostgreSQL │
                    │   (Shared)  │
                    └─────────────┘
```

### 配置步骤

1. **准备共享数据库**

```bash
# 使用 PostgreSQL 作为共享数据库
docker run -d \
  --name postgres \
  -e POSTGRES_DB=qwq \
  -e POSTGRES_USER=qwq \
  -e POSTGRES_PASSWORD=your-password \
  -p 5432:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine
```

2. **配置各节点**

在每个节点上配置 `.env`：

```bash
# Node 1
CLUSTER_ENABLED=true
CLUSTER_NODE_ID=node1
CLUSTER_NODES=node1:8899,node2:8899,node3:8899
DB_TYPE=postgres
DB_HOST=postgres-server
DB_PORT=5432
DB_NAME=qwq
DB_USER=qwq
DB_PASSWORD=your-password

# Node 2, Node 3 类似，只需修改 CLUSTER_NODE_ID
```

3. **配置负载均衡**

Nginx 配置：

```nginx
upstream qwq_backend {
    least_conn;
    server node1:8899 max_fails=3 fail_timeout=30s;
    server node2:8899 max_fails=3 fail_timeout=30s;
    server node3:8899 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name your-domain.com;

    location /api {
        proxy_pass http://qwq_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_next_upstream error timeout http_502 http_503 http_504;
    }
}
```

## 常见部署场景

### 场景 1：单机开发环境

```bash
# 最简单的部署方式
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops
docker-compose up -d
```

### 场景 2：小型团队（单服务器）

```bash
# 使用 Docker Compose + Nginx + SSL
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops
cp .env.example .env
# 编辑 .env 配置 AI API
./deploy.sh
# 配置 Nginx 反向代理和 SSL 证书
```

### 场景 3：中型企业（集群部署）

```bash
# 3 节点集群 + PostgreSQL + Redis
# 参考上面的集群部署章节
```

### 场景 4：大型企业（Kubernetes）

```bash
# 使用 Kubernetes 部署
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

## 技术支持

### 文档资源

- **用户手册**：[docs/user-manual.md](user-manual.md)
- **故障排查**：[docs/troubleshooting-guide.md](troubleshooting-guide.md)
- **API 文档**：http://localhost:8080/api/docs
- **项目总结**：[docs/project-completion-summary.md](project-completion-summary.md)

### 社区支持

- **GitHub 仓库**：https://github.com/QwQBiG/qwq-aiops
- **问题反馈**：https://github.com/QwQBiG/qwq-aiops/issues
- **功能建议**：https://github.com/QwQBiG/qwq-aiops/discussions

### 版本信息

- **当前版本**：v1.0.0
- **发布日期**：2025-12-07
- **更新日志**：[docs/release-notes-v1.0.md](release-notes-v1.0.md)

## 许可证

MIT License. Copyright (c) 2025 qwqBig.

---

**部署愉快！如有问题，欢迎提交 Issue。** 🚀
