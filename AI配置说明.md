# 🤖 AI 服务配置说明

## 重要提示 ⚠️

**qwq 是 AI 驱动的智能运维平台，必须配置 AI 服务才能正常运行！**

如果不配置 AI 服务，启动时会报错：
```
Error: critical: 未找到 API Key
```

## 快速配置

### 方法 1：使用配置向导（推荐）⭐

```bash
chmod +x 配置AI服务.sh
./配置AI服务.sh
```

### 方法 2：使用一键部署脚本

```bash
chmod +x 一键部署.sh
sudo ./一键部署.sh
```

脚本会在部署前提示配置 AI 服务。

### 方法 3：手动配置

编辑 `docker-compose.yml` 文件：

```bash
nano docker-compose.yml
```

找到 AI 配置部分（约第 35 行），根据你的情况选择：

## AI 服务选项

### 选项 1：OpenAI API（推荐新手）

**优点**：
- ✅ 功能最强大
- ✅ 配置简单
- ✅ 响应速度快

**缺点**：
- ❌ 需要付费
- ❌ 数据会发送到 OpenAI

**配置方法**：

```yaml
environment:
  - AI_PROVIDER=openai
  - OPENAI_API_KEY=sk-你的真实API-Key
```

**获取 API Key**：
1. 访问 https://platform.openai.com/api-keys
2. 注册并创建 API Key
3. 复制 Key 到配置中

### 选项 2：Ollama 本地模型（推荐企业）

**优点**：
- ✅ 完全免费
- ✅ 数据不出服务器
- ✅ 隐私安全
- ✅ 无网络依赖

**缺点**：
- ❌ 需要额外安装
- ❌ 占用服务器资源

**配置方法**：

1. **安装 Ollama**：
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

2. **下载模型**（推荐 qwen2.5）：
```bash
ollama pull qwen2.5:7b
```

3. **配置 docker-compose.yml**：
```yaml
environment:
  - AI_PROVIDER=ollama
  - OLLAMA_HOST=http://host.docker.internal:11434
```

4. **重启服务**：
```bash
docker compose restart qwq
```

### 选项 3：其他兼容服务

支持任何兼容 OpenAI API 的服务，如：
- 硅基流动（国内推荐）
- DeepSeek API
- 阿里云通义千问
- 腾讯云混元

**配置方法**：

```yaml
environment:
  - AI_PROVIDER=openai
  - OPENAI_API_KEY=你的API-Key
  - OPENAI_BASE_URL=https://api.服务商.com/v1
```

## 配置后的操作

### 1. 重启服务

```bash
docker compose restart qwq
```

### 2. 查看日志

```bash
docker compose logs -f qwq
```

**成功的日志应该类似**：
```
qwq  | Starting qwq AIOps Platform...
qwq  | AI Provider: openai
qwq  | Server listening on :8080
```

### 3. 访问系统

打开浏览器访问：http://localhost:8081

## 常见问题

### Q1: 我没有 OpenAI API Key 怎么办？

**解决方案**：使用 Ollama 本地模型，完全免费。

```bash
# 安装 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 下载模型
ollama pull qwen2.5:7b

# 配置并重启
./配置AI服务.sh
```

### Q2: Ollama 连接失败怎么办？

**检查 Ollama 是否运行**：
```bash
# 检查服务状态
systemctl status ollama

# 或者测试连接
curl http://localhost:11434/api/tags
```

**如果 Ollama 在 Docker 中运行**：
```yaml
- OLLAMA_HOST=http://ollama:11434  # 容器名
```

**如果 Ollama 在其他服务器**：
```yaml
- OLLAMA_HOST=http://服务器IP:11434
```

### Q3: 可以先跳过 AI 配置吗？

**不建议**。qwq 的核心功能依赖 AI，不配置会导致：
- ❌ 服务无法启动
- ❌ 所有 AI 功能不可用
- ❌ 需要手动配置后重启

**如果确实要跳过**：
1. 在一键部署时选择"跳过配置"
2. 稍后运行 `./配置AI服务.sh` 配置
3. 重启服务

### Q4: 如何更换 AI 服务？

```bash
# 1. 运行配置向导
./配置AI服务.sh

# 2. 选择新的 AI 服务
# 3. 重启服务
docker compose restart qwq
```

### Q5: API Key 会泄露吗？

**安全措施**：
- ✅ API Key 存储在服务器本地
- ✅ 不会提交到 Git（已加入 .gitignore）
- ✅ 只有容器内部可以访问

**建议**：
- 🔒 使用环境变量存储 Key
- 🔒 定期轮换 API Key
- 🔒 设置 API Key 使用限额

## 推荐配置

### 个人学习/测试

```yaml
- AI_PROVIDER=ollama
- OLLAMA_HOST=http://host.docker.internal:11434
```

**理由**：免费，适合学习和测试。

### 小型团队

```yaml
- AI_PROVIDER=openai
- OPENAI_API_KEY=sk-your-key
```

**理由**：功能强大，配置简单。

### 企业生产环境

```yaml
- AI_PROVIDER=ollama
- OLLAMA_HOST=http://ollama-server:11434
```

**理由**：数据安全，成本可控。

## 相关文档

- [快速开始.md](快速开始.md) - 快速部署指南
- [一键部署说明.md](一键部署说明.md) - 详细部署说明
- [docs/deployment-guide.md](docs/deployment-guide.md) - 完整部署指南

## 技术支持

如果遇到问题：

1. **查看日志**：`docker compose logs -f qwq`
2. **提交 Issue**：https://github.com/QwQBiG/qwq-aiops/issues
3. **社区讨论**：https://github.com/QwQBiG/qwq-aiops/discussions

---

**记住**：AI 配置是 qwq 正常运行的前提！🤖
