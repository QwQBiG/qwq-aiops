# qwq AIOps 平台部署指南

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [生产环境部署](#生产环境部署)
- [配置说明](#配置说明)
- [监控和维护](#监控和维护)
- [故障排查](#故障排查)

## 系统要求

### 最低配置

- CPU: 2核
- 内存: 4GB
- 磁盘: 20GB
- 操作系统: Linux (Ubuntu 20.04+, CentOS 7+, Debian 10+)
- Docker: 20.10+
- Docker Compose: 1.29+ (可选)

### 推荐配置

- CPU: 4核+
- 内存: 8GB+
- 磁盘: 50GB+ SSD
- 操作系统: Ubuntu 22.04 LTS
- Docker: 最新稳定版
- Docker Compose: 最新稳定版

## 快速开始

### 方式一：使用部署脚本（推荐）

```bash
# 1. 克隆代码
git clone https://github.com/your-org/qwq.git
cd qwq

# 2. 运行部署脚本
chmod +x deploy.sh
./deploy.sh

# 3. 访问系统
# 浏览器打开 http://localhost:8899
# 默认用户名: admin
# 默认密码: admin123
```

### 方式二：使用 Docker Compose

```bash
# 1. 克隆代码
git clone https://github.com/your-org/qwq.git
cd qwq

# 2. 启动所有服务
docker-compose up -d

# 3. 查看服务状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f qwq
```

### 方式三：手动 Docker 部署

```bash
# 1. 构建镜像
docker build -t qwq-aiops:latest .

# 2. 运行容器
docker run -d \
  --name qwq \
  --restart unless-stopped \
  -p 8899:8899 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ./data:/root/data \
  qwq-aiops:latest

# 3. 查看日志
docker logs -f qwq
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

创建 `config/app.yaml`：

```yaml
server:
  port: 8899
  mode: production

database:
  type: sqlite
  path: /root/data/qwq.db

security:
  jwt_secret: "your-secret-key-change-me"
  session_timeout: 3600

ai:
  provider: openai
  api_key: "your-openai-api-key"
  model: gpt-4

monitoring:
  enabled: true
  prometheus_port: 9090

logging:
  level: info
  file: /root/data/logs/qwq.log
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

    # 代理配置
    location / {
        proxy_pass http://localhost:8899;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持
    location /ws {
        proxy_pass http://localhost:8899;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
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

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `PORT` | 服务端口 | 8899 |
| `DB_PATH` | 数据库路径 | /root/data/qwq.db |
| `LOG_LEVEL` | 日志级别 | info |
| `JWT_SECRET` | JWT密钥 | - |
| `AI_API_KEY` | AI API密钥 | - |
| `TZ` | 时区 | Asia/Shanghai |

### 数据持久化

重要数据目录：

- `/root/data/qwq.db` - SQLite 数据库
- `/root/data/logs/` - 日志文件
- `/root/data/backups/` - 备份文件
- `/root/config/` - 配置文件

确保这些目录已挂载到宿主机：

```bash
docker run -v ./data:/root/data -v ./config:/root/config ...
```

## 监控和维护

### 健康检查

```bash
# 检查服务状态
curl http://localhost:8899/api/health

# 查看系统指标
curl http://localhost:8899/api/monitoring/metrics
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

# 测试 API 连接
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.openai.com/v1/models
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

## 安全建议

1. **修改默认密码**：首次登录后立即修改管理员密码
2. **启用 HTTPS**：生产环境必须使用 HTTPS
3. **配置防火墙**：只开放必要的端口
4. **定期备份**：设置自动备份任务
5. **更新系统**：定期更新系统和 Docker
6. **监控日志**：启用日志监控和告警
7. **限制访问**：使用 IP 白名单或 VPN

## 技术支持

- 文档：https://docs.qwq.io
- GitHub：https://github.com/your-org/qwq
- 问题反馈：https://github.com/your-org/qwq/issues
- 社区讨论：https://community.qwq.io

## 许可证

MIT License
