# qwq AIOps Platform

<div align="center">

**AI-Powered Intelligent Operations Platform**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![Vue Version](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-20.10+-2496ED?logo=docker)](https://www.docker.com/)

English | [简体中文](README.md)

</div>

## ✨ Features

- 🤖 **AI-Powered Diagnostics** - Intelligent problem analysis and solution recommendations based on OpenAI/Ollama
- 📊 **Real-time Monitoring** - System resources, service status, and container health monitoring
- 🔔 **Smart Alerts** - Automatic anomaly detection with notifications via DingTalk/WeChat/Email
- 🐳 **Container Management** - Docker container start, stop, restart, and log viewing
- 📈 **Visualization Dashboard** - Prometheus + Grafana monitoring visualization
- 🔐 **Secure & Reliable** - JWT authentication, data encryption, access control
- 🚀 **One-Click Deployment** - Docker Compose one-click deployment, ready to use

## 🎯 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose V2
- 4GB+ available memory
- 10GB+ available disk space

### One-Click Deployment

```bash
# 1. Clone the repository
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops

# 2. Configure AI service (Required)
cp .env.example .env
nano .env  # Edit configuration

# 3. Run deployment script
chmod +x deploy.sh
./deploy.sh

# Or use simplified script
chmod +x start.sh
./start.sh
```

**Windows Users**:
```bash
start.bat
```

### AI Configuration

qwq is an AI-driven platform and requires AI service configuration.

#### Option 1: OpenAI API (Recommended for beginners)

Edit `.env` file:

```bash
AI_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-3.5-turbo
```

#### Option 2: Ollama Local Model (Recommended for enterprises)

```bash
# 1. Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. Download model
ollama pull qwen2.5:7b

# 3. Edit .env
AI_PROVIDER=ollama
OLLAMA_HOST=http://host.docker.internal:11434
OLLAMA_MODEL=qwen2.5:7b
```

### Access System

After successful deployment, access:

| Service | URL | Description |
|---------|-----|-------------|
| Web UI | http://localhost:8081 | Main interface |
| API Docs | http://localhost:8081/api/docs | Swagger documentation |
| Prometheus | http://localhost:9091 | Monitoring metrics |
| Grafana | http://localhost:3000 | Visualization dashboard |

**Default Credentials**:
- Username: `admin`
- Password: `admin123`

## 📖 Documentation

- [Quick Start](快速开始.md) - 5-minute quick start guide
- [Deployment Guide](docs/deployment-guide.md) - Detailed deployment instructions
- [User Manual](docs/user-manual.md) - Feature usage guide
- [API Documentation](http://localhost:8081/api/docs) - RESTful API docs
- [FAQ](docs/faq.md) - Frequently asked questions

## 🏗️ Architecture

```
                    ┌─────────────────────────────┐
                    │    Frontend (Vue 3)          │
                    │  Dashboard | Container Mgmt  │
                    │  AI Diag   | Settings         │
                    └─────────────┬───────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────────┐
                    │    Backend API (Go)          │
                    │  Web Server | AI Agent       │
                    │  Monitor    | Alerting       │
                    └─────────────┬───────────────┘
                                  │
            ┌─────────────────────┼─────────────────────┐
            ▼                     ▼                     ▼
    ┌──────────────┐      ┌──────────────┐    ┌──────────────┐
    │    MySQL     │      │    Redis     │    │  Prometheus  │
    │  Data Store  │      │    Cache     │    │   Metrics    │
    └──────────────┘      └──────────────┘    └──────────────┘
```

## 🔧 Tech Stack

### Backend
- **Language**: Go 1.23+
- **Framework**: Cobra (CLI), Gorilla (WebSocket)
- **AI**: OpenAI API / Ollama
- **Database**: SQLite / MySQL / PostgreSQL
- **Cache**: Redis
- **Monitoring**: Prometheus

### Frontend
- **Framework**: Vue 3 + Vite
- **UI**: Element Plus
- **Charts**: ECharts
- **Editor**: Monaco Editor
- **State**: Pinia

### DevOps
- **Container**: Docker + Docker Compose
- **Reverse Proxy**: Nginx (optional)
- **Monitoring**: Prometheus + Grafana

## 📊 Features

### 1. AI Diagnostics
- AI-powered problem analysis
- Automatic solution generation
- Command execution suggestions
- Knowledge base integration

### 2. System Monitoring
- Real-time CPU, memory, disk monitoring
- Load and network connection monitoring
- Custom monitoring rules
- Historical data queries

### 3. Container Management
- Docker container listing
- Container start/stop/restart
- Real-time log viewing
- Container resource monitoring

### 4. Alert Notifications
- DingTalk bot
- WeChat Work bot
- Email notifications
- Slack integration

### 5. Automatic Inspection
- Scheduled system inspection
- Automatic anomaly detection
- AI analysis reports
- Daily health reports

## 🚀 Deployment Options

### Docker Compose (Recommended)

```bash
# Quick start
./deploy.sh

# Or manual start
docker compose up -d --build
```

### Manual Deployment

```bash
# 1. Build frontend
cd frontend
npm install
npm run build

# 2. Build backend
cd ..
go build -o qwq ./cmd/qwq/main.go

# 3. Run
./qwq web
```

### Kubernetes

```bash
# Using Helm Chart
helm install qwq ./charts/qwq

# Or using kubectl
kubectl apply -f k8s/
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Service port | 8080 | No |
| `AI_PROVIDER` | AI provider (openai/ollama) | - | Yes |
| `OPENAI_API_KEY` | OpenAI API Key | - | Conditional |
| `OLLAMA_HOST` | Ollama service URL | http://localhost:11434 | Conditional |
| `DB_TYPE` | Database type | sqlite | No |
| `JWT_SECRET` | JWT secret key | - | Yes |

See [.env.example](.env.example) for complete configuration.

### Port Mapping

| Port | Service | Description |
|------|---------|-------------|
| 8081 | qwq Main | Frontend + API |
| 3308 | MySQL | Database |
| 6380 | Redis | Cache |
| 9091 | Prometheus | Monitoring |
| 3000 | Grafana | Visualization |

## 🔒 Security Recommendations

### Production Deployment

1. **Change Default Passwords**
   ```bash
   # Modify secrets in .env
   JWT_SECRET=$(openssl rand -base64 32)
   ENCRYPTION_KEY=$(openssl rand -base64 32)
   ```

2. **Enable HTTPS**
   ```bash
   # Use Nginx reverse proxy
   # Configure SSL certificates
   ```

3. **Configure Firewall**
   ```bash
   # Only open necessary ports
   ufw allow 80/tcp
   ufw allow 443/tcp
   ```

4. **Regular Backups**
   ```bash
   # Configure automatic backups
   BACKUP_ENABLED=true
   BACKUP_SCHEDULE="0 2 * * *"
   ```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 Changelog

### v1.0.0 (2024-12-08)

- ✨ Initial release
- 🤖 OpenAI and Ollama AI integration
- 📊 Complete monitoring and alerting system
- 🐳 Docker container management
- 📈 Prometheus + Grafana integration
- 🔐 JWT authentication and access control

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [OpenAI](https://openai.com/) - AI capabilities
- [Ollama](https://ollama.com/) - Local AI models
- [Vue.js](https://vuejs.org/) - Frontend framework
- [Go](https://go.dev/) - Backend language
- [Docker](https://www.docker.com/) - Containerization
- [Prometheus](https://prometheus.io/) - Monitoring system

## 📞 Contact

- Project Homepage: https://github.com/QwQBiG/qwq-aiops
- Issue Tracker: https://github.com/QwQBiG/qwq-aiops/issues
- Documentation: https://github.com/QwQBiG/qwq-aiops/wiki

---

<div align="center">

**If this project helps you, please give it a ⭐️ Star!**

Made with ❤️ by qwq Team

</div>
