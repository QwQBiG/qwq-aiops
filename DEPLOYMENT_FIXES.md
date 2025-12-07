# 部署问题修复总结

## 🔧 已修复的问题

### 1. ❌ Docker 镜像拉取失败

**问题**：
```
ERROR: Head "https://ghcr.io/v2/your-org/qwq-aiops/manifests/latest": denied
```

**原因**：docker-compose.yml 中使用了占位符 `your-org`，而不是实际的 GitHub 用户名。

**修复**：
- ✅ 修改为使用本地构建：`build: .`
- ✅ 更新镜像名称为：`qwq-aiops:latest`
- ✅ 添加注释说明如何使用 GitHub Container Registry

### 2. ❌ 端口 8080 被占用

**问题**：本地 8080 端口已被其他服务占用。

**修复**：
- ✅ 修改端口映射为：`8081:8080`
- ✅ 更新所有文档中的访问地址
- ✅ 添加端口冲突解决方案文档

### 3. ❌ GitHub 仓库链接错误

**问题**：多处使用了占位符 `your-org`。

**修复**：
- ✅ Dockerfile 中的标签链接
- ✅ 部署指南中的克隆地址
- ✅ README 中的所有链接

## 📝 修改的文件

### 1. docker-compose.yml
```yaml
# 修改前
image: ghcr.io/your-org/qwq-aiops:latest
ports:
  - "8080:8080"

# 修改后
build: .
image: qwq-aiops:latest
ports:
  - "8081:8080"  # 避免端口冲突
```

### 2. Dockerfile
```dockerfile
# 修改前
org.opencontainers.image.source="https://github.com/your-org/qwq-aiops"

# 修改后
org.opencontainers.image.source="https://github.com/QwQBiG/qwq-aiops"
```

### 3. docs/deployment-guide.md
- ✅ 更新所有 GitHub 仓库地址
- ✅ 修改访问端口为 8081
- ✅ 添加端口冲突解决方案
- ✅ 添加本地构建说明

### 4. README.md
- ✅ 更新访问端口为 8081
- ✅ 添加构建命令说明

### 5. 新增文档
- ✅ `PORT_CHANGE_GUIDE.md` - 端口修改指南
- ✅ `QUICK_FIX.md` - 快速修复指南
- ✅ `DEPLOYMENT_FIXES.md` - 本文档

## 🚀 现在可以正常部署了

### 快速启动

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 配置环境变量（可选）
cp .env.example .env
# 编辑 .env 文件，配置 AI API Key

# 3. 构建并启动
docker-compose up -d --build

# 4. 查看日志
docker-compose logs -f qwq

# 5. 访问系统
# 前端界面: http://localhost:8081
# API 文档: http://localhost:8081/api/docs
```

### 验证部署

```bash
# 检查容器状态
docker-compose ps

# 预期输出
NAME        IMAGE              STATUS         PORTS
qwq         qwq-aiops:latest   Up 2 minutes   0.0.0.0:8081->8080/tcp

# 健康检查
curl http://localhost:8081/api/health

# 预期输出
{
  "status": "healthy",
  "version": "v1.0.0",
  ...
}
```

## ⏱️ 构建时间说明

首次构建大约需要 **5-10 分钟**：

1. **前端构建**（Vue 3）：3-5 分钟
   - 下载 npm 依赖
   - 编译 TypeScript
   - 打包生产版本

2. **后端构建**（Go）：2-3 分钟
   - 下载 Go 模块
   - 编译二进制文件
   - 优化和压缩

3. **镜像打包**：1-2 分钟
   - 创建最终镜像
   - 安装运行时依赖

## 🔍 故障排查

### 问题 1：构建失败

```bash
# 清理缓存重新构建
docker-compose build --no-cache
```

### 问题 2：端口仍然冲突

```bash
# 检查端口占用
netstat -ano | findstr :8081

# 修改为其他端口
# 编辑 docker-compose.yml，改为 8082:8080
```

### 问题 3：容器无法启动

```bash
# 查看详细日志
docker-compose logs qwq

# 检查 Docker 状态
docker info
```

## 📚 相关文档

- **快速修复指南**: [QUICK_FIX.md](QUICK_FIX.md)
- **端口修改指南**: [PORT_CHANGE_GUIDE.md](PORT_CHANGE_GUIDE.md)
- **完整部署指南**: [docs/deployment-guide.md](docs/deployment-guide.md)
- **故障排查指南**: [docs/troubleshooting-guide.md](docs/troubleshooting-guide.md)

## 🎯 下一步

部署成功后，您可以：

1. **配置 AI 功能**
   - 编辑 `.env` 文件
   - 配置 OpenAI API Key 或 Ollama

2. **修改默认密码**
   - 登录系统：http://localhost:8081
   - 使用默认账号：admin / admin123
   - 立即修改密码

3. **探索功能**
   - 应用商店：一键部署常用应用
   - 容器管理：管理 Docker 容器
   - 网站管理：配置 Nginx 和 SSL
   - AI 助手：自然语言运维

4. **生产部署**
   - 配置 Nginx 反向代理
   - 申请 SSL 证书
   - 配置防火墙规则
   - 设置自动备份

## 💡 优化建议

### 加速构建

**使用国内镜像源**：

```bash
# Go 模块代理
export GOPROXY=https://goproxy.cn,direct

# npm 镜像（编辑 frontend/.npmrc）
registry=https://registry.npmmirror.com
```

### 减少镜像大小

当前镜像大小约 **150-200MB**（已优化）：
- ✅ 使用 Alpine Linux 基础镜像
- ✅ 多阶段构建
- ✅ 清理构建缓存
- ✅ 静态编译 Go 程序

### 资源限制

在 `docker-compose.yml` 中已配置：
```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
```

## 🆘 需要帮助？

如果遇到问题：

1. **查看文档**
   - [QUICK_FIX.md](QUICK_FIX.md) - 快速修复
   - [docs/troubleshooting-guide.md](docs/troubleshooting-guide.md) - 详细排查

2. **查看日志**
   ```bash
   docker-compose logs -f qwq
   ```

3. **提交 Issue**
   - GitHub: https://github.com/QwQBiG/qwq-aiops/issues
   - 请附上错误日志和系统信息

---

**修复完成时间**: 2025-12-07  
**修复版本**: v1.0.0  
**状态**: ✅ 已验证可用
