# 架构与目录职责

[English](architecture.en.md) · [返回 README](../README.md)

这个仓库以单个 Go 程序提供命令行和 Web 工作台。`web` 模式直接启动 `net/http` 服务；`gateway` 模式另外启动基于 `net/http` 的网关，并把 Web 服务作为其中一个入口。了解 `web` 模式时，应从 `cmd/qwq/main.go` 和 `internal/server` 开始。

## 一次 Web 请求经过哪里

```text
浏览器中的 Vue 页面
  ├─ HTTP /api/... → 入口认证 → internal/server 的处理函数
  ├─ WebSocket /ws/chat → 入口认证与来源检查 → 对话与命令执行
  └─ 页面和静态资源 → Go 二进制嵌入的 internal/server/dist
```

入口程序加载配置、初始化日志与模型客户端，再启动巡检和 Web 服务。监控和容器操作会调用系统命令；通知使用配置中的渠道。`/health` 提供公开的最小存活响应，`/api/health` 返回需要认证的部署诊断信息。

## 目录怎么分工

| 目录 | 职责 |
| --- | --- |
| `cmd/qwq` | 命令注册、初始化和巡检调度；保持启动相关逻辑集中 |
| `internal/server` | 当前 Web 路由、静态资源、认证、健康检查和接口适配 |
| `internal/config` | JSON 配置、环境变量覆盖，以及部署配置检查 |
| `internal/agent`、`executor`、`security` | 对话、命令执行和风险判断 |
| `internal/monitor`、`notify`、`logger`、`utils` | 系统检查、通知、日志和底层工具 |
| `internal/gateway`、`registry` | 可选网关、路由和服务发现 |
| 其余 `internal` 业务包 | 容器部署、数据库、网站、备份、集群等独立服务实现 |
| `frontend/src/views` | 路由直接加载的页面 |
| `frontend/src/components` | 组件与早期页面实现；引用关系以当前路由为准 |
| `frontend/src/i18n` | 中文和英文界面文案 |
| `scripts` | 跨平台构建入口；`legacy` 保存依赖旧布局的历史脚本 |
| `config` | Prometheus、MySQL 等可选基础服务的配置 |
| `docs` | 当前操作指南；`archive` 保存旧版本资料 |
| `.github/workflows` | 测试、二进制构建和镜像构建 |

新增能力时，优先在已有业务包中实现，再由 Web 处理函数做输入校验、认证和响应转换。业务包存在不代表 `web` 模式已经使用它，也不应把页面上出现的菜单当成服务已接入的依据。

## 当前功能边界

| Web 入口 | 当前行为与依赖 |
| --- | --- |
| 仪表盘、日志、巡检 | 使用当前运行环境中的系统命令；完整流程依赖 Linux 工具 |
| 容器列表和操作 | 调用 Docker 命令，需要客户端、可达的守护进程及相应权限 |
| 文件浏览与编辑 | 存在 `/hostfs` 时以其为根，否则使用进程文件系统的 `/`；受路径检查和禁止目录限制 |
| 终端对话 | 调用兼容接口，可能执行系统命令；只应开放给可信操作者 |
| 网站、用户、角色 | 部分接口使用进程内存，重启会丢失；没有完整的租户与权限隔离 |
| 应用商店、数据库页面 | 尚未接通对应业务包；列表为空，管理写操作返回 `501`，不会声称部署或连接成功 |
| 备份、集群、扩展监控等 | 仓库中有独立包，使用前需核对各自调用入口与测试 |

Web 入口使用共享的 HTTP Basic Auth。用户管理页的数据与入口凭据不是同一个认证系统。需要多用户、持久化权限或租户隔离时，应先完成认证和存储接入，再把工作台用于这些场景。

改变状态的操作使用写请求：`/api/container/action`、`/api/files/action` 和 `/api/trigger` 只接受 `POST`，使用 `GET` 会返回 `405`。浏览器写请求和 WebSocket 连接会检查 `Origin`：来源地址中的主机与端口必须和请求的 `Host` 一致，非法或外域来源返回 `403`。不带 `Origin` 的命令行客户端保持兼容，仍需通过入口认证。

反向代理终止 HTTPS 时，应保留浏览器发送的 `Host`；服务接受 `http` 和 `https` 来源，不依赖转发头恢复来源。代理重写 `Host` 会造成正常写请求被拒绝。

## 前端资源如何进入二进制

```text
frontend/package-lock.json
  → npm ci → Vite build → frontend/dist
  → 同步到 internal/server/dist → go build → bin/qwq
```

`internal/server/dist` 是生成目录，仓库中的占位文件只保证包可以编译。它们不提供完整的工作台。`node scripts/build.mjs` 按上述顺序执行，`--frontend-only` 只准备嵌入资源。修改前端后必须重新编译 Go，已编译的程序不会读取新生成的页面。

构建产物放在 `frontend/dist`、`internal/server/dist` 和 `bin`，不应手工维护带哈希的资源文件。依赖锁文件和源码属于版本控制内容；运行时的凭据、数据和日志应留在仓库之外。

## 如何判断改动已经接通

先构建完整程序，再从浏览器或 HTTP 客户端走一次实际路径：请求是否通过认证、处理函数调用了哪个服务、服务是否真的改变了目标状态、错误是否如实返回。单个业务包的测试通过，只能说明该包覆盖到的行为；它不能证明菜单、Web 接口、外部依赖和持久化已经连通。

具体命令见[开发指南](development.md)，部署限制见[部署指南](deployment.md)。历史实现报告保留在[归档](archive/README.md)，不作为当前功能完成的凭据。
