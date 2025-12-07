# 📝 需要自定义的内容清单

## 🔧 必须修改的内容

### 1. 环境变量配置 (`.env`)

创建 `.env` 文件并配置以下内容：

```bash
# AI 模型配置（必填）
# 选择一个：OpenAI API 或本地 Ollama

# 方式 1: 使用 OpenAI API
OPENAI_API_KEY=sk-your-api-key-here          # ⚠️ 需要填写你的 OpenAI API Key
OPENAI_API_BASE=https://api.openai.com/v1   # 或使用其他兼容的 API（如硅基流动）

# 方式 2: 使用本地 Ollama（推荐，免费）
OLLAMA_HOST=http://localhost:11434           # Ollama 服务地址
OLLAMA_MODEL=deepseek-coder:latest           # 使用的模型名称

# 数据库配置（可选，默认使用 SQLite）
DB_TYPE=sqlite                                # sqlite / postgres / mysql
DB_HOST=localhost
DB_PORT=5432
DB_NAME=qwq
DB_USER=qwq
DB_PASSWORD=your-password-here                # ⚠️ 需要设置数据库密码

# Redis 配置（可选）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=                               # ⚠️ 如果 Redis 有密码，需要填写

# 管理员账号（首次启动时创建）
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123                       # ⚠️ 建议修改默认密码
ADMIN_EMAIL=admin@example.com                 # ⚠️ 需要填写管理员邮箱

# JWT 密钥（用于生成 Token）
JWT_SECRET=your-random-secret-key-here        # ⚠️ 需要生成随机密钥

# 服务配置
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# 日志配置
LOG_LEVEL=info                                # debug / info / warn / error
LOG_FILE=/app/logs/qwq.log
```

### 2. Docker Compose 配置 (`docker-compose.yml`)

如果使用 Docker Compose 部署，需要修改：

```yaml
services:
  qwq:
    environment:
      # ⚠️ 修改这些环境变量
      - ADMIN_PASSWORD=your-secure-password    # 管理员密码
      - JWT_SECRET=your-random-secret          # JWT 密钥
      - OPENAI_API_KEY=sk-xxx                  # OpenAI API Key（如果使用）
```

### 3. SSL 证书配置（如果使用 HTTPS）

在 `docker-compose.yml` 或部署脚本中配置：

```yaml
# ⚠️ 需要配置你的域名和邮箱
environment:
  - DOMAIN=your-domain.com                     # 你的域名
  - EMAIL=your-email@example.com               # Let's Encrypt 通知邮箱
```

### 4. 云服务 API 配置（可选）

如果使用云服务功能，需要配置：

```bash
# 阿里云 DNS（用于自动申请 SSL 证书）
ALIYUN_ACCESS_KEY_ID=your-access-key          # ⚠️ 阿里云 Access Key
ALIYUN_ACCESS_KEY_SECRET=your-secret          # ⚠️ 阿里云 Secret Key

# 腾讯云 DNS
TENCENT_SECRET_ID=your-secret-id              # ⚠️ 腾讯云 Secret ID
TENCENT_SECRET_KEY=your-secret-key            # ⚠️ 腾讯云 Secret Key

# S3 存储（用于备份）
S3_ENDPOINT=https://s3.amazonaws.com          # ⚠️ S3 端点
S3_ACCESS_KEY=your-access-key                 # ⚠️ S3 Access Key
S3_SECRET_KEY=your-secret-key                 # ⚠️ S3 Secret Key
S3_BUCKET=your-bucket-name                    # ⚠️ S3 Bucket 名称
```

---

## 📋 可选修改的内容

### 1. README.md 中的占位符

以下内容已使用真实仓库链接，但你可能想修改：

- ✅ GitHub 仓库链接：已更新为 `https://github.com/QwQBiG/qwq-aiops`
- ⚠️ 徽章链接：可以添加真实的构建状态徽章
- ⚠️ 截图：可以添加实际的系统截图
- ⚠️ 演示视频：可以添加演示视频链接

### 2. 添加真实的 GitHub Actions 徽章

在 README.md 中替换：

```markdown
<!-- 当前 -->
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/QwQBiG/qwq-aiops)

<!-- 替换为真实的 GitHub Actions 徽章 -->
[![Build Status](https://github.com/QwQBiG/qwq-aiops/workflows/Build%20and%20Test/badge.svg)](https://github.com/QwQBiG/qwq-aiops/actions)
[![Docker Build](https://github.com/QwQBiG/qwq-aiops/workflows/Docker%20Build%20and%20Publish/badge.svg)](https://github.com/QwQBiG/qwq-aiops/actions)
```

### 3. 添加 License 文件

创建 `LICENSE` 文件（MIT License）：

```
MIT License

Copyright (c) 2025 QwQBiG

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### 4. 添加贡献者信息

在 README.md 末尾添加：

```markdown
## 👥 贡献者

感谢所有为这个项目做出贡献的人！

<!-- 可以使用 https://contrib.rocks 生成贡献者图片 -->
<a href="https://github.com/QwQBiG/qwq-aiops/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=QwQBiG/qwq-aiops" />
</a>
```

### 5. 添加联系方式

在 README.md 中添加：

```markdown
## 📧 联系我们

- 📮 Email: your-email@example.com           # ⚠️ 填写你的邮箱
- 💬 微信群: [扫码加入]                       # ⚠️ 可以添加微信群二维码
- 🐦 Twitter: @your_twitter                  # ⚠️ 填写你的 Twitter
- 📺 YouTube: [频道链接]                      # ⚠️ 如果有演示视频
```

---

## 🔐 安全建议

### 1. 生成安全的密钥

```bash
# 生成 JWT 密钥（32 字节随机字符串）
openssl rand -base64 32

# 生成管理员密码（建议使用密码管理器）
openssl rand -base64 16
```

### 2. 不要提交敏感信息

确保以下文件在 `.gitignore` 中：

```
.env
*.key
*.pem
*.crt
config/secrets.yaml
```

### 3. 使用 GitHub Secrets

在 GitHub Actions 中使用敏感信息：

1. 进入仓库 Settings → Secrets and variables → Actions
2. 添加以下 Secrets：
   - `OPENAI_API_KEY`
   - `DOCKER_USERNAME`
   - `DOCKER_PASSWORD`
   - 其他敏感配置

---

## ✅ 检查清单

部署前请确认：

- [ ] 已创建并配置 `.env` 文件
- [ ] 已修改默认管理员密码
- [ ] 已生成 JWT 密钥
- [ ] 已配置 AI 模型（OpenAI 或 Ollama）
- [ ] 已配置数据库（如果不使用 SQLite）
- [ ] 已配置云服务 API（如果使用相关功能）
- [ ] 已添加 LICENSE 文件
- [ ] 已更新 README 中的联系方式
- [ ] 已配置 GitHub Secrets（如果使用 CI/CD）
- [ ] 已测试本地部署
- [ ] 已测试 Docker 部署

---

## 📚 相关文档

- [部署指南](docs/deployment-guide.md)
- [配置说明](docs/configuration.md)
- [安全最佳实践](docs/security-audit-report.md)
- [故障排查](docs/troubleshooting-guide.md)

---

**最后更新**: 2025-12-07  
**状态**: ✅ 仓库链接已更新
