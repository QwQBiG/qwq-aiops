# GitHub 发布准备清单

## 发布前准备

### 1. 代码准备

- [x] 所有功能已完成
- [x] 所有测试已通过
- [x] 代码已审查
- [x] 文档已更新
- [x] CHANGELOG 已更新

### 2. Docker 镜像准备

- [x] Dockerfile 已优化
- [x] .dockerignore 已配置
- [x] 多阶段构建已实现
- [x] 健康检查已配置
- [x] 非 root 用户运行

### 3. CI/CD 配置

- [x] GitHub Actions 工作流已配置
- [x] 自动构建已测试
- [x] 多平台构建已启用
- [x] 镜像推送已配置

### 4. 文档准备

- [x] README.md 已更新
- [x] DOCKER.md 已创建
- [x] 用户手册已完成
- [x] API 文档已生成
- [x] 发布说明已准备

## 发布步骤

### 步骤 1: 创建 GitHub 仓库

```bash
# 1. 在 GitHub 上创建新仓库
# 仓库名: qwq-aiops
# 描述: AI-Powered Intelligent Operations Platform
# 可见性: Public

# 2. 初始化本地仓库
git init
git add .
git commit -m "Initial commit: qwq AIOps Platform v1.0.0"

# 3. 添加远程仓库
git remote add origin https://github.com/your-org/qwq-aiops.git

# 4. 推送代码
git branch -M main
git push -u origin main
```

### 步骤 2: 配置 GitHub Secrets

在 GitHub 仓库设置中添加以下 Secrets：

1. **GITHUB_TOKEN**: 自动提供，用于推送镜像到 GHCR
2. **DOCKERHUB_USERNAME**: Docker Hub 用户名（可选）
3. **DOCKERHUB_TOKEN**: Docker Hub 访问令牌（可选）

### 步骤 3: 创建 Release

```bash
# 1. 创建标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 2. 推送标签
git push origin v1.0.0

# 3. 在 GitHub 上创建 Release
# - 访问: https://github.com/your-org/qwq-aiops/releases/new
# - 选择标签: v1.0.0
# - 标题: qwq AIOps Platform v1.0.0
# - 描述: 复制 docs/release-notes-v1.0.md 的内容
# - 上传资产: 无需上传（Docker 镜像会自动构建）
```

### 步骤 4: 验证自动构建

```bash
# 1. 检查 GitHub Actions
# 访问: https://github.com/your-org/qwq-aiops/actions

# 2. 等待构建完成（约 10-15 分钟）

# 3. 验证镜像
docker pull ghcr.io/your-org/qwq-aiops:v1.0.0
docker pull ghcr.io/your-org/qwq-aiops:latest

# 4. 测试镜像
docker run --rm ghcr.io/your-org/qwq-aiops:v1.0.0 --version
```

### 步骤 5: 更新文档

```bash
# 1. 更新 README.md 中的镜像地址
# 将所有 ghcr.io/your-org 替换为实际的组织名

# 2. 更新 DOCKER.md 中的示例
# 确保所有命令使用正确的镜像地址

# 3. 提交更新
git add README.md DOCKER.md
git commit -m "docs: update image registry URLs"
git push
```

## 发布后任务

### 1. 社区推广

- [ ] 在 GitHub 上添加 Topics
  - aiops
  - devops
  - monitoring
  - docker
  - golang
  - vue
  - ai
  - llm

- [ ] 创建 GitHub Discussions
  - 公告板块
  - Q&A 板块
  - 功能建议板块

- [ ] 提交到 Awesome Lists
  - awesome-aiops
  - awesome-devops
  - awesome-docker

### 2. 文档站点

- [ ] 部署文档站点（可选）
  - 使用 GitHub Pages
  - 或使用 Read the Docs
  - 或使用 Docusaurus

### 3. 监控和反馈

- [ ] 设置 GitHub Issues 模板
- [ ] 设置 Pull Request 模板
- [ ] 配置 GitHub Insights
- [ ] 监控 Star 和 Fork 数量

### 4. 持续维护

- [ ] 定期更新依赖
- [ ] 修复报告的 Bug
- [ ] 实现功能请求
- [ ] 发布新版本

## GitHub 仓库配置

### 仓库设置

**General**:
- Description: AI-Powered Intelligent Operations Platform
- Website: https://your-org.github.io/qwq-aiops
- Topics: aiops, devops, monitoring, docker, golang, vue, ai, llm

**Features**:
- ✅ Issues
- ✅ Projects
- ✅ Wiki
- ✅ Discussions
- ✅ Sponsorships

**Pull Requests**:
- ✅ Allow squash merging
- ✅ Allow rebase merging
- ✅ Automatically delete head branches

**Actions**:
- ✅ Allow all actions and reusable workflows
- ✅ Allow GitHub Actions to create and approve pull requests

**Packages**:
- ✅ Inherit access from source repository

### 分支保护规则

**main 分支**:
- ✅ Require a pull request before merging
- ✅ Require approvals (1)
- ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging
- ✅ Require conversation resolution before merging

### Issue 模板

创建 `.github/ISSUE_TEMPLATE/` 目录并添加：

1. **bug_report.md** - Bug 报告模板
2. **feature_request.md** - 功能请求模板
3. **question.md** - 问题咨询模板

### Pull Request 模板

创建 `.github/pull_request_template.md`

## 镜像仓库配置

### GitHub Container Registry (GHCR)

**优势**:
- 与 GitHub 深度集成
- 免费且无限制
- 自动权限管理
- 支持多平台镜像

**配置**:
1. 启用 GHCR: Settings → Packages → Container registry
2. 设置可见性: Public
3. 配置访问权限: 继承仓库权限

### Docker Hub（可选）

**优势**:
- 更广泛的用户基础
- 更好的发现性
- 官方镜像认证

**配置**:
1. 创建 Docker Hub 仓库
2. 添加 README 和描述
3. 配置自动构建（可选）

## 版本管理策略

### 语义化版本

遵循 [Semantic Versioning 2.0.0](https://semver.org/)：

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

### 标签策略

- `v1.0.0` - 完整版本号
- `v1.0` - 次版本号
- `v1` - 主版本号
- `latest` - 最新稳定版
- `main` - 主分支最新构建
- `develop` - 开发分支最新构建

### 发布周期

- **主版本**: 每年 1-2 次
- **次版本**: 每季度 1 次
- **补丁版本**: 按需发布
- **预览版本**: 每月 1 次

## 许可证

项目使用 **MIT License**，确保：

1. LICENSE 文件已添加
2. 所有源文件包含版权声明
3. 第三方依赖许可证兼容

## 安全策略

创建 `SECURITY.md` 文件：

```markdown
# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

Please report security vulnerabilities to security@example.com
```

## 贡献指南

创建 `CONTRIBUTING.md` 文件，包含：

1. 如何报告 Bug
2. 如何提出功能建议
3. 如何提交 Pull Request
4. 代码规范
5. 提交信息规范

## 行为准则

创建 `CODE_OF_CONDUCT.md` 文件，采用：

- [Contributor Covenant](https://www.contributor-covenant.org/)

## 检查清单

发布前最终检查：

- [ ] 所有测试通过
- [ ] 文档完整且准确
- [ ] Docker 镜像构建成功
- [ ] 示例配置可用
- [ ] 安全漏洞已修复
- [ ] 性能测试通过
- [ ] 许可证正确
- [ ] 版本号正确
- [ ] CHANGELOG 更新
- [ ] Release Notes 准备好

## 发布公告

发布后在以下平台发布公告：

1. **GitHub Discussions** - 项目公告
2. **Twitter/X** - 社交媒体
3. **Reddit** - r/devops, r/golang, r/vuejs
4. **Hacker News** - Show HN
5. **Dev.to** - 技术博客
6. **Medium** - 详细介绍文章

## 示例发布公告

```markdown
# 🎉 qwq AIOps Platform v1.0.0 发布！

我们很高兴地宣布 qwq AIOps Platform v1.0.0 正式发布！

## 🚀 主要特性

- 🤖 AI 驱动的智能运维
- 🎨 现代化的用户界面
- 🔒 企业级安全和权限
- 📊 智能监控和告警
- 🌐 国际化支持

## 📦 快速开始

\`\`\`bash
docker pull ghcr.io/your-org/qwq-aiops:latest
docker run -d -p 8080:8080 ghcr.io/your-org/qwq-aiops:latest
\`\`\`

## 📚 文档

- [用户手册](https://github.com/your-org/qwq-aiops/blob/main/docs/user-manual.md)
- [部署指南](https://github.com/your-org/qwq-aiops/blob/main/docs/deployment-guide.md)
- [API 文档](https://github.com/your-org/qwq-aiops/blob/main/docs/api-integration-complete.md)

## 🙏 致谢

感谢所有贡献者和支持者！

---

⭐ 如果你喜欢这个项目，请给我们一个 Star！
```

---

**准备完成！准备发布 qwq AIOps Platform v1.0.0！** 🚀
