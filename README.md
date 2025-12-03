# 🤖 qwq - 极简主义 AIOps 智能运维 Agent

**你的 24 小时 AI 运维专家**，集交互排查、自动巡检、可视化监控于一体。

qwq 是一个基于 Go 语言开发的轻量级 AIOps Agent。它利用大语言模型（LLM，如 Qwen2.5）的推理能力，结合 Linux/Kubernetes 原生命令，实现服务器的自然语言交互排查和无人值守智能巡检。

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.21%2B-cyan.svg)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-lightgrey.svg)

## ✨ 核心功能

### 💬 交互式排查 (Chat Mode)
- **自然语言指令**：直接说"帮我查一下 CPU 最高的进程"或"分析 K8s Pod 为什么起不来"。
- **智能执行**：自动识别只读命令（如 ls, top, kubectl get）并秒级执行；高危命令（如 rm, kill）需人工确认。
- **ReAct 推理**：支持多步推理，例如先查 PID，再根据 PID 查启动时间。

### 🚨 无人值守巡检 (Patrol Mode)
- **全天候守护**：后台静默运行，每 5 分钟自动检测磁盘、负载、OOM 日志及僵尸进程。
- **精准告警**：内置智能过滤逻辑，杜绝权限报错、空结果等误报。
- **AI 诊断**：告警信息包含 AI 生成的根因分析与修复命令（如自动识别僵尸进程需杀父进程）。

### 📊 可视化仪表盘 (Web Dashboard)
- **赛博朋克 UI**：内置 Web 服务器（端口 8899），提供暗黑风实时监控面板。
- **实时数据**：动态展示 CPU、内存、磁盘进度条及实时运行日志。
- **一键交互**：支持在网页端手动触发巡检。

### 📱 钉钉深度集成
- **健康日报**：每 8 小时自动发送服务器状态卡片（表格化排版）。
- **实时告警**：异常情况秒级推送。

## 🛠️ 安装指南

### 1. 环境要求
- Linux (推荐 Ubuntu) 或 macOS
- Go 1.21+ (用于编译)
- API Key: 硅基流动 (SiliconFlow) 或其他兼容 OpenAI 格式的 API Key。

### 2. 编译项目
```bash
# 1. 克隆项目
git clone https://github.com/your-username/qwq-aiops.git
cd qwq-aiops

# 2. 下载依赖
go mod tidy

# 3. 编译二进制文件
go build -o qwq main.go
```

## 3. 配置环境变量
为了安全起见，API Key 不硬编码在代码中，请设置环境变量：

```bash
# 临时生效
export OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx

# 永久生效 (推荐)
echo 'export OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxx' >> ~/.bashrc
source ~/.bashrc
```

## 📖 使用手册

### 1. 启动 Web 可视化面板 (推荐)
这是最强大的模式，同时启动 Web 服务器 + 后台巡检 + 健康日报。

```bash
# 使用 sudo -E 以便读取内核日志(dmesg)并保留环境变量
# 注意：URL 请使用双引号包裹
sudo -E nohup ./qwq web --webhook="https://oapi.dingtalk.com/robot/send?access_token=你的TOKEN" > web.log 2>&1 &
```

**访问地址**: http://服务器IP:8899  
**查看日志**: `tail -f web.log`

### 2. 交互式排查 (Chat)
像和真人专家聊天一样排查问题。

```bash
./qwq chat
```

**演示案例：**
```text
用户: "帮我看看 8080 端口被谁占用了？"
qwq: (自动执行 lsof -i :8080) "被 java 进程占用，PID 是 12345。"
用户: "杀掉它。"
qwq: "⚠️ 这是一个修改操作，确认执行 kill -9 12345 吗？(Y/n)"
```

### 3. 纯后台巡检 (Patrol)
如果你不需要 Web 面板，只想安静地监控。

```bash
sudo -E nohup ./qwq patrol --webhook="你的钉钉URL" > patrol.log 2>&1 &
```

### 4. 立即发送状态报告
手动触发一次钉钉健康日报推送。

```bash
./qwq status --webhook="你的钉钉URL"
```

## 🛡️ 安全机制

**Human-in-the-loop (人在回路):**
- 所有修改类命令（rm, kill, kubectl delete 等）必须经过用户 Y/n 确认。
- 查询类命令（ls, top, kubectl get）自动放行，提升效率。

**黑名单拦截:**
- 硬编码拦截 `rm -rf /`, `mkfs`, `dd if=/dev/zero` 等毁灭性命令。

**智能防幻觉:**
- 如果命令执行结果为空或报错，代码层直接拦截，防止 AI 编造虚假输出。

## ❓ 常见问题 (FAQ)

**Q: 启动时报错 API Error？**  
A: 请检查 `echo $OPENAI_API_KEY` 是否为空。如果使用 sudo 运行，请务必加上 `-E` 参数（`sudo -E ./qwq ...`）以传递环境变量。

**Q: 钉钉收不到消息？**  
A:  
1. 检查钉钉机器人安全设置，自定义关键词必须包含"告警"或"日报"。  
2. 启动命令中的 Webhook URL 不要包含反斜杠 `\` 转义符。

**Q: 僵尸进程检测不到？**  
A: 僵尸进程通常稍纵即逝。qwq 采用精准捕获逻辑，只有当 ps 状态栏明确显示 Z 时才会报警。如果父进程已退出，僵尸进程会被系统自动回收，此时显示"系统健康"是正常的。

**Q: 为什么需要 root 权限？**  
A: 读取内核环形缓冲区日志 (dmesg) 通常需要 root 权限。如果以普通用户运行，OOM（内存溢出）检测功能可能会失效（代码会自动忽略权限报错）。

## 🏗️ 项目结构

```text
.
├── main.go          # 核心源码 (单文件架构，易于维护)
├── go.mod           # 依赖管理
├── qwq              # 编译后的二进制程序
├── web.log          # Web 模式运行日志
└── README.md        # 项目文档
```