# 快速修复指南

## 问题：Docker 镜像拉取失败

如果您遇到以下错误：

```
ERROR: Head "https://ghcr.io/v2/your-org/qwq-aiops/manifests/latest": denied
```

这是因为 docker-compose.yml 中配置的是占位符镜像地址。

## ✅ 已修复

我已经将 `docker-compose.yml` 修改为使用本地构建，不再依赖远程镜像。

## 🚀 现在可以这样启动

### 方法 1：完整启动（推荐）

```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看日志
docker-compose logs -f qwq
```

### 方法 2：仅启动核心服务

如果您不需要 MySQL、Redis、Prometheus 等可选服务：

```bash
# 只启动 qwq 主服务
docker-compose up -d --build qwq

# 查看日志
docker-compose logs -f qwq
```

### 方法 3：手动构建

```bash
# 1. 构建镜像
docker build -t qwq-aiops:latest .

# 2. 运行容器
docker run -d \
  --name qwq \
  --restart unless-stopped \
  -p 8081:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -e TZ=Asia/Shanghai \
  -e AI_PROVIDER=openai \
  -e OPENAI_API_KEY=your-api-key-here \
  qwq-aiops:latest

# 3. 查看日志
docker logs -f qwq
```

## 📝 构建说明

### 构建时间

首次构建大约需要 **5-10 分钟**，包括：

1. **前端构建**（Vue 3）：约 3-5 分钟
2. **后端构建**（Go）：约 2-3 分钟
3. **镜像打包**：约 1-2 分钟

### 构建要求

- **磁盘空间**：至少 2GB 可用空间
- **内存**：建议 4GB+
- **网络**：需要下载 Node.js 和 Go 依赖

### 加速构建

如果构建很慢，可以配置国内镜像源：

**Go 模块代理**：

```bash
# 在构建前设置环境变量
export GOPROXY=https://goproxy.cn,direct
docker-compose build
```

**npm 镜像**：

编辑 `frontend/.npmrc`：

```
registry=https://registry.npmmirror.com
```

## 🔍 验证构建

构建完成后，验证镜像：

```bash
# 查看镜像
docker images | grep qwq-aiops

# 预期输出
qwq-aiops    latest    xxxxx    2 minutes ago    xxx MB
```

## 🌐 访问系统

构建并启动成功后：

- **前端界面**: http://localhost:8081
- **API 文档**: http://localhost:8081/api/docs
- **健康检查**: http://localhost:8081/api/health

## ❌ 常见构建错误

### 错误 1：前端构建失败

```
ERROR: failed to solve: process "/bin/sh -c npm ci" did not complete successfully
```

**解决方案**：

```bash
# 清理前端依赖
cd frontend
rm -rf node_modules package-lock.json
npm install
cd ..

# 重新构建
docker-compose build --no-cache
```

### 错误 2：Go 模块下载失败

```
ERROR: failed to solve: process "/bin/sh -c go mod download" did not complete successfully
```

**解决方案**：

```bash
# 使用国内代理
export GOPROXY=https://goproxy.cn,direct
docker-compose build
```

### 错误 3：磁盘空间不足

```
ERROR: failed to solve: no space left on device
```

**解决方案**：

```bash
# 清理 Docker 缓存
docker system prune -a

# 检查磁盘空间
df -h
```

## 📦 发布到 GitHub Container Registry（可选）

如果您想发布镜像到 GitHub Container Registry，方便其他人使用：

### 1. 创建 Personal Access Token

在 GitHub 设置中创建 PAT，权限选择：
- `write:packages`
- `read:packages`
- `delete:packages`

### 2. 登录 GHCR

```bash
echo "YOUR_PAT" | docker login ghcr.io -u QwQBiG --password-stdin
```

### 3. 构建并推送

```bash
# 构建镜像
docker build -t ghcr.io/qwqbig/qwq-aiops:latest .
docker build -t ghcr.io/qwqbig/qwq-aiops:v1.0.0 .

# 推送镜像
docker push ghcr.io/qwqbig/qwq-aiops:latest
docker push ghcr.io/qwqbig/qwq-aiops:v1.0.0
```

### 4. 更新 docker-compose.yml

发布后，可以修改 `docker-compose.yml` 使用远程镜像：

```yaml
services:
  qwq:
    image: ghcr.io/qwqbig/qwq-aiops:latest
    # build: .  # 注释掉本地构建
```

## 🆘 需要帮助？

- **部署指南**: [docs/deployment-guide.md](docs/deployment-guide.md)
- **端口修改**: [PORT_CHANGE_GUIDE.md](PORT_CHANGE_GUIDE.md)
- **故障排查**: [docs/troubleshooting-guide.md](docs/troubleshooting-guide.md)
- **GitHub Issues**: https://github.com/QwQBiG/qwq-aiops/issues

---

**提示**：构建成功后，后续启动只需要 `docker-compose up -d`，不需要 `--build` 参数。
