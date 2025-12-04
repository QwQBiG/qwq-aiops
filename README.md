# 🤖 qwq - Enterprise AIOps Agent

> **私有化 AI 运维大手子 | 交互式排查 · 全自动巡检 · 可视化监控 · 本地模型支持**

![Go Version](https://img.shields.io/badge/Go-1.23%2B-cyan.svg)
![Docker](https://img.shields.io/badge/Docker-Supported-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

**qwq** 是一个现代化的 AIOps 智能运维平台。它打破了传统脚本的限制，利用大语言模型（LLM）的推理能力，将运维工作转化为自然语言交互。支持连接**云端 API**（如 OpenAI/硅基流动）或 **本地私有模型（Ollama/DeepSeek）**，确保数据安全不出域。

---

## ✨ 核心功能 (Features)

### 🧠 1. 智能交互 (Chat Mode)
*   **自然语言运维**：直接对话 "帮我查一下 CPU 最高的进程" 或 "分析 K8s Pod 为什么 Crash"。
*   **ReAct 推理引擎**：AI 自动拆解任务（如：查 PID -> 查启动时间 -> 分析日志），支持多步执行。
*   **Web/CLI 双端**：支持终端命令行交互，也支持 Web 网页端实时对话。

### 🚨 2. 全自动巡检 (Patrol Mode)
*   **深度健康检查**：后台静默运行，每 5 分钟检测磁盘、负载、OOM 及僵尸进程。
*   **智能根因分析**：发现异常后，AI 自动分析原因并给出修复建议（如自动识别僵尸进程需杀父进程）。
*   **自定义规则**：支持在配置文件中添加 Shell 脚本规则（如检查 Nginx 进程、Docker 容器状态）。

### 📊 3. 可视化控制台 (Web Dashboard)
*   **不丑的 UI**：内置 dark 风 Web 面板（端口 8899）。
*   **实时监控**：基于 ECharts 的 CPU、内存、磁盘实时趋势图。
*   **应用拨测**：内置 HTTP 监控，实时检测业务网站/API 连通性。
*   **实时日志**：通过 WebSocket 实时推送后台运行日志。

### 🔒 4. 企业级安全
*   **Web 鉴权**：支持 HTTP Basic Auth，防止面板未授权访问。
*   **命令风控**：内置黑名单（拦截 `rm -rf`），高危命令需人工确认（Human-in-the-loop）。
*   **数据脱敏**：自动隐藏日志中的 IP、密钥等敏感信息后再发送给 AI。

### 🏠 5. 本地模型与知识库 (RAG)
*   **Ollama 支持**：完美适配 DeepSeek、Qwen 等本地模型，零成本、零泄露。
*   **私有知识库**：挂载 `docs.txt`，让 AI 学会你们公司的特定运维知识（如服务重启步骤）。

---

## 🚀 快速开始 (Docker 方式)

无需安装 Go 环境，直接使用 Docker 一键启动。

### 1. 准备配置文件
在服务器创建目录 `qwq-ops`，新建 `config.json`：

```json
{
  "api_key": "ollama", 
  "base_url": "http://127.0.0.1:11434/v1",
  "model": "deepseek-r1:7b",
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN",
  "web_user": "admin",
  "web_password": "password123",
  "knowledge_file": "/root/docs.txt",
  "debug": false,
  "patrol_rules": [
    { "name": "Nginx检查", "command": "pgrep nginx || echo 'Nginx Down'" }
  ],
  "http_rules": [
    { "name": "百度连通性", "url": "https://www.baidu.com", "code": 200 }
  ]
}
```

### 2. 启动容器
```bash
docker run -d \
  --name qwq \
  --restart unless-stopped \
  --network host \
  -v $(pwd)/config.json:/root/config.json \
  -v $(pwd)/qwq.log:/root/qwq.log \
  ghcr.io/qwqbig/qwq-aiops:main \
  web -c /root/config.json
  ```

  **访问面板**: http://服务器IP:8899  
**查看日志**: `tail -f qwq.log`

## 🛠️ 开发者指南

### 目录结构

```text
qwq-aiops/
├── cmd/qwq/main.go         # 程序入口
├── internal/               # 核心模块
│   ├── agent/              # AI 智能体 (OpenAI/Ollama)
│   ├── config/             # 配置管理
│   ├── executor/           # 智能执行器
│   ├── logger/             # 日志系统 (Lumberjack)
│   ├── monitor/            # HTTP 应用监控
│   ├── notify/             # 告警中心 (DingTalk/Telegram)
│   ├── security/           # 安全风控与脱敏
│   ├── server/             # Web 服务器 & 前端
│   └── utils/              # 底层工具
├── Dockerfile              # 构建文件
└── go.mod                  # 依赖定义
```

### 本地编译
```bash
git clone https://github.com/qwqbig/qwq-aiops.git
go mod tidy
go build -o qwq cmd/qwq/main.go
```

## 📄 License
MIT License. Copyright (c) 2025 qwqBig.

---

## 🎬 第二部分：全 Docker "try a try" (示范)

假设你现在拿到了一台**全新的 Ubuntu 服务器**，里面什么都没有（只有 Docker）。
我们要实现：**本地跑 DeepSeek-R1 大模型 + qwq 智能运维平台**。

请按以下步骤复制粘贴：

### 1. 启动大脑 (Ollama + DeepSeek)

```bash
# 1.1 启动 Ollama 服务
sudo docker run -d \
  --name ollama \
  --restart always \
  --network host \
  -v ollama:/root/.ollama \
  ollama/ollama

# 1.2 下载 DeepSeek-R1 模型 (7B版本)
# 注意：这一步取决于网速，可能需要几分钟
sudo docker exec -it ollama ollama run deepseek-r1:7b
# (下载完成后，出现 >>> 提示符时，按 Ctrl+D 退出)
```

### 2. 准备 qwq 配置
```bash
# 2.1 创建工作目录
mkdir -p ~/qwq-ops && cd ~/qwq-ops

# 2.2 创建知识库 (可选)
echo "如果遇到磁盘报警，请优先清理 /var/log/journal 目录。" > docs.txt

# 2.3 创建配置文件
cat <<EOF > config.json
{
  "api_key": "ollama",
  "base_url": "http://127.0.0.1:11434/v1",
  "model": "deepseek-r1:7b",
  "webhook": "", 
  "web_user": "admin",
  "web_password": "123",
  "knowledge_file": "/root/docs.txt",
  "debug": true,
  "patrol_rules": [
    { "name": "Docker守护进程", "command": "systemctl is-active docker || echo 'Docker is dead'" }
  ],
  "http_rules": [
    { "name": "本地Ollama", "url": "http://127.0.0.1:11434", "code": 200 }
  ]
}
EOF
# (注意：如果你有钉钉 Webhook，请把上面的 webhook 字段填上)
```

### 3. 启动 qwq 智能体
```bash
# 3.1 拉取并启动 (使用 host 网络模式以便连接 Ollama)
sudo docker run -d \
  --name qwq \
  --restart unless-stopped \
  --network host \
  -v $(pwd)/config.json:/root/config.json \
  -v $(pwd)/docs.txt:/root/docs.txt \
  -v $(pwd)/qwq.log:/root/qwq.log \
  ghcr.io/qwqbig/qwq-aiops:main \
  web -c /root/config.json
  ```

  ### 4. 见证时刻
1. **打开浏览器**：访问 http://你的服务器IP:8899。
2. **登录**：输入账号 `admin`，密码 `123`。 （输入你自己的哈）
3. **看面板**：你会看到 CPU、内存曲线开始跳动，左下角显示 "本地Ollama UP"。
4. **调戏 AI**：
   - 在右侧聊天框输入：**磁盘满了怎么办？**
   - **预期回答**：它会根据 docs.txt 回答你：“根据内部知识库，请优先清理 /var/log/journal 目录。”
   - 输入：**帮我看看当前系统负载。**
   - **预期回答**：它会自动执行 `uptime` 并告诉你结果。
   
---

**THANKS**