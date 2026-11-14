# 🚀 从这里开始

欢迎使用 qwq AIOps 平台！

## 📋 快速导航

### 新手入门
1. **[快速开始](快速开始.md)** - 5 分钟快速部署
2. **[部署检查清单](部署检查清单.md)** - 确保部署成功
3. **[用户手册](../reports/user-manual.md)** - 功能使用说明

### 部署方式

#### 🎯 推荐：一键部署

**Linux/macOS**:
```bash
chmod +x deploy.sh
./deploy.sh
```

**Windows**:
```bash
start.bat
```

#### 🔧 手动部署

```bash
# 1. 配置 AI 服务
cp .env.example .env
nano .env

# 2. 启动服务
docker compose up -d --build

# 3. 查看日志
docker compose logs -f qwq
```

## ⚡ 快速命令

```bash
# 启动服务
docker compose up -d

# 停止服务
docker compose down

# 查看日志
docker compose logs -f qwq

# 重启服务
docker compose restart

# 查看状态
docker compose ps

# 健康检查
curl http://localhost:8081/api/health
```

## 🌐 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端界面 | http://localhost:8081 | 主界面 |
| API 文档 | http://localhost:8081/api/docs | Swagger |
| Prometheus | http://localhost:9091 | 监控 |
| Grafana | http://localhost:3000 | 可视化 |

**默认账号**: admin / admin123

## 📚 文档目录

- [README](../../../README.md) - 项目介绍
- [快速开始](快速开始.md) - 快速部署指南
- [部署检查清单](部署检查清单.md) - 部署验证
- [用户手册](../reports/user-manual.md) - 功能说明
- [部署指南](../reports/deployment-guide.md) - 详细部署
- API 文档 - 历史引用的 `docs/api.md` 未随仓库提供
- [常见问题](../reports/troubleshooting-guide.md) - FAQ

## ⚠️ 重要提示

### 必须配置 AI 服务

qwq 是 AI 驱动的平台，必须配置 AI 服务才能使用！

**选项 1: OpenAI API**
```bash
AI_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key-here
```

**选项 2: Ollama 本地模型**
```bash
AI_PROVIDER=ollama
OLLAMA_HOST=http://host.docker.internal:11434
OLLAMA_MODEL=qwen2.5:7b
```

### 生产环境部署

1. 修改默认密码
2. 修改 JWT_SECRET 和 ENCRYPTION_KEY
3. 启用 HTTPS
4. 配置防火墙
5. 设置自动备份

## 🆘 需要帮助？

- 📖 查看[完整文档](../../../README.md)
- 🐛 [提交 Issue](https://github.com/QwQBiG/qwq-aiops/issues)
- 💬 [社区讨论](https://github.com/QwQBiG/qwq-aiops/discussions)
- 📧 联系我们

## 🎉 开始使用

现在就开始你的智能运维之旅吧！

```bash
./deploy.sh
```

访问 **http://localhost:8081** 🚀
