# 🚀 从这里开始

欢迎使用 qwq AIOps 平台！本文档将帮助您快速启动系统。

## ✅ 问题已修复

我已经修复了以下问题：

1. ✅ **Docker 镜像拉取失败** - 改为本地构建
2. ✅ **端口 8080 冲突** - 改为使用 8081 端口
3. ✅ **GitHub 仓库链接错误** - 更新为正确的地址

## 🎯 快速启动（3 种方式）

### 方式 1：使用启动脚本（最简单）⭐

**Windows 用户**：
```cmd
双击运行 start.bat
```

**Linux/macOS 用户**：
```bash
chmod +x start.sh
./start.sh
```

### 方式 2：使用 Docker Compose

```bash
# 构建并启动（约 6-10 分钟）
docker compose up -d --build

# 查看日志
docker compose logs -f qwq

# 注意：使用 docker compose（V2，无连字符）而不是 docker-compose（V1）
```

### 方式 3：手动构建

```bash
# 构建镜像
docker build -t qwq-aiops:latest .

# 运行容器
docker run -d \
  --name qwq \
  -p 8081:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v $(pwd)/data:/app/data \
  qwq-aiops:latest
```

## 🌐 访问系统

启动成功后，访问以下地址：

- **前端界面**: http://localhost:8081
- **API 文档**: http://localhost:8081/api/docs
- **健康检查**: http://localhost:8081/api/health

**默认账号**：
- 用户名: `admin`
- 密码: `admin123`

⚠️ **重要**：首次登录后请立即修改密码！

## ⏱️ 首次启动说明

首次启动需要构建 Docker 镜像，大约需要 **5-10 分钟**：

1. **前端构建**（Vue 3）：3-5 分钟
2. **后端构建**（Go）：2-3 分钟
3. **镜像打包**：1-2 分钟

请耐心等待，构建完成后会自动启动。

## 🔧 配置 AI 功能（可选）

如果需要使用 AI 智能运维功能：

1. **复制配置文件**：
   ```bash
   cp .env.example .env
   ```

2. **编辑 .env 文件**，配置 AI API：
   ```bash
   # OpenAI
   AI_PROVIDER=openai
   OPENAI_API_KEY=sk-your-api-key-here
   
   # 或使用 Ollama 本地模型
   AI_PROVIDER=ollama
   OLLAMA_HOST=http://localhost:11434
   ```

3. **重启服务**：
   ```bash
   docker compose restart qwq
   ```

## 📚 文档导航

### 快速参考
- **[快速修复指南](QUICK_FIX.md)** - 解决常见启动问题
- **[端口修改指南](PORT_CHANGE_GUIDE.md)** - 修改访问端口
- **[修复总结](DEPLOYMENT_FIXES.md)** - 查看所有修复内容

### 完整文档
- **[部署指南](docs/deployment-guide.md)** - 详细的部署说明
- **[用户手册](docs/user-manual.md)** - 功能使用指南
- **[故障排查](docs/troubleshooting-guide.md)** - 问题解决方案
- **[README](README.md)** - 项目介绍

## 🔍 验证部署

启动后，运行以下命令验证：

```bash
# 检查容器状态
docker compose ps

# 预期输出
NAME   IMAGE              STATUS         PORTS
qwq    qwq-aiops:latest   Up 2 minutes   0.0.0.0:8081->8080/tcp

# 健康检查
curl http://localhost:8081/api/health

# 预期输出
{
  "status": "healthy",
  "version": "v1.0.0",
  ...
}
```

## ❓ 常见问题

### Q1: 构建失败怎么办？

**如果是网络超时（Go 模块下载失败）**：

我已经配置了国内代理，直接重新构建：

```bash
# 清理缓存重新构建
docker compose down
docker system prune -f
docker compose build --no-cache
docker compose up -d
```

详细说明：[NETWORK_FIX.md](NETWORK_FIX.md)

**如果是其他错误**：

```bash
# 查看详细日志
docker compose build --progress=plain
```

### Q2: 端口 8081 也被占用？

编辑 `docker-compose.yml`，修改端口映射：
```yaml
ports:
  - "8082:8080"  # 改为 8082 或其他可用端口
```

### Q3: 如何查看日志？

```bash
# 查看实时日志
docker compose logs -f qwq

# 查看最近 100 行
docker compose logs --tail 100 qwq
```

### Q4: 如何停止服务？

```bash
# 停止服务
docker compose down

# 停止并删除数据卷
docker compose down -v
```

### Q5: 如何更新系统？

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker compose up -d --build
```

## 🎓 下一步

部署成功后，您可以：

### 1. 探索核心功能

- **应用商店** - 一键部署 MySQL、Redis、Nginx 等应用
- **容器管理** - 管理 Docker 容器和镜像
- **网站管理** - 配置 Nginx 反向代理和 SSL 证书
- **数据库管理** - 管理多种数据库
- **AI 助手** - 自然语言运维对话

### 2. 配置生产环境

- 配置 Nginx 反向代理
- 申请 SSL 证书（Let's Encrypt）
- 配置防火墙规则
- 设置自动备份
- 配置监控告警

### 3. 学习高级功能

- 集群部署（高可用）
- 多租户管理
- 权限控制（RBAC）
- 自定义监控规则
- Webhook 集成

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

4. **社区讨论**
   - Discussions: https://github.com/QwQBiG/qwq-aiops/discussions

## 📊 系统要求

- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **内存**: 4GB+（推荐 8GB+）
- **磁盘**: 20GB+（推荐 50GB+）
- **CPU**: 2核+（推荐 4核+）

## 🌟 特性亮点

- 🤖 **AI 驱动** - 自然语言运维交互
- 🔒 **数据安全** - 支持本地私有模型
- 🚀 **一站式管理** - 应用商店、容器、网站、数据库
- ⚡ **高可用架构** - 集群部署、负载均衡
- ✅ **生产就绪** - 13 个核心属性，96+ 子属性测试

## 📄 许可证

MIT License. Copyright (c) 2025 qwqBig.

---

**祝您使用愉快！** 🎉

如有问题，欢迎提交 Issue 或参与讨论。
