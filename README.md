# 🤖 qwq - Enterprise AIOps Agent

> **你的私有化 AI 运维专家 | 交互式排查 · 全自动巡检 · 可视化监控 · 本地模型支持**

![Go Version](https://img.shields.io/badge/Go-1.23%2B-cyan.svg)
![Docker](https://img.shields.io/badge/Docker-Supported-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

**qwq** 是一个现代化的 AIOps 智能运维平台。它打破了传统脚本的限制，利用大语言模型（LLM）的推理能力，将运维工作转化为自然语言交互。支持连接云端 API（如 OpenAI/硅基流动）或 **本地私有模型（Ollama/DeepSeek）**，确保数据安全不出域。

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
*   **赛博朋克 UI**：内置暗黑风 Web 面板（端口 8899）。
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

  