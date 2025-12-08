# qwq AIOps - AI-Powered Intelligent Operations Platform

<div align="center">

![qwq AIOps](https://img.shields.io/badge/qwq-AIOps-blue?style=for-the-badge)
![Version](https://img.shields.io/badge/version-1.0.0-green?style=for-the-badge)
![License](https://img.shields.io/badge/license-MIT-orange?style=for-the-badge)

English | **[简体中文](./README.md)**

A modern AI-powered intelligent operations platform providing container management, system monitoring, and automated operations

[Quick Start](#quick-start) • [Features](#features) • [Deployment](#deployment) • [Documentation](#documentation)

</div>

---

## 📖 Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Deployment Guide](#deployment-guide)
- [Configuration](#configuration)
- [Documentation](#documentation)
- [Development](#development)
- [FAQ](#faq)
- [Contributing](#contributing)
- [License](#license)

---

## 🎯 Introduction

qwq AIOps is a modern intelligent operations platform that combines AI technology with traditional DevOps tools, providing enterprises with:

- 🤖 **AI-Driven Analysis** - Automatically analyze system anomalies and provide solutions
- 🐳 **Container Management** - Complete Docker container lifecycle management
- 📊 **Real-time Monitoring** - Monitor system resources, service status, and performance metrics
- 🔔 **Smart Alerts** - Multi-channel notifications (DingTalk, WeChat, Email, etc.)
- 🚀 **One-Click Deployment** - Complete automation scripts, get started in 5 minutes
- 🌐 **Modern UI** - Responsive web interface built with Vue 3 + Element Plus

---

## ✨ Features

### Core Capabilities

| Module | Description | Status |
|--------|-------------|--------|
| 🎛️ **System Monitoring** | Real-time CPU, memory, disk, network monitoring | ✅ Complete |
| 🐳 **Container Management** | Docker container start, stop, restart, log viewing | ✅ Complete |
| 🌐 **Website Monitoring** | HTTP/HTTPS health checks and response time monitoring | ✅ Complete |
| 💾 **Database Management** | MySQL, PostgreSQL, Redis management | ✅ Complete |
| 📦 **App Store** | One-click deployment of common apps (WordPress, MySQL, etc.) | ✅ Complete |
| 📁 **File Management** | Online file browsing, editing, upload/download | ✅ Complete |
| 💬 **AI Terminal** | Intelligent CLI assistant, natural language operations | ✅ Complete |
| 📊 **Visualization** | Prometheus + Grafana integration | ✅ Complete |
| 🔔 **Alert Notifications** | DingTalk, WeChat, Slack, Email multi-channel | ✅ Complete |
| 👥 **Multi-tenancy** | Tenant isolation, permission management | ✅ Complete |

### AI Capabilities

- **Intelligent Anomaly Analysis** - Automatically analyze system logs and metrics
- **Solution Recommendations** - Provide fixes based on historical data and best practices
- **Natural Language Interaction** - Execute operations commands through conversation
- **Automated Script Generation** - Generate operations scripts based on requirements

### Supported AI Services

- ✅ **OpenAI** (GPT-3.5/GPT-4)
- ✅ **Ollama** (Local deployment, completely free)
- ✅ **Custom API** (OpenAI-compatible format)

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend Layer (Vue 3)                   │
│  Element Plus UI • Vue Router • Pinia • ECharts • Axios     │
└─────────────────────────────────────────────────────────────┘
                              ↓ HTTP/WebSocket
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway Layer (Go)                   │
│      Routing • Auth • Rate Limiting • Logging • Errors      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                   Business Logic Layer (Go)                 │
│  Container Mgmt • Monitoring • AI Analysis • Alerts • Jobs  │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Data Storage Layer                     │
│    SQLite/MySQL • Redis • Prometheus • File System          │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                     │
│         Docker • Kubernetes • Linux • Cloud Providers       │
└─────────────────────────────────────────────────────────────┘
```

### Tech Stack

**Backend**
- Go 1.23+ - High-performance backend service
- Gin - Web framework
- GORM - ORM framework
- Docker SDK - Container management
- Prometheus Client - Metrics collection

**Frontend**
- Vue 3 - Progressive frontend framework
- Element Plus - UI component library
- ECharts - Data visualization
- Vite - Build tool
- Pinia - State management

**Infrastructure**
- Docker & Docker Compose - Containerized deployment
- Prometheus - Monitoring data collection
- Grafana - Visualization dashboard
- MySQL/SQLite - Data storage
- Redis - Cache and queue

---

## 🚀 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose V2
- 2GB+ available memory
- 10GB+ available disk space

### One-Click Deployment

```bash
# 1. Clone the repository
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. Configure environment variables
cp .env.example .env
nano .env  # Edit configuration file

# 3. Run deployment script
chmod +x deploy.sh
./deploy.sh
```

The deployment script will automatically:
- ✅ Check environment (Docker, ports, disk space)
- ✅ Validate configuration (AI service, database, etc.)
- ✅ Build images (frontend + backend)
- ✅ Start services (all containers)
- ✅ Health check (ensure services are running)

### Access Services

After successful deployment, access:

| Service | URL | Default Credentials |
|---------|-----|-------------------|
| 🎛️ **Main Console** | http://localhost:8081 | - |
| 📊 **Prometheus** | http://localhost:9091 | - |
| 📈 **Grafana** | http://localhost:3000 | admin / admin |

---

## 📦 Deployment Guide

### Method 1: Docker Compose (Recommended)

Suitable for quick experience and small-scale deployment.

```bash
# Full deployment (all services)
./deploy.sh

# Quick rebuild (after code updates)
./rebuild.sh

# View logs
docker compose logs -f qwq

# Stop services
docker compose down
```

### Method 2: Manual Deployment

Suitable for custom configuration and production environments.

```bash
# 1. Build frontend
cd frontend
npm install
npm run build

# 2. Build backend
cd ..
go mod download
go build -o qwq ./cmd/qwq/main.go

# 3. Run service
./qwq web
```

### Method 3: Kubernetes Deployment

Suitable for large-scale production environments.

```bash
# Deploy using Helm Chart
helm install qwq-aiops ./charts/qwq-aiops

# Or use kubectl
kubectl apply -f k8s/
```

---

## ⚙️ Configuration

### Environment Variables

Edit `.env` file for configuration:

```bash
# ============================================
# Basic Configuration
# ============================================
PORT=8080                    # Service port
ENVIRONMENT=production       # Runtime environment
LOG_LEVEL=info              # Log level

# ============================================
# AI Configuration (Required)
# ============================================

# Option 1: Use OpenAI
AI_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-3.5-turbo

# Option 2: Use Ollama (Recommended, Free)
AI_PROVIDER=ollama
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=qwen2.5:7b

# ============================================
# Notification Configuration
# ============================================
DINGTALK_WEBHOOK=https://oapi.dingtalk.com/robot/send?access_token=xxx
WECHAT_WEBHOOK=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx
SLACK_WEBHOOK=https://hooks.slack.com/services/xxx

# ============================================
# Database Configuration
# ============================================
DB_TYPE=sqlite              # sqlite, mysql, postgresql
DB_PATH=./data/qwq.db      # SQLite database path

# MySQL Configuration (Optional)
# DB_HOST=localhost
# DB_PORT=3306
# DB_USER=qwq
# DB_PASSWORD=your-password
# DB_NAME=qwq

# ============================================
# Security Configuration
# ============================================
JWT_SECRET=change-this-to-random-secret
ENCRYPTION_KEY=change-this-to-32-byte-key

# Web Authentication (Optional)
WEB_USER=admin
WEB_PASSWORD=admin123
```

### Ollama Configuration in Docker

If your Ollama runs in Docker, special configuration is needed:

```bash
# Linux environment
OLLAMA_HOST=http://172.17.0.1:11434  # Docker bridge IP

# Or use host IP
OLLAMA_HOST=http://your-server-ip:11434

# Or add Ollama to the same network
docker network connect qwqops_qwq-network ollama
OLLAMA_HOST=http://ollama:11434
```

---

## 📚 Documentation

### System Monitoring

View real-time system resource usage:

- **CPU Load** - System load average
- **Memory Usage** - Used/Total memory, usage rate
- **Disk Space** - Usage of each partition
- **Network Connections** - TCP connection statistics

### Container Management

Manage Docker containers:

```bash
# Operations in Web UI
1. Go to "Container Management" page
2. View all container statuses
3. Click "Start/Stop/Restart" buttons
4. View container logs
```

### AI Terminal

Execute operations tasks using natural language:

```
You: Check system load
AI: Executing uptime command...
    System uptime: 5 days 3 hours
    Load: 0.5, 0.6, 0.7

You: Restart nginx container
AI: Executing docker restart nginx...
    Container restarted
```

### Alert Configuration

Configure automatic alert rules:

1. Edit `.env` file, configure notification channels
2. System automatically monitors:
   - Disk usage > 85%
   - System load > 4.0
   - Out of memory (OOM)
   - Service anomalies
3. Automatically push notifications when alerts trigger

---

## 🛠️ Development

### Local Development Environment

```bash
# 1. Start backend (development mode)
go run cmd/qwq/main.go web

# 2. Start frontend (development mode)
cd frontend
npm run dev

# 3. Access development servers
# Frontend: http://localhost:5173
# Backend: http://localhost:8080
```

### Project Structure

```
qwq-aiops/
├── cmd/                    # CLI entry points
│   └── qwq/
│       └── main.go       # Main program entry
├── internal/             # Internal packages
│   ├── agent/            # AI agent
│   ├── config/           # Configuration management
│   ├── container/        # Container management
│   ├── gateway/          # API gateway
│   ├── monitor/          # Monitoring collection
│   ├── notify/           # Notification push
│   └── server/           # Web server
├── frontend/             # Frontend project
│   ├── src/
│   │   ├── views/        # Page components
│   │   ├── router/       # Router configuration
│   │   ├── i18n/         # Internationalization
│   │   └── main.js       # Entry file
│   └── vite.config.js    # Vite configuration
├── config/               # Configuration files
│   ├── prometheus.yml    # Prometheus config
│   └── mysql.cnf         # MySQL config
├── docs/                 # Documentation
├── docker-compose.yml    # Docker Compose config
├── Dockerfile            # Docker image build
├── deploy.sh             # Deployment script
├── rebuild.sh            # Rebuild script
└── .env.example          # Environment variables example
```

### Adding New Features

1. **Backend API**
```go
// internal/server/server.go
http.HandleFunc("/api/your-endpoint", basicAuth(handleYourEndpoint))

func handleYourEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation logic
}
```

2. **Frontend Page**
```vue
<!-- frontend/src/views/YourPage.vue -->
<template>
  <div>Your Page Content</div>
</template>

<script setup>
// Page logic
</script>
```

3. **Router Configuration**
```javascript
// frontend/src/router/index.js
{
  path: '/your-page',
  name: 'YourPage',
  component: () => import('../views/YourPage.vue')
}
```

---

## ❓ FAQ

### 1. Port Already in Use

```bash
# Check port usage
lsof -i :8081

# Change port
# Edit docker-compose.yml, modify ports configuration
ports:
  - "8082:8080"  # Change to 8082
```

### 2. AI Service Connection Failed

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Check network connection
docker compose exec qwq ping 172.17.0.1

# View logs
docker compose logs qwq | grep AI
```

### 3. Frontend Page Blank

```bash
# Clear browser cache
Ctrl + Shift + Delete

# Force refresh
Ctrl + F5

# Check console errors
F12 -> Console
```

### 4. Container Build Failed

```bash
# Clean Docker cache
docker system prune -a

# Rebuild
./rebuild.sh

# View build logs
docker compose build --no-cache --progress=plain
```

### 5. Database Connection Failed

```bash
# Check MySQL container status
docker compose ps mysql

# View MySQL logs
docker compose logs mysql

# Reset database
docker compose down -v
docker compose up -d
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!

### Contribution Process

1. Fork this repository
2. Create feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open Pull Request

### Code Standards

- **Go Code** - Follow [Effective Go](https://golang.org/doc/effective_go)
- **Vue Code** - Follow [Vue Style Guide](https://vuejs.org/style-guide/)
- **Commit Messages** - Follow [Conventional Commits](https://www.conventionalcommits.org/)

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Thanks to these open source projects:

- [Vue.js](https://vuejs.org/) - Progressive JavaScript framework
- [Element Plus](https://element-plus.org/) - Vue 3 component library
- [Go](https://golang.org/) - Efficient programming language
- [Docker](https://www.docker.com/) - Containerization platform
- [Prometheus](https://prometheus.io/) - Monitoring system
- [Ollama](https://ollama.ai/) - Local AI model runtime

---

## 📞 Contact

- **Issue Reports**: [GitHub Issues](https://github.com/QwQBiG/qwq-aiops/issues)
- **Feature Requests**: [GitHub Discussions](https://github.com/QwQBiG/qwq-aiops/discussions)
- **Email**: support@qwq-aiops.com

---

<div align="center">

**[⬆ Back to Top](#qwq-aiops---ai-powered-intelligent-operations-platform)**

Made with ❤️ by qwqBiG.

</div>
