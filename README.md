# 🤖 qwq - AIOps 智能运维 Agent

> **你的 24 小时 AI 运维专家，集交互排查、自动巡检、可视化监控于一体。**

**qwq** 是一个基于 Go 语言开发的轻量级 AIOps Agent。它利用大语言模型（LLM）的推理能力，结合 Linux/Kubernetes 原生命令，实现服务器的**自然语言交互排查**和**无人值守智能巡检**。

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.21%2B-cyan.svg)
![Docker](https://img.shields.io/badge/Docker-Supported-blue)

## ✨ 核心功能

*   **💬 交互式排查 (Chat Mode)**: 自然语言下达指令，支持多步推理（ReAct），自动执行只读命令，高危命令需人工确认。
*   **🚨 无人值守巡检 (Patrol Mode)**: 后台静默运行，每 5 分钟检测磁盘、负载、OOM 及僵尸进程。内置智能过滤，杜绝误报。
*   **📊 可视化仪表盘 (Web Dashboard)**: 端口 8899，提供暗黑风实时监控面板，支持实时日志查看和手动触发巡检。
*   **🧠 知识库增强 (RAG)**: 支持挂载私有知识库 (`docs.txt`)，让 AI 学会你的业务运维知识。
*   **📱 钉钉集成**: 异常实时告警，每 8 小时发送健康日报。

## 🚀 快速开始 (Docker 方式 - 推荐)

这是在任何新机器上部署的最快方法。

### 1. 准备配置文件
在服务器上创建一个目录（例如 `qwq-ops`），并新建 `config.json`：

```json
{
  "api_key": "sk-你的硅基流动Key",
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=你的Token",
  "web_user": "admin",
  "web_password": "secure_password",
  "knowledge_file": "/root/docs.txt",
  "debug": false
}
```

