# qwq AIOps 平台故障排查指南

## 目录

1. [常见问题](#常见问题)
2. [启动问题](#启动问题)
3. [容器管理问题](#容器管理问题)
4. [数据库问题](#数据库问题)
5. [网络问题](#网络问题)
6. [性能问题](#性能问题)
7. [AI 服务问题](#ai-服务问题)
8. [日志分析](#日志分析)
9. [诊断工具](#诊断工具)

## 常见问题

### Q1: 服务无法启动

**症状**: 运行 `./qwq` 或 `docker-compose up` 后服务无法启动

**可能原因**:
1. 端口被占用
2. 数据库连接失败
3. Docker 守护进程未运行
4. 配置文件错误

**解决方案**:

```bash
# 1. 检查端口占用
netstat -ano | findstr :8080  # Windows
lsof -i :8080                 # Linux/macOS

# 2. 检查 Docker 状态
docker ps
docker info

# 3. 查看日志
docker-compose logs qwq

# 4. 验证配置
cat .env
```

**详细步骤**:

1. **端口冲突**:
   ```bash
   # 修改 .env 文件中的端口
   PORT=8081
   
   # 或停止占用端口的进程
   # Windows
   taskkill /PID <PID> /F
   # Linux/macOS
   kill -9 <PID>
   ```

2. **Docker 未运行**:
   ```bash
   # Windows: 启动 Docker Desktop
   # Linux: 启动 Docker 服务
   sudo systemctl start docker
   sudo systemctl enable docker
   ```

3. **数据库连接失败**:
   ```bash
   # 检查数据库配置
   echo $DB_TYPE
   echo $DB_PATH
   
   # 确保数据目录存在
   mkdir -p ./data
   chmod 755 ./data
   ```

### Q2: 前端页面无法访问

**症状**: 浏览器访问 http://localhost:8080 显示无法连接

**可能原因**:
1. 服务未启动
2. 端口配置错误
3. 防火墙阻止
4. 前端构建失败

**解决方案**:

```bash
# 1. 检查服务状态
docker-compose ps
curl http://localhost:8080/health

# 2. 检查防火墙
# Windows
netsh advfirewall firewall add rule name="qwq" dir=in action=allow protocol=TCP localport=8080

# Linux
sudo ufw allow 8080/tcp

# 3. 重新构建前端
cd frontend
npm install
npm run build

# 4. 查看 Nginx 日志
docker-compose logs nginx
```

### Q3: AI 功能不可用

**症状**: AI 对话无响应或报错

**可能原因**:
1. API Key 未配置
2. Ollama 服务未运行
3. 网络连接问题
4. API 配额用尽

**解决方案**:

```bash
# 1. 检查 AI 配置
echo $AI_PROVIDER
echo $OPENAI_API_KEY
echo $OLLAMA_HOST

# 2. 测试 OpenAI 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# 3. 测试 Ollama 连接
curl http://localhost:11434/api/tags

# 4. 查看 AI 服务日志
docker-compose logs qwq | grep -i "ai\|llm\|openai\|ollama"
```

**配置 Ollama**:
```bash
# 1. 安装 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. 启动 Ollama 服务
ollama serve

# 3. 下载模型
ollama pull llama2

# 4. 配置环境变量
AI_PROVIDER=ollama
OLLAMA_HOST=http://localhost:11434
```

## 启动问题

### 问题: 数据库初始化失败

**错误信息**:
```
Error: failed to initialize database: unable to open database file
```

**解决方案**:

```bash
# 1. 检查数据目录权限
ls -la ./data
chmod 755 ./data

# 2. 检查磁盘空间
df -h

# 3. 手动创建数据库
sqlite3 ./data/qwq.db "VACUUM;"

# 4. 重新启动服务
docker-compose restart qwq
```

### 问题: 依赖服务未就绪

**错误信息**:
```
Error: dial tcp: connect: connection refused
```

**解决方案**:

```bash
# 1. 检查所有服务状态
docker-compose ps

# 2. 按顺序启动服务
docker-compose up -d mysql
sleep 10
docker-compose up -d redis
sleep 5
docker-compose up -d qwq

# 3. 使用健康检查
docker-compose up --wait
```

## 容器管理问题

### 问题: 无法连接 Docker 守护进程

**错误信息**:
```
Error: Cannot connect to the Docker daemon
```

**解决方案**:

```bash
# 1. 检查 Docker 服务状态
# Windows: 确保 Docker Desktop 正在运行
# Linux:
sudo systemctl status docker
sudo systemctl start docker

# 2. 检查 Docker Socket
ls -la /var/run/docker.sock

# 3. 添加用户到 docker 组
sudo usermod -aG docker $USER
newgrp docker

# 4. 验证连接
docker ps
docker info
```

### 问题: 容器部署失败

**错误信息**:
```
Error: failed to create container: port already allocated
```

**解决方案**:

```bash
# 1. 查找占用端口的容器
docker ps -a | grep <port>

# 2. 停止冲突的容器
docker stop <container_id>
docker rm <container_id>

# 3. 或修改端口映射
# 在应用配置中更改端口

# 4. 清理未使用的容器
docker container prune
```

### 问题: 容器自愈不工作

**症状**: 容器停止后没有自动重启

**解决方案**:

```bash
# 1. 检查自愈服务状态
curl http://localhost:8080/api/v1/containers/self-healing/status

# 2. 查看自愈日志
docker-compose logs qwq | grep -i "self-healing\|health-check"

# 3. 手动触发健康检查
curl -X POST http://localhost:8080/api/v1/containers/health-check

# 4. 验证容器重启策略
docker inspect <container_id> | grep -i restart
```

## 数据库问题

### 问题: 数据库连接超时

**错误信息**:
```
Error: dial tcp: i/o timeout
```

**解决方案**:

```bash
# 1. 检查数据库服务状态
docker-compose ps mysql

# 2. 测试数据库连接
mysql -h localhost -u root -p

# 3. 检查网络连接
ping <database_host>
telnet <database_host> 3306

# 4. 增加连接超时时间
# 在 .env 中添加
DB_TIMEOUT=30s
```

### 问题: 查询性能慢

**症状**: SQL 查询执行时间过长

**解决方案**:

```bash
# 1. 分析慢查询
# 在数据库管理界面执行
EXPLAIN SELECT * FROM containers WHERE user_id = 1;

# 2. 添加索引
CREATE INDEX idx_containers_user_id ON containers(user_id);

# 3. 优化查询
# 使用 AI 查询优化功能
curl -X POST http://localhost:8080/api/v1/database/connections/1/optimize \
  -d '{"sql": "SELECT * FROM containers WHERE user_id = 1"}'

# 4. 清理数据
# 删除旧数据或归档
```

### 问题: 数据库锁定

**错误信息**:
```
Error: database is locked
```

**解决方案**:

```bash
# SQLite 特有问题
# 1. 关闭所有连接
docker-compose restart qwq

# 2. 检查是否有其他进程访问数据库
lsof ./data/qwq.db

# 3. 考虑迁移到 PostgreSQL
# 修改 .env
DB_TYPE=postgresql
DB_HOST=localhost
DB_PORT=5432
```

## 网络问题

### 问题: API 请求超时

**症状**: API 调用长时间无响应

**解决方案**:

```bash
# 1. 检查网络连接
ping <server_ip>
curl -v http://localhost:8080/health

# 2. 检查服务负载
docker stats qwq

# 3. 增加超时时间
# 在客户端配置中
timeout: 60000  # 60 秒

# 4. 查看慢请求日志
docker-compose logs qwq | grep -i "slow\|timeout"
```

### 问题: CORS 错误

**错误信息**:
```
Access to XMLHttpRequest has been blocked by CORS policy
```

**解决方案**:

```bash
# 1. 检查 CORS 配置
# 在 internal/gateway/server.go 中

# 2. 添加允许的源
# 修改 CORS 配置
AllowOrigins: []string{"http://localhost:3000", "https://yourdomain.com"}

# 3. 临时解决（开发环境）
AllowOrigins: []string{"*"}

# 4. 重启服务
docker-compose restart qwq
```

### 问题: SSL 证书错误

**错误信息**:
```
Error: x509: certificate signed by unknown authority
```

**解决方案**:

```bash
# 1. 检查证书有效性
openssl s_client -connect yourdomain.com:443

# 2. 更新证书
curl -X POST http://localhost:8080/api/v1/ssl/certs/1/renew

# 3. 验证证书配置
curl -X GET http://localhost:8080/api/v1/ssl/certs/1

# 4. 手动申请证书
certbot certonly --standalone -d yourdomain.com
```

## 性能问题

### 问题: 系统响应慢

**症状**: 页面加载缓慢，API 响应时间长

**诊断步骤**:

```bash
# 1. 检查系统资源
docker stats

# 2. 查看 CPU 使用率
top
htop

# 3. 查看内存使用
free -h
docker stats --no-stream

# 4. 查看磁盘 I/O
iostat -x 1

# 5. 分析慢请求
docker-compose logs qwq | grep -i "slow\|latency" | tail -100
```

**优化方案**:

```bash
# 1. 增加资源限制
# 在 docker-compose.yml 中
resources:
  limits:
    cpus: '2'
    memory: 4G

# 2. 启用缓存
# 在 .env 中
ENABLE_CACHE=true
CACHE_TTL=300

# 3. 优化数据库
# 添加索引
# 清理旧数据

# 4. 使用 CDN
# 配置静态资源 CDN
```

### 问题: 内存泄漏

**症状**: 内存使用持续增长

**诊断步骤**:

```bash
# 1. 监控内存使用
docker stats qwq --no-stream

# 2. 生成内存 profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 3. 分析 profile
go tool pprof heap.prof

# 4. 查看 goroutine 泄漏
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

**解决方案**:

```bash
# 1. 重启服务（临时）
docker-compose restart qwq

# 2. 设置内存限制
# 在 docker-compose.yml 中
mem_limit: 2g

# 3. 启用 GC 优化
# 在 .env 中
GOGC=50

# 4. 升级到最新版本
git pull
docker-compose build
docker-compose up -d
```

## AI 服务问题

### 问题: AI 响应错误

**错误信息**:
```
Error: AI service unavailable
```

**解决方案**:

```bash
# 1. 检查 AI 服务状态
curl http://localhost:8080/api/v1/ai/status

# 2. 验证 API Key
echo $OPENAI_API_KEY

# 3. 测试 API 连接
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# 4. 切换到 Ollama
AI_PROVIDER=ollama
OLLAMA_HOST=http://localhost:11434

# 5. 重启服务
docker-compose restart qwq
```

### 问题: AI 响应慢

**症状**: AI 对话响应时间过长

**优化方案**:

```bash
# 1. 使用更快的模型
# OpenAI: gpt-3.5-turbo
# Ollama: llama2:7b

# 2. 减少上下文长度
# 在配置中限制历史消息数量
MAX_CONTEXT_MESSAGES=5

# 3. 启用流式响应
# 在 API 调用中
stream: true

# 4. 使用本地模型
# 安装 Ollama 并使用本地模型
```

## 日志分析

### 查看日志

```bash
# 1. 查看所有日志
docker-compose logs

# 2. 查看特定服务日志
docker-compose logs qwq
docker-compose logs nginx
docker-compose logs mysql

# 3. 实时跟踪日志
docker-compose logs -f qwq

# 4. 查看最近的日志
docker-compose logs --tail=100 qwq

# 5. 按时间过滤
docker-compose logs --since="2024-12-07T10:00:00"
```

### 日志级别

```bash
# 修改日志级别
# 在 .env 中
LOG_LEVEL=debug  # debug, info, warn, error

# 重启服务
docker-compose restart qwq
```

### 常见日志错误

**错误 1: "permission denied"**
```
解决: 检查文件权限
chmod 755 ./data
chown -R 1000:1000 ./data
```

**错误 2: "connection refused"**
```
解决: 检查服务是否启动
docker-compose ps
docker-compose up -d <service>
```

**错误 3: "out of memory"**
```
解决: 增加内存限制
在 docker-compose.yml 中增加 mem_limit
```

## 诊断工具

### 健康检查

```bash
# 1. 系统健康检查
curl http://localhost:8080/health

# 2. 详细健康状态
curl http://localhost:8080/health/detailed

# 3. 组件状态
curl http://localhost:8080/api/v1/system/status
```

### 性能分析

```bash
# 1. CPU Profile
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# 2. 内存 Profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# 3. Goroutine Profile
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# 4. 阻塞 Profile
curl http://localhost:8080/debug/pprof/block > block.prof
go tool pprof block.prof
```

### 数据库诊断

```bash
# 1. 检查数据库大小
du -sh ./data/qwq.db

# 2. 分析表大小
sqlite3 ./data/qwq.db "SELECT name, SUM(pgsize) as size FROM dbstat GROUP BY name ORDER BY size DESC;"

# 3. 检查索引
sqlite3 ./data/qwq.db ".indexes"

# 4. 优化数据库
sqlite3 ./data/qwq.db "VACUUM;"
sqlite3 ./data/qwq.db "ANALYZE;"
```

### 网络诊断

```bash
# 1. 检查端口监听
netstat -tulpn | grep 8080

# 2. 测试 API 连接
curl -v http://localhost:8080/api/v1/health

# 3. 检查 DNS 解析
nslookup yourdomain.com
dig yourdomain.com

# 4. 测试 SSL 连接
openssl s_client -connect yourdomain.com:443
```

## 获取帮助

如果以上方法都无法解决问题，请：

1. **查看文档**: 
   - [用户手册](user-manual.md)
   - [部署指南](deployment-guide.md)
   - [API 文档](http://localhost:8080/api/docs)

2. **收集信息**:
   ```bash
   # 生成诊断报告
   ./scripts/diagnostic.sh > diagnostic.txt
   ```

3. **提交 Issue**:
   - 访问 GitHub Issues
   - 提供详细的错误信息
   - 附上诊断报告

4. **联系支持**:
   - 邮件: support@qwq-aiops.com
   - 社区论坛: https://community.qwq-aiops.com

## 预防措施

### 定期维护

```bash
# 1. 清理 Docker 资源
docker system prune -a

# 2. 备份数据库
./scripts/backup.sh

# 3. 更新依赖
docker-compose pull
docker-compose build --no-cache

# 4. 检查日志大小
du -sh /var/log/qwq/
```

### 监控告警

```bash
# 1. 配置监控
# 启用 Prometheus 和 Grafana

# 2. 设置告警规则
# 在监控界面配置告警

# 3. 定期检查
# 每周查看系统状态
```

### 最佳实践

1. **定期备份**: 每天自动备份数据库
2. **监控资源**: 设置资源使用告警
3. **更新系统**: 及时更新到最新版本
4. **查看日志**: 定期检查错误日志
5. **测试恢复**: 定期测试备份恢复

---

**文档版本**: v1.0  
**最后更新**: 2024-12-07  
**维护者**: qwq AIOps Team
