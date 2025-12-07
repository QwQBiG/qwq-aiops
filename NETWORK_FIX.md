# 网络问题修复指南

## 问题描述

构建 Docker 镜像时遇到网络超时错误：

```
ERROR: failed to solve: process "/bin/sh -c go mod download && go mod verify" did not complete successfully
go: github.com/alecthomas/chroma@v0.10.0: Get "https://proxy.golang.org/...": dial tcp 142.250.66.81:443: i/o timeout
```

## 原因分析

这是因为访问 Go 官方代理 `proxy.golang.org` 超时，通常是网络问题导致的。

## ✅ 已修复

我已经在 Dockerfile 中添加了国内 Go 代理配置：

```dockerfile
# 设置 Go 代理（使用国内镜像加速）
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GO111MODULE=on
```

## 🚀 现在可以重新构建

### 方法 1：使用启动脚本

**Windows**：
```cmd
start.bat
```

**Linux/macOS**：
```bash
./start.sh
```

### 方法 2：手动构建

```bash
# 清理之前的构建缓存
docker-compose down
docker system prune -f

# 重新构建
docker-compose build --no-cache

# 启动服务
docker-compose up -d
```

### 方法 3：使用构建参数

如果还是有问题，可以在构建时指定代理：

```bash
docker-compose build --build-arg GOPROXY=https://goproxy.cn,direct
```

## 🔧 其他解决方案

### 方案 1：配置 Docker 代理（如果您有代理）

创建或编辑 `~/.docker/config.json`：

```json
{
  "proxies": {
    "default": {
      "httpProxy": "http://proxy.example.com:8080",
      "httpsProxy": "http://proxy.example.com:8080",
      "noProxy": "localhost,127.0.0.1"
    }
  }
}
```

### 方案 2：使用本地 Go 模块缓存

如果您本地已经下载过依赖：

```bash
# 在宿主机上下载依赖
go mod download

# 然后构建时会使用本地缓存
docker-compose build
```

### 方案 3：修改 go.mod 使用国内镜像

在项目根目录创建 `.netrc` 文件（不推荐，已在 Dockerfile 中配置）：

```
machine goproxy.cn
machine goproxy.io
```

## 📊 可用的 Go 代理列表

按推荐顺序：

1. **goproxy.cn** (七牛云) - 推荐 ⭐
   - `https://goproxy.cn`
   - 国内访问速度快，稳定性好

2. **goproxy.io** (备用)
   - `https://goproxy.io`
   - 备用代理

3. **阿里云**
   - `https://mirrors.aliyun.com/goproxy/`

4. **腾讯云**
   - `https://mirrors.tencent.com/go/`

## 🔍 验证代理配置

构建时查看日志，应该看到：

```
=> [backend-builder 3/10] ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
=> [backend-builder 4/10] ENV GO111MODULE=on
=> [backend-builder 5/10] COPY go.mod go.sum ./
=> [backend-builder 6/10] RUN go mod download && go mod verify
```

如果看到从 `goproxy.cn` 下载，说明配置成功。

## ⏱️ 预期构建时间

使用国内代理后：

- **Go 依赖下载**：1-2 分钟（首次）
- **Go 编译**：1-2 分钟
- **总构建时间**：5-8 分钟

## ❌ 如果还是失败

### 检查网络连接

```bash
# 测试是否能访问 goproxy.cn
curl -I https://goproxy.cn

# 预期输出
HTTP/2 200
```

### 检查 DNS

```bash
# Windows
nslookup goproxy.cn

# Linux/macOS
dig goproxy.cn
```

### 使用 VPN 或代理

如果您的网络环境有限制，可能需要：

1. 使用 VPN
2. 配置系统代理
3. 配置 Docker 代理

## 🐛 调试构建过程

如果需要查看详细的构建日志：

```bash
# 查看详细构建日志
docker-compose build --progress=plain --no-cache

# 或者
docker build --progress=plain --no-cache -t qwq-aiops:latest .
```

## 📝 构建成功标志

构建成功后，您会看到：

```
=> [backend-builder 6/10] RUN go mod download && go mod verify  ✓
=> [backend-builder 7/10] COPY cmd/ ./cmd/                      ✓
=> [backend-builder 8/10] COPY internal/ ./internal/            ✓
=> [backend-builder 9/10] RUN CGO_ENABLED=0 GOOS=linux ...      ✓
...
=> => naming to docker.io/library/qwq-aiops:latest             ✓
```

## 🎯 快速测试

构建完成后，快速测试：

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f qwq

# 健康检查
curl http://localhost:8081/api/health
```

## 💡 优化建议

### 1. 使用构建缓存

Docker 会缓存每一层，如果 `go.mod` 没有变化，不会重新下载依赖。

### 2. 多阶段构建

Dockerfile 已经使用了多阶段构建，最终镜像只包含必要的文件。

### 3. 并行构建

如果您的机器性能好，可以增加并行度：

```bash
docker-compose build --parallel
```

## 🆘 仍然无法解决？

1. **查看完整日志**
   ```bash
   docker-compose build --progress=plain 2>&1 | tee build.log
   ```

2. **检查磁盘空间**
   ```bash
   docker system df
   ```

3. **清理 Docker 缓存**
   ```bash
   docker system prune -a
   ```

4. **提交 Issue**
   - GitHub: https://github.com/QwQBiG/qwq-aiops/issues
   - 附上 `build.log` 文件

## 📚 相关文档

- [QUICK_FIX.md](QUICK_FIX.md) - 快速修复指南
- [START_HERE.md](START_HERE.md) - 快速开始
- [docs/deployment-guide.md](docs/deployment-guide.md) - 完整部署指南

---

**修复时间**: 2025-12-07  
**状态**: ✅ 已添加国内代理配置
