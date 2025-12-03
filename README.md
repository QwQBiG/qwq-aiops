# 🤖 qwq - AIOps 智能运维 Agent

**你的 24 小时 AI 运维专家 | 交互式排查 · 全自动巡检 · 可视化监控**

![Go Version](https://img.shields.io/badge/Go-1.23%2B-cyan.svg)
![Docker](https://img.shields.io/badge/Docker-Supported-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen)

qwq 是一个基于 Go 语言开发的现代化 AIOps 平台。它利用大语言模型（LLM，如 Qwen2.5）的推理能力，结合 Linux/Kubernetes 原生命令，将传统的运维工作转化为自然语言交互，并提供全天候的智能守护。

## ✨ 核心功能 (Features)

### 🧠 1. 智能交互 (Chat Mode)
- **自然语言运维**：直接对话 "帮我查一下 CPU 最高的进程" 或 "分析 K8s Pod 为什么 Crash"。
- **ReAct 推理引擎**：AI 会自动拆解任务（如：查 PID -> 查启动时间 -> 分析日志），支持多步执行。
- **Web/CLI 双端支持**：既可以在终端使用，也可以在 Web 页面像用 ChatGPT 一样对话。

### 🚨 2. 全自动巡检 (Patrol Mode)
- **深度健康检查**：每 5 分钟自动检测磁盘、负载、OOM 日志及僵尸进程。
- **智能根因分析**：发现异常后，AI 自动分析原因并给出修复建议（例如：自动识别僵尸进程并建议杀掉父进程）。
- **自定义规则**：支持在配置文件中添加 Shell 脚本规则（如检查 Nginx 进程、Docker 容器状态）。

### 📊 3. 可视化控制台 (Web Dashboard)
- **赛博朋克 UI**：内置暗黑风 Web 面板（端口 8899），基于 ECharts 实现。
- **实时监控**：动态展示 CPU、内存、磁盘趋势图。
- **应用层监控**：内置 HTTP 拨测功能，实时监控网站/API 连通性。
- **实时日志**：通过 WebSocket 实时推送后台运行日志。

### 📢 4. 多渠道告警 (Notification)
- **钉钉 (DingTalk)**：支持 Markdown 卡片告警。
- **Telegram**：支持 Bot 消息推送。
- **健康日报**：每 8 小时自动发送服务器状态汇总报表。

### 📚 5. 知识库增强 (RAG)
- **私有知识注入**：挂载 `docs.txt`，让 AI 学会你们公司的特定运维知识（如服务重启步骤、错误码含义）。

## 🚀 快速开始 (Docker 部署 - 推荐)
这是最快、最稳定的部署方式。

### 1. 准备配置文件
在服务器上创建目录 `qwq-ops`，并新建 `config.json`：
```json
{
  "api_key": "sk-你的硅基流动Key",
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=你的Token",
  "telegram_token": "",
  "telegram_chat_id": "",
  "web_user": "admin",
  "web_password": "secure_password",
  "knowledge_file": "/root/docs.txt",
  "debug": false,
  "patrol_rules": [
    {
      "name": "Nginx 存活检查",
      "command": "if ! pgrep nginx > /dev/null; then echo 'Nginx process not found'; fi"
    }
  ],
  "http_rules": [
    {
      "name": "官网主页",
      "url": "https://www.baidu.com",
      "code": 200
    }
  ]
}
```

### 2. 启动容器
```bash
# 确保你已经构建或拉取了镜像
# 如果是本地构建：docker build -t qwq-aiops .

docker run -d \
  --name qwq \
  --restart unless-stopped \
  --network host \
  -v $(pwd)/config.json:/root/config.json \
  -v $(pwd)/qwq.log:/root/qwq.log \
  qwq-aiops \
  web -c /root/config.json
```

**访问面板**: http://服务器IP:8899 (账号密码见配置文件)  
**查看日志**: `tail -f qwq.log`

## 🛠️ 从源码编译 (开发模式)
如果你想二次开发或在本地运行：

### 1. 环境要求
- Go 1.23+
- Linux / macOS (Windows 仅限编译，无法运行 Shell 命令)

### 2. 编译
```bash
git clone https://github.com/your-username/qwq-aiops.git
cd qwq-aiops

# 下载依赖
go mod tidy

# 编译二进制文件
go build -o qwq main.go
```

### 3. 运行模式

**Web 模式 (推荐)**：启动 Web 服务器 + 后台巡检。
```bash
sudo -E ./qwq web -c config.json
```

**交互模式**：纯命令行对话。
```bash
./qwq chat -c config.json
```

**巡检模式**：仅后台运行，无 Web 界面。
```bash
sudo -E ./qwq patrol -c config.json
```

## 📂 项目结构 (Modular Architecture)
项目采用标准的 Go 模块化结构设计，易于维护和扩展。

```text
qwq-aiops/
├── main.go                 # 程序主入口
├── go.mod                  # 依赖定义
├── Dockerfile              # Docker 构建文件
├── config.json             # 配置文件示例
├── internal/               # 核心代码库
│   ├── agent/              # AI 智能体 (OpenAI 交互、ReAct 逻辑)
│   ├── config/             # 配置加载与解析
│   ├── logger/             # 日志系统 (Lumberjack 轮转、多端输出)
│   ├── monitor/            # 应用层监控 (HTTP 拨测)
│   ├── notify/             # 消息通知中心 (DingTalk, Telegram)
│   ├── server/             # Web 服务器 (HTTP/WebSocket)
│   │   └── static/         # 前端资源 (HTML/CSS/JS)
│   └── utils/              # 底层工具 (Shell 执行、安全检查)
└── README.md               # 项目文档
```


## 🛡️ 安全机制
- **Web 鉴权**: 强制支持 HTTP Basic Auth，防止面板未授权访问。
- **命令拦截**: 内置黑名单，拦截 `rm -rf /`, `mkfs` 等高危命令。
- **Human-in-the-loop**: 修改类命令（Write Ops）必须经过人工确认，只读命令（Read Ops）自动执行。
- **防幻觉设计**: 代码层拦截空结果和错误码，防止 AI 编造虚假输出。

## ❓ 常见问题 (FAQ)

**Q: 为什么 Web 面板里内存显示和 `free -h` 不一样？**  
A: Web 面板使用 `free -m` 计算精确百分比 `(used/total)*100`，比 `free -h` 的概览更适合图表展示。

**Q: 僵尸进程检测不到？**  
A: 僵尸进程通常稍纵即逝。qwq 采用精准捕获逻辑，只有当 ps 状态栏明确显示 Z 时才会报警。如果父进程已退出，僵尸进程会被系统自动回收，此时显示"系统健康"是正常的。

**Q: 如何添加新的告警渠道？**  
A: 修改 `internal/notify/notify.go`，在 Send 函数中增加新的发送逻辑即可。

## 📄 License
MIT License. Copyright (c) 2025 qwqBiG.