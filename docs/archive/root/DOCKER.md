# Docker 部署指南

## 快速开始

### 使用预构建镜像（推荐）

```bash
# 1. 拉取最新镜像
docker pull ghcr.io/your-org/qwq-aiops:latest

# 2. 创建数据目录
mkdir -p data logs backups

# 3. 创建配置文件（可选）
cat > config.json <<EOF
{
  "api_key": "your-openai-api-key",
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-3.5-turbo",
  "web_user": "admin",
  "web_password": "change-this-password"
}
EOF

# 4. 启动容器
docker run -d \
  --name qwq \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -v $(pwd)/backups:/app/backups \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -e TZ=Asia/Shanghai \
  ghcr.io/your-org/qwq-aiops:latest

# 5. 查看日志
docker logs -f qwq

# 6. 访问系统
# 浏览器打开: http://localhost:8080
```

### 使用 Docker Compose（推荐生产环境）

```bash
# 1. 克隆项目
git clone https://github.com/your-org/qwq-aiops.git
cd qwq-aiops

# 2. 修改配置
cp .env.example .env
# 编辑 .env 文件，设置必要的环境变量

# 3. 启动所有服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f qwq

# 6. 停止服务
docker-compose down

# 7. 停止并删除数据
docker-compose down -v
```

## 本地构建

### 构建镜像

```bash
# 基础构建
docker build -t qwq-aiops:local .

# 不安装 kubectl（减小镜像体积）
docker build --build-arg INSTALL_KUBECTL=false -t qwq-aiops:local .

# 多平台构建
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t qwq-aiops:local \
  --push .
```

### 构建优化

```bash
# 使用 BuildKit 缓存
DOCKER_BUILDKIT=1 docker build \
  --cache-from ghcr.io/your-org/qwq-aiops:latest \
  -t qwq-aiops:local .

# 查看构建历史
docker history qwq-aiops:local

# 分析镜像大小
docker images qwq-aiops:local
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `PORT` | 服务端口 | 8080 | 否 |
| `TZ` | 时区 | Asia/Shanghai | 否 |
| `LOG_LEVEL` | 日志级别 | info | 否 |
| `DB_TYPE` | 数据库类型 | sqlite | 否 |
| `DB_PATH` | SQLite 路径 | /app/data/qwq.db | 否 |
| `AI_PROVIDER` | AI 提供商 | openai | 是 |
| `OPENAI_API_KEY` | OpenAI Key | - | 条件 |
| `OLLAMA_HOST` | Ollama 地址 | - | 条件 |
| `JWT_SECRET` | JWT 密钥 | - | 是 |
| `ENCRYPTION_KEY` | 加密密钥 | - | 是 |

### 数据卷

| 容器路径 | 说明 | 推荐挂载 |
|---------|------|---------|
| `/app/data` | 数据库和数据文件 | `./data` |
| `/app/logs` | 日志文件 | `./logs` |
| `/app/backups` | 备份文件 | `./backups` |
| `/var/run/docker.sock` | Docker Socket | `/var/run/docker.sock:ro` |

### 端口映射

| 容器端口 | 说明 | 推荐映射 |
|---------|------|---------|
| 8080 | 主服务端口 | 8080:8080 |

## 生产环境部署

### 使用 Nginx 反向代理

```nginx
upstream qwq_backend {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name qwq.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name qwq.example.com;

    ssl_certificate /etc/letsencrypt/live/qwq.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/qwq.example.com/privkey.pem;

    location / {
        proxy_pass http://qwq_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 使用 Traefik

```yaml
version: '3.8'

services:
  qwq:
    image: ghcr.io/your-org/qwq-aiops:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.qwq.rule=Host(`qwq.example.com`)"
      - "traefik.http.routers.qwq.entrypoints=websecure"
      - "traefik.http.routers.qwq.tls.certresolver=letsencrypt"
      - "traefik.http.services.qwq.loadbalancer.server.port=8080"
```

### 集群部署

```yaml
version: '3.8'

services:
  qwq-1:
    image: ghcr.io/your-org/qwq-aiops:latest
    environment:
      - NODE_NAME=qwq-1
      - CLUSTER_ENABLED=true
      - CLUSTER_NODES=qwq-1,qwq-2,qwq-3

  qwq-2:
    image: ghcr.io/your-org/qwq-aiops:latest
    environment:
      - NODE_NAME=qwq-2
      - CLUSTER_ENABLED=true
      - CLUSTER_NODES=qwq-1,qwq-2,qwq-3

  qwq-3:
    image: ghcr.io/your-org/qwq-aiops:latest
    environment:
      - NODE_NAME=qwq-3
      - CLUSTER_ENABLED=true
      - CLUSTER_NODES=qwq-1,qwq-2,qwq-3

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - qwq-1
      - qwq-2
      - qwq-3
```

## 维护操作

### 备份数据

```bash
# 备份数据库
docker exec qwq sqlite3 /app/data/qwq.db .dump > backup.sql

# 备份整个数据目录
tar -czf qwq-backup-$(date +%Y%m%d).tar.gz data/

# 使用 Docker 卷备份
docker run --rm \
  -v qwq_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/qwq-data-backup.tar.gz /data
```

### 恢复数据

```bash
# 恢复数据库
docker exec -i qwq sqlite3 /app/data/qwq.db < backup.sql

# 恢复数据目录
tar -xzf qwq-backup-20241207.tar.gz

# 从卷恢复
docker run --rm \
  -v qwq_data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/qwq-data-backup.tar.gz --strip 1"
```

### 升级版本

```bash
# 1. 备份数据
docker-compose exec qwq sqlite3 /app/data/qwq.db .dump > backup.sql

# 2. 停止服务
docker-compose down

# 3. 拉取新版本
docker-compose pull

# 4. 启动新版本
docker-compose up -d

# 5. 查看日志
docker-compose logs -f qwq

# 6. 如果有问题，回滚
docker-compose down
docker-compose up -d --force-recreate
```

### 查看日志

```bash
# 实时日志
docker logs -f qwq

# 最近 100 行
docker logs --tail 100 qwq

# 带时间戳
docker logs -t qwq

# 特定时间范围
docker logs --since "2024-12-07T10:00:00" qwq
```

### 进入容器

```bash
# 使用 bash
docker exec -it qwq bash

# 使用 sh（如果没有 bash）
docker exec -it qwq sh

# 执行单个命令
docker exec qwq ls -la /app/data
```

## 故障排查

### 容器无法启动

```bash
# 查看容器状态
docker ps -a

# 查看详细日志
docker logs qwq

# 检查配置
docker inspect qwq

# 验证镜像
docker run --rm qwq-aiops:latest --version
```

### 性能问题

```bash
# 查看资源使用
docker stats qwq

# 查看进程
docker top qwq

# 导出性能数据
docker stats --no-stream qwq > stats.txt
```

### 网络问题

```bash
# 检查网络
docker network ls
docker network inspect qwq-network

# 测试连接
docker exec qwq curl -v http://localhost:8080/health

# 查看端口
docker port qwq
```

## 安全建议

1. **不要使用 root 用户**：镜像已配置为非 root 用户运行
2. **限制资源**：使用 `deploy.resources` 限制 CPU 和内存
3. **只读挂载**：敏感文件使用 `:ro` 只读挂载
4. **网络隔离**：使用自定义网络隔离服务
5. **定期更新**：及时更新到最新版本
6. **强密码**：修改所有默认密码
7. **HTTPS**：生产环境使用 HTTPS
8. **备份**：定期备份数据

## 常见问题

### Q: 如何修改端口？

```bash
# 方法 1: 修改端口映射
docker run -p 9000:8080 ghcr.io/your-org/qwq-aiops:latest

# 方法 2: 修改环境变量
docker run -e PORT=9000 -p 9000:9000 ghcr.io/your-org/qwq-aiops:latest
```

### Q: 如何使用本地 Ollama？

```bash
# 1. 启动 Ollama
docker run -d --name ollama -p 11434:11434 ollama/ollama

# 2. 启动 qwq 并连接
docker run -d \
  --name qwq \
  --link ollama:ollama \
  -e AI_PROVIDER=ollama \
  -e OLLAMA_HOST=http://ollama:11434 \
  ghcr.io/your-org/qwq-aiops:latest
```

### Q: 如何查看版本？

```bash
docker run --rm ghcr.io/your-org/qwq-aiops:latest --version
```

### Q: 镜像太大怎么办？

```bash
# 使用不带 kubectl 的版本
docker build --build-arg INSTALL_KUBECTL=false -t qwq-aiops:slim .

# 清理未使用的镜像
docker image prune -a
```

## 参考资源

- [官方文档](https://github.com/your-org/qwq-aiops)
- [Docker Hub](https://hub.docker.com/r/your-org/qwq-aiops)
- [GitHub Container Registry](https://github.com/your-org/qwq-aiops/pkgs/container/qwq-aiops)
- [问题反馈](https://github.com/your-org/qwq-aiops/issues)

---

**版本**: v1.0.0  
**最后更新**: 2024-12-07
