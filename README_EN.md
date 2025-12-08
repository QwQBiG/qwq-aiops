# 🤖 qwq - Enterprise AIOps Platform

<div align="center">

**Private AI Operations Platform | Intelligent Operations · App Store · Container Orchestration · Website Management · Database Management**

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-cyan.svg)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue-3.x-brightgreen.svg)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-Supported-blue.svg)](https://www.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/QwQBiG/qwq-aiops)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue.svg)](https://github.com/QwQBiG/qwq-aiops/releases)
[![Status](https://img.shields.io/badge/status-production--ready-success.svg)](docs/production-readiness-checklist.md)
[![Test Coverage](https://img.shields.io/badge/property--tests-13%20core%20%7C%2096%2B%20sub-success.svg)](docs/project-completion-summary.md)

English | [简体中文](README.md)

</div>

## 📖 Table of Contents

- [Introduction](#-introduction)
- [Core Features](#-core-features)
- [Quick Start](#-quick-start)
- [Feature Details](#-feature-details)
- [Technical Architecture](#-technical-architecture)
- [Quality Assurance](#-quality-assurance)
- [Documentation](#-documentation)
- [Contributing](#-contributing)
- [License](#-license)

## 🎯 Introduction

**qwq** is a modern enterprise-level AIOps intelligent operations platform designed to surpass traditional operations panels by providing a perfect fusion of **"AI + Traditional Operations"**.

### Core Advantages

🤖 **AI-Driven**: Leverages Large Language Model (LLM) reasoning capabilities to transform operations work into natural language interactions  
🔒 **Data Security**: Supports cloud APIs (OpenAI/SiliconFlow) or local private models (Ollama/DeepSeek), ensuring data stays within your domain  
🚀 **All-in-One Management**: App store, container orchestration, website management, database management, monitoring and alerting all in one place  
⚡ **High Availability Architecture**: Supports cluster deployment, load balancing, and failover to ensure business continuity  
🎨 **Modern Interface**: Vue 3 + Element Plus with Chinese and English internationalization support  
✅ **Production Ready**: Comprehensive property testing (13 core properties, 96+ sub-properties) ensures code quality

### Project Status

| Metric | Status |
|--------|--------|
| **Version** | v1.0.0 |
| **Release Date** | 2025-12-07 |
| **Development Status** | ✅ Production Ready |
| **Feature Completion** | 100% (10 major modules) |
| **Test Coverage** | 13 core properties, 96+ sub-properties |
| **Performance Grade** | A |
| **Security Grade** | A- |
| **Documentation** | 25+ detailed documents |

### Comparison with 1Panel

| Feature | qwq | 1Panel |
|---------|-----|--------|
| AI Intelligent Operations | ✅ Natural language interaction | ❌ |
| AI App Recommendations | ✅ Smart recommendations | ❌ |
| AI Architecture Optimization | ✅ Automatic analysis | ❌ |
| AI Query Optimization | ✅ SQL optimization suggestions | ❌ |
| App Store | ✅ | ✅ |
| Container Management | ✅ Docker Compose | ✅ |
| Website Management | ✅ Nginx + SSL | ✅ |
| Database Management | ✅ Multi-database support | ✅ |
| Monitoring & Alerting | ✅ AI predictive analysis | ✅ Basic monitoring |
| Cluster Deployment | ✅ High availability architecture | ❌ |
| Property Testing | ✅ 13 core properties | ❌ |

## 🚀 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 8GB+ RAM
- 20GB+ Disk Space

### Method 1: One-Click Deployment (Recommended) ⭐

```bash
# 1. Clone the repository
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. Run one-click deployment script (will automatically configure AI service)
chmod +x 一键部署.sh
sudo ./一键部署.sh

# 3. Follow prompts to select AI service type
# Option 1: OpenAI API (requires API Key)
# Option 2: Ollama local model (free)
# Option 3: Skip configuration (configure manually later)

# 4. Access the system
# Frontend: http://localhost:8081
# API Docs: http://localhost:8081/api/docs
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3000
# Default credentials: admin / admin123
```

**The script will automatically:**
- ✅ Configure AI service (OpenAI or Ollama)
- ✅ Configure Docker registry mirrors (speed up downloads)
- ✅ Create required configuration files
- ✅ Build Docker images
- ✅ Start all services
- ✅ Verify service status

### Method 2: Manual .env Configuration (For Advanced Users)

```bash
# 1. Clone the repository
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. Copy and edit environment variables file
cp .env.example .env
nano .env  # or use any editor

# 3. Configure AI service (required, choose one)
# Option 1: OpenAI API
# AI_PROVIDER=openai
# OPENAI_API_KEY=sk-your-api-key-here
# OPENAI_BASE_URL=https://api.openai.com/v1
# OPENAI_MODEL=gpt-3.5-turbo

# Option 2: Ollama local model
# AI_PROVIDER=ollama
# OLLAMA_HOST=http://localhost:11434
# OLLAMA_MODEL=qwen2.5:7b

# 4. Uncomment one option (remove # at line start) and fill in correct values

# 5. Start services
docker compose up -d --build

# 6. Access the system
# Frontend: http://localhost:8081
```

**Notes**:
- ⚠️ Must configure AI service, otherwise it won't start
- Remove `#` and space at line start when uncommenting
- OpenAI requires valid API Key
- Ollama requires installation and running service

### Method 3: Configuration Script + Docker Compose

```bash
# 1. Configure AI service (using interactive script)
chmod +x 配置AI服务.sh
./配置AI服务.sh

# 2. Build and start all services (first run takes 6-10 minutes)
docker compose up -d --build

# 3. View logs
docker compose logs -f qwq

# 4. Stop services
docker compose down

# 5. Access the system
# Frontend: http://localhost:8081
# API Docs: http://localhost:8081/api/docs

# Note:
# - Use docker compose (V2, no hyphen) instead of docker-compose (V1)
# - Port changed to 8081 (to avoid conflicts with other services)
```

### Manual Build

```bash
# Build backend
go build -o qwq cmd/qwq/main.go

# Build frontend
cd frontend
npm install
npm run build

# Run
./qwq
```

For detailed deployment instructions, please refer to the [Deployment Guide](docs/deployment-guide.md).

---

## ✨ Core Features

### 🧠 1. Intelligent Interaction (Chat Mode)
*   **Natural Language Operations**: Direct dialogue like "Check the process with highest CPU" or "Analyze why K8s Pod is crashing"
*   **ReAct Reasoning Engine**: AI automatically breaks down tasks (e.g., check PID -> check start time -> analyze logs), supports multi-step execution
*   **Web/CLI Dual Interface**: Supports terminal command-line interaction and web-based real-time dialogue

### 🚨 2. Automated Patrol (Patrol Mode)
*   **Deep Health Checks**: Runs silently in the background, checking disk, load, OOM, and zombie processes every 5 minutes
*   **Intelligent Root Cause Analysis**: When anomalies are detected, AI automatically analyzes causes and provides fix suggestions (e.g., automatically identifies that zombie processes need parent process termination)
*   **Custom Rules**: Supports adding Shell script rules in configuration files (e.g., check Nginx process, Docker container status)

### 📊 3. Visual Console (Web Dashboard)
*   **Modern UI**: Brand new Vue 3 + Element Plus interface with Chinese/English switching
*   **Real-time Monitoring**: ECharts-based CPU, memory, disk real-time trend charts
*   **Application Monitoring**: Built-in HTTP monitoring for real-time business website/API connectivity checks
*   **Real-time Logs**: Real-time backend log push via WebSocket

### 🏪 4. App Store
*   **One-Click Deployment**: Pre-configured templates for MySQL, Redis, Nginx, GitLab, and other common applications
*   **Docker Compose Support**: Visual orchestration of multi-container applications
*   **AI Recommendations**: Smart application combination recommendations based on usage scenarios
*   **Version Management**: Supports application updates and rollbacks

### 🌐 5. Website Management
*   **Reverse Proxy**: Automatically generates Nginx configurations with load balancing support
*   **SSL Certificates**: Integrated Let's Encrypt for automatic certificate application and renewal
*   **DNS Management**: Supports Alibaba Cloud and Tencent Cloud DNS record management
*   **AI Optimization**: Automatically detects configuration issues and provides optimization suggestions

### 💾 6. Database Management
*   **Multi-Database Support**: Unified management of MySQL, PostgreSQL, Redis, MongoDB
*   **SQL Editor**: Integrated Monaco Editor with syntax highlighting and auto-completion
*   **AI Query Optimization**: Automatically analyzes slow queries and provides index and optimization suggestions
*   **Automatic Backup**: Supports local, S3, FTP, and other storage backends

### 📈 7. Intelligent Monitoring & Alerting
*   **Custom Metrics**: Supports custom monitoring metrics and aggregation rules
*   **Smart Alerting**: AI noise reduction to minimize alert storms
*   **Predictive Analysis**: Predicts resource usage trends based on historical data
*   **Capacity Planning**: AI analysis and expansion recommendations

### 🔐 8. Enterprise-Grade Security & Permissions
*   **RBAC Permissions**: Complete role and permission management system
*   **Multi-Tenant Isolation**: Supports multi-tenant environments with complete resource isolation
*   **Audit Logs**: Records all operations for compliance auditing
*   **Command Risk Control**: Built-in blacklist (blocks `rm -rf`), high-risk commands require manual confirmation
*   **Data Masking**: Automatically hides sensitive information like IPs and keys in logs

### 🏠 9. Local Models & Knowledge Base (RAG)
*   **Ollama Support**: Perfect integration with DeepSeek, Qwen, and other local models - zero cost, zero leakage
*   **Private Knowledge Base**: Mount `docs.txt` to teach AI your company's specific operations knowledge (e.g., service restart procedures)

### ⚡ 10. High Availability Architecture
*   **Cluster Deployment**: Supports multi-node clusters with automatic load balancing
*   **Health Checks**: Automatically detects node status with automatic failover
*   **Zero-Downtime Upgrades**: Supports rolling updates without affecting business
*   **Container Self-Healing**: Automatically restarts abnormal containers to ensure service stability

---

## 🏗️ Technical Architecture

### Technology Stack

**Backend**
- Language: Go 1.23+
- Web Framework: Gin
- Database: SQLite / PostgreSQL
- Cache: Redis
- Container: Docker + Docker Compose
- Testing: gopter (Property-Based Testing)

**Frontend**
- Framework: Vue 3 + TypeScript
- UI Library: Element Plus
- State Management: Pinia
- Charts: ECharts
- Editor: Monaco Editor
- Terminal: Xterm.js

**AI Technology**
- LLM Integration: OpenAI API / Ollama
- Reasoning Mode: ReAct (Reasoning + Acting)
- Tool Calling: Function Calling
- Vector Database: Chroma / Qdrant (Optional)

### Architecture Diagram

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

### Code Structure

```
qwq/
├── cmd/                  # Application entry
│   └── qwq/              # Main program
├── internal/             # Internal packages
│   ├── aiagent/          # AI intelligent service
│   ├── appstore/         # App store
│   ├── container/        # Container management
│   ├── website/          # Website management
│   ├── dbmanager/        # Database management
│   ├── backup/           # Backup & recovery
│   ├── monitoring/       # Monitoring & alerting
│   ├── cluster/          # Cluster management
│   ├── security/         # Security & permissions
│   └── gateway/          # API gateway
├── frontend/             # Frontend code
│   ├── src/
│   │   ├── views/        # Page components
│   │   ├── components/   # Common components
│   │   └── stores/       # State management
│   └── package.json
├── docs/                 # Documentation
├── deploy.sh             # Deployment script
├── docker-compose.yml    # Docker Compose config
└── README.md
```

---

## ✅ Quality Assurance

### Property-Based Testing

qwq uses property-based testing to ensure the system works correctly in all scenarios. Each property test runs **100 random iterations**, covering normal flows, boundary conditions, and error handling.

#### Test Coverage

| Module | Properties | Sub-Properties | Test Files |
|--------|------------|----------------|------------|
| AI Intelligent Service | 1 | 4 | `aiagent/task_execution_property_test.go` |
| App Store | 2 | 11 | `appstore/*_property_test.go` |
| Container Management | 3 | 25 | `container/*_property_test.go` |
| Website Management | 3 | 39 | `website/*_property_test.go` |
| Security & Permissions | 2 | 15 | `database/rbac_property_test.go`, `security/multi_tenant_property_test.go` |
| Automation Integration | 1 | 2 | `webhook/webhook_property_test.go` |
| High Availability | 1 | 6 | `registry/service_discovery_property_test.go` |
| **Total** | **13** | **96+** | - |

#### Core Properties

1. **AI Task Execution Integrity**: Verifies completeness of AI task planning, configuration generation, and deployment execution
2. **App Installation Conflict Resolution**: Automatically detects and resolves port conflicts and data volume conflicts
3. **AI App Recommendation Relevance**: Recommendation results are relevant to user needs
4. **Docker Compose Parsing Correctness**: Round-trip consistency and validation functionality
5. **Container Service Self-Healing**: Automatic restart of abnormal containers
6. **AI Architecture Optimization Quality**: Provides valuable optimization suggestions
7. **Website Configuration Automation**: Automatically generates Nginx configurations
8. **SSL Certificate Lifecycle Management**: Automatic application, deployment, and renewal
9. **DNS Management Integrity**: Complete DNS record management
10. **User Permission Isolation**: Strict role permission checks
11. **Multi-Tenant Environment Isolation**: Complete resource isolation between different tenants
12. **Automated Task Execution Reliability**: Detailed execution logs and error handling
13. **Cluster Deployment High Availability**: Load balancing, failover, service recovery

### Running Tests

```bash
# Run all property tests
go test ./internal/... -v -run Property

# Run specific module property tests
go test ./internal/appstore -v -run Property
go test ./internal/container -v -run Property
go test ./internal/website -v -run Property

# View test coverage
go test ./internal/... -cover
```

### Performance Metrics

- **API Response Time**: < 100ms (P95)
- **Concurrent Processing**: 1000+ QPS
- **Memory Usage**: < 512MB (idle)
- **Startup Time**: < 5s

### Security Audit

- ✅ SQL Injection Protection
- ✅ XSS Protection
- ✅ CSRF Protection
- ✅ Command Injection Protection
- ✅ Sensitive Data Masking
- ✅ Audit Log Recording

For detailed security audit reports, please refer to [Security Audit Report](docs/security-audit-report.md).

---

## 📚 Documentation

### User Documentation

- [User Manual](docs/user-manual.md) - Complete feature usage guide
- [Deployment Guide](docs/deployment-guide.md) - Detailed deployment and configuration instructions
- [Troubleshooting Guide](docs/troubleshooting-guide.md) - Common issues and solutions
- [API Documentation](http://localhost:8080/api/docs) - Interactive API documentation (Swagger UI)

### Development Documentation

- [Project Summary](docs/project-completion-summary.md) - Project completion overview
- [Release Notes](docs/release-notes-v1.0.md) - v1.0 release notes
- [Performance Optimization Report](docs/performance-optimization-report.md) - Performance optimization details
- [Security Audit Report](docs/security-audit-report.md) - Security audit results

### Technical Documentation

- [AI Architecture Optimizer](docs/ai-architecture-optimizer.md) - AI architecture analysis implementation
- [AI Recommendation System](docs/ai-recommendation-system.md) - AI app recommendation implementation
- [Container Self-Healing System](docs/container-self-healing-system.md) - Container self-healing mechanism
- [App Store Template System](docs/appstore-template-system.md) - Application template design

---

## 🤝 Contributing

We welcome all forms of contributions!

### How to Contribute

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go code conventions (`gofmt`, `golint`)
- Add property tests for new features
- Update relevant documentation
- Ensure all tests pass

### Reporting Issues

If you find a bug or have a feature suggestion, please [create an Issue](https://github.com/QwQBiG/qwq-aiops/issues).

---

## 🌟 Star History

If this project helps you, please give us a Star ⭐️

---

## 📄 License

MIT License. Copyright (c) 2025 qwqBig.
