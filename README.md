# 🤖 qwq - Enterprise AIOps Platform

<div align="center">

**私有化 AI 运维平台 | 智能运维 · 应用商店 · 容器编排 · 网站管理 · 数据库管理**

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-cyan.svg)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue-3.x-brightgreen.svg)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-Supported-blue.svg)](https://www.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/QwQBiG/qwq-aiops)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue.svg)](https://github.com/QwQBiG/qwq-aiops/releases)
[![Status](https://img.shields.io/badge/status-production--ready-success.svg)](docs/production-readiness-checklist.md)
[![Test Coverage](https://img.shields.io/badge/property--tests-13%20core%20%7C%2096%2B%20sub-success.svg)](docs/project-completion-summary.md)

[English](README_EN.md) | 简体中文

</div>

## 📖 目录

- [项目简介](#-项目简介)
- [核心特性](#-核心特性)
- [快速开始](#-快速开始)
- [功能详解](#-功能详解)
- [技术架构](#-技术架构)
- [质量保证](#-质量保证)
- [文档资源](#-文档资源)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

## 🎯 项目简介

**qwq** 是一个现代化的企业级 AIOps 智能运维平台，旨在超越传统运维面板提供 **"AI + 传统运维"** 的完美融合体验。

### 核心优势

🤖 **AI 驱动**：利用大语言模型（LLM）的推理能力，将运维工作转化为自然语言交互  
🔒 **数据安全**：支持云端 API（OpenAI/硅基流动）或本地私有模型（Ollama/DeepSeek），确保数据不出域  
🚀 **一站式管理**：应用商店、容器编排、网站管理、数据库管理、监控告警一应俱全  
⚡ **高可用架构**：支持集群部署、负载均衡、故障转移，保障业务连续性  
🎨 **现代化界面**：Vue 3 + Element Plus，支持中英文国际化  
✅ **生产就绪**：通过全面的属性测试（13 个核心属性，96+ 子属性），代码质量有保障

### 项目状态

| 指标 | 状态 |
|------|------|
| **版本** | v1.0.0 |
| **发布日期** | 2025-12-07 |
| **开发状态** | ✅ 生产就绪 |
| **功能完成度** | 100% (10 个主要模块) |
| **测试覆盖** | 13 个核心属性，96+ 子属性 |
| **性能等级** | A 级 |
| **安全等级** | A- 级 |
| **文档完整度** | 25+ 个详细文档 |

### 与 1Panel 的对比

| 功能 | qwq | 1Panel |
|------|-----|--------|
| AI 智能运维 | ✅ 自然语言交互 | ❌ |
| AI 应用推荐 | ✅ 智能推荐 | ❌ |
| AI 架构优化 | ✅ 自动分析 | ❌ |
| AI 查询优化 | ✅ SQL 优化建议 | ❌ |
| 应用商店 | ✅ | ✅ |
| 容器管理 | ✅ Docker Compose | ✅ |
| 网站管理 | ✅ Nginx + SSL | ✅ |
| 数据库管理 | ✅ 多数据库支持 | ✅ |
| 监控告警 | ✅ AI 预测分析 | ✅ 基础监控 |
| 集群部署 | ✅ 高可用架构 | ❌ |
| 属性测试 | ✅ 13 个核心属性 | ❌ |

## 🚀 快速开始

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- 8GB+ 内存
- 20GB+ 磁盘空间

### 方式一：一键部署（推荐）⭐

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 运行一键部署脚本（会自动配置 AI 服务）
chmod +x 一键部署.sh
sudo ./一键部署.sh

# 3. 按提示选择 AI 服务类型
# 选项 1: OpenAI API（需要 API Key）
# 选项 2: Ollama 本地模型（免费）
# 选项 3: 跳过配置（稍后手动配置）

# 4. 访问系统
# 前端界面: http://localhost:8081
# API 文档: http://localhost:8081/api/docs
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3000
# 默认账号: admin / admin123
```

**脚本会自动完成：**
- ✅ 配置 AI 服务（OpenAI 或 Ollama）
- ✅ 配置 Docker 国内镜像源（加速下载）
- ✅ 创建所需的配置文件
- ✅ 构建 Docker 镜像
- ✅ 启动所有服务
- ✅ 验证服务状态

### 方式二：手动配置 .env（推荐熟悉配置的用户）

```bash
# 1. 克隆项目
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. 复制并编辑环境变量文件
cp .env.example .env
nano .env  # 或使用其他编辑器

# 3. 配置 AI 服务（必须配置，二选一）
# 方式 1: OpenAI API
# AI_PROVIDER=openai
# OPENAI_API_KEY=sk-your-api-key-here
# OPENAI_BASE_URL=https://api.openai.com/v1
# OPENAI_MODEL=gpt-3.5-turbo

# 方式 2: Ollama 本地模型
# AI_PROVIDER=ollama
# OLLAMA_HOST=http://localhost:11434
# OLLAMA_MODEL=qwen2.5:7b

# 4. 取消某一种方式的注释（删除行首的 #）并填入正确的值

# 5. 启动服务
docker compose up -d --build

# 6. 访问系统
# 前端界面: http://localhost:8081
```

**注意**：
- ⚠️ 必须配置 AI 服务，否则无法启动
- 取消注释时删除行首的 `#` 和空格
- OpenAI 需要有效的 API Key
- Ollama 需要先安装并启动服务

### 方式三：使用配置脚本 + Docker Compose

```bash
# 1. 配置 AI 服务（使用交互式脚本）
chmod +x 配置AI服务.sh
./配置AI服务.sh

# 2. 构建并启动所有服务（首次运行，约 6-10 分钟）
docker compose up -d --build

# 3. 查看日志
docker compose logs -f qwq

# 4. 停止服务
docker compose down

# 5. 访问系统
# 前端界面: http://localhost:8081
# API 文档: http://localhost:8081/api/docs

# 注意：
# - 使用 docker compose（V2，无连字符）而不是 docker-compose（V1）
# - 端口已修改为 8081（避免与其他服务冲突）
```

### 手动编译

```bash
# 后端编译
go build -o qwq cmd/qwq/main.go

# 前端编译
cd frontend
npm install
npm run build

# 运行
./qwq
```

详细部署说明请参考 [部署指南](docs/deployment-guide.md)。

---

## ✨ 核心特性

### 🧠 1. 智能交互 (Chat Mode)
*   **自然语言运维**：直接对话 "帮我查一下 CPU 最高的进程" 或 "分析 K8s Pod 为什么 Crash"。
*   **ReAct 推理引擎**：AI 自动拆解任务（如：查 PID -> 查启动时间 -> 分析日志），支持多步执行。
*   **Web/CLI 双端**：支持终端命令行交互，也支持 Web 网页端实时对话。

### 🚨 2. 全自动巡检 (Patrol Mode)
*   **深度健康检查**：后台静默运行，每 5 分钟检测磁盘、负载、OOM 及僵尸进程。
*   **智能根因分析**：发现异常后，AI 自动分析原因并给出修复建议（如自动识别僵尸进程需杀父进程）。
*   **自定义规则**：支持在配置文件中添加 Shell 脚本规则（如检查 Nginx 进程、Docker 容器状态）。

### 📊 3. 可视化控制台 (Web Dashboard)
*   **现代化 UI**：全新 Vue 3 + Element Plus 界面，支持中英文切换。
*   **实时监控**：基于 ECharts 的 CPU、内存、磁盘实时趋势图。
*   **应用拨测**：内置 HTTP 监控，实时检测业务网站/API 连通性。
*   **实时日志**：通过 WebSocket 实时推送后台运行日志。

### 🏪 4. 应用商店 (App Store)
*   **一键部署**：预置 MySQL、Redis、Nginx、GitLab 等常用应用模板。
*   **Docker Compose 支持**：可视化编排多容器应用。
*   **AI 推荐**：根据使用场景智能推荐应用组合。
*   **版本管理**：支持应用更新和回滚。

### 🌐 5. 网站管理 (Website Management)
*   **反向代理**：自动生成 Nginx 配置，支持负载均衡。
*   **SSL 证书**：集成 Let's Encrypt，自动申请和续期证书。
*   **DNS 管理**：支持阿里云、腾讯云 DNS 记录管理。
*   **AI 优化**：自动检测配置问题并提供优化建议。

### 💾 6. 数据库管理 (Database Management)
*   **多数据库支持**：MySQL、PostgreSQL、Redis、MongoDB 统一管理。
*   **SQL 编辑器**：集成 Monaco Editor，支持语法高亮和自动补全。
*   **AI 查询优化**：自动分析慢查询，提供索引和优化建议。
*   **自动备份**：支持本地、S3、FTP 等多种存储后端。

### 📈 7. 智能监控告警 (Monitoring & Alerting)
*   **自定义指标**：支持自定义监控指标和聚合规则。
*   **智能告警**：AI 降噪，减少告警风暴。
*   **预测分析**：基于历史数据预测资源使用趋势。
*   **容量规划**：AI 分析并提供扩容建议。

### 🔐 8. 企业级安全与权限
*   **RBAC 权限**：完整的角色和权限管理体系。
*   **多租户隔离**：支持多租户环境，资源完全隔离。
*   **审计日志**：记录所有操作，支持合规审计。
*   **命令风控**：内置黑名单（拦截 `rm -rf`），高危命令需人工确认。
*   **数据脱敏**：自动隐藏日志中的 IP、密钥等敏感信息。

### 🏠 9. 本地模型与知识库 (RAG)
*   **Ollama 支持**：完美适配 DeepSeek、Qwen 等本地模型，零成本、零泄露。
*   **私有知识库**：挂载 `docs.txt`，让 AI 学会你们公司的特定运维知识（如服务重启步骤）。

### ⚡ 10. 高可用架构 (High Availability)
*   **集群部署**：支持多节点集群，自动负载均衡。
*   **健康检查**：自动检测节点状态，故障自动转移。
*   **零停机升级**：支持滚动更新，不影响业务。
*   **容器自愈**：自动重启异常容器，保障服务稳定。

---

## 🏗️ 技术架构

### 技术栈

**后端**
- 语言：Go 1.23+
- Web 框架：Gin
- 数据库：SQLite / PostgreSQL
- 缓存：Redis
- 容器：Docker + Docker Compose
- 测试：gopter (Property-Based Testing)

**前端**
- 框架：Vue 3 + TypeScript
- UI 库：Element Plus
- 状态管理：Pinia
- 图表：ECharts
- 编辑器：Monaco Editor
- 终端：Xterm.js

**AI 技术**
- LLM 接入：OpenAI API / Ollama
- 推理模式：ReAct (Reasoning + Acting)
- 工具调用：Function Calling
- 向量数据库：Chroma / Qdrant（可选）

### 架构图

```
┌───────────────────────────────────────────────────────────────┐
│                    Frontend Layer                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐  │
│  │   Web UI    │ │  Mobile App │ │      CLI Tool           │  │
│  │  (Vue 3)    │ │   (React)   │ │     (Cobra)             │  │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────────────┐
                    │   API Gateway   │
                    │   (Gin/Fiber)   │
                    └─────────────────┘
                              │
┌───────────────────────────────────────────────────────────────┐
│                   Core Services Layer                         │
│                                                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐  │
│  │ AI Agent    │ │ App Store   │ │   Container Manager     │  │
│  │ Service     │ │ Service     │ │      Service            │  │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘  │
│                                                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐  │
│  │ Website     │ │ Database    │ │    Backup & Recovery    │  │
│  │ Manager     │ │ Manager     │ │       Service           │  │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘  │
│                                                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐  │
│  │ User & Auth │ │ Monitoring  │ │    Notification         │  │
│  │ Service     │ │ Service     │ │       Service           │  │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
                              │
┌───────────────────────────────────────────────────────────────┐
│                Infrastructure Layer                           │
│                                                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐  │
│  │   Docker    │ │ Kubernetes  │ │      File System        │  │
│  │   Engine    │ │   Cluster   │ │       Storage           │  │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### 代码结构

```
qwq/
├── cmd/                  # 应用入口
│   └── qwq/              # 主程序
├── internal/             # 内部包
│   ├── aiagent/          # AI 智能服务
│   ├── appstore/         # 应用商店
│   ├── container/        # 容器管理
│   ├── website/          # 网站管理
│   ├── dbmanager/        # 数据库管理
│   ├── backup/           # 备份恢复
│   ├── monitoring/       # 监控告警
│   ├── cluster/          # 集群管理
│   ├── security/         # 安全权限
│   └── gateway/          # API 网关
├── frontend/             # 前端代码
│   ├── src/
│   │   ├── views/        # 页面组件
│   │   ├── components/   # 通用组件
│   │   └── stores/       # 状态管理
│   └── package.json
├── docs/                 # 文档
├── deploy.sh             # 部署脚本
├── docker-compose.yml    # Docker Compose 配置
└── README.md
```

---

## ✅ 质量保证

### 属性测试（Property-Based Testing）

qwq 采用属性测试方法，确保系统在各种情况下都能正确工作。每个属性测试运行 **100 次随机迭代**，覆盖正常流程、边界条件和错误处理。

#### 测试覆盖

| 模块 | 属性数 | 子属性数 | 测试文件 |
|------|--------|----------|----------|
| AI 智能服务 | 1 | 4 | `aiagent/task_execution_property_test.go` |
| 应用商店 | 2 | 11 | `appstore/*_property_test.go` |
| 容器管理 | 3 | 25 | `container/*_property_test.go` |
| 网站管理 | 3 | 39 | `website/*_property_test.go` |
| 权限安全 | 2 | 15 | `database/rbac_property_test.go`, `security/multi_tenant_property_test.go` |
| 自动化集成 | 1 | 2 | `webhook/webhook_property_test.go` |
| 高可用性 | 1 | 6 | `registry/service_discovery_property_test.go` |
| **总计** | **13** | **96+** | - |

#### 核心属性

1. **AI 任务执行完整性**：验证 AI 任务规划、配置生成、部署执行的完整性
2. **应用安装冲突解决**：自动检测并解决端口冲突、数据卷冲突
3. **AI 应用推荐相关性**：推荐结果与用户需求相关
4. **Docker Compose 解析正确性**：往返一致性、验证功能
5. **容器服务自愈能力**：异常容器自动重启
6. **AI 架构优化建议质量**：提供有价值的优化建议
7. **网站配置自动化**：自动生成 Nginx 配置
8. **SSL 证书生命周期管理**：自动申请、部署和续期
9. **DNS 管理完整性**：完整的 DNS 记录管理
10. **用户权限隔离**：严格的角色权限检查
11. **多租户环境隔离**：不同租户资源完全隔离
12. **自动化任务执行可靠性**：详细的执行日志和错误处理
13. **集群部署高可用性**：负载均衡、故障转移、服务恢复

### 运行测试

```bash
# 运行所有属性测试
go test ./internal/... -v -run Property

# 运行特定模块的属性测试
go test ./internal/appstore -v -run Property
go test ./internal/container -v -run Property
go test ./internal/website -v -run Property

# 查看测试覆盖率
go test ./internal/... -cover
```

### 性能指标

- **API 响应时间**: < 100ms (P95)
- **并发处理能力**: 1000+ QPS
- **内存占用**: < 512MB (空闲)
- **启动时间**: < 5s

### 安全审计

- ✅ SQL 注入防护
- ✅ XSS 防护
- ✅ CSRF 防护
- ✅ 命令注入防护
- ✅ 敏感数据脱敏
- ✅ 审计日志记录

详细的安全审计报告请参考 [安全审计报告](docs/security-audit-report.md)。

---

## 📚 文档资源

### 用户文档

- [用户手册](docs/user-manual.md) - 完整的功能使用指南
- [部署指南](docs/deployment-guide.md) - 详细的部署和配置说明
- [故障排查指南](docs/troubleshooting-guide.md) - 常见问题和解决方案
- [API 文档](http://localhost:8080/api/docs) - 交互式 API 文档（Swagger UI）

### 开发文档

- [项目总结](docs/project-completion-summary.md) - 项目完成情况总览
- [发布说明](docs/release-notes-v1.0.md) - v1.0 版本发布说明
- [性能优化报告](docs/performance-optimization-report.md) - 性能优化详情
- [安全审计报告](docs/security-audit-report.md) - 安全审计结果

### 技术文档

- [AI 架构优化器](docs/ai-architecture-optimizer.md) - AI 架构分析实现
- [AI 推荐系统](docs/ai-recommendation-system.md) - AI 应用推荐实现
- [容器自愈系统](docs/container-self-healing-system.md) - 容器自愈机制
- [应用商店模板系统](docs/appstore-template-system.md) - 应用模板设计

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 如何贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 开发规范

- 遵循 Go 代码规范（`gofmt`, `golint`）
- 为新功能添加属性测试
- 更新相关文档
- 确保所有测试通过

### 报告问题

如果您发现 bug 或有功能建议，请[创建 Issue](https://github.com/QwQBiG/qwq-aiops/issues)。

---

## 🌟 Star History

如果这个项目对您有帮助，请给我们一个 Star ⭐️

---

## 📄 License
MIT License. Copyright (c) 2025 qwqBig.