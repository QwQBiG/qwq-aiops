# 架构与目录职责

[English](architecture.en.md) · [返回 README](../README.md)

这个仓库以单个 Go 程序提供命令行和 Web 工作台。`web` 模式直接启动 `net/http` 服务；`gateway` 模式通过网关转发到同进程内、仅监听回环地址的 Web 服务。两种模式使用相同的处理函数和入口凭据。了解请求流程时，应从 `cmd/qwq/main.go` 和 `internal/server` 开始。

## 一次 Web 请求经过哪里

```text
浏览器中的 Vue 页面
  ├─ HTTP /api/... → 入口认证 → internal/server 的处理函数
  ├─ WebSocket /ws/chat → 入口认证与来源检查 → 对话与命令执行
  └─ 页面和静态资源 → Go 二进制嵌入的 internal/server/dist
```

入口程序加载配置、初始化日志与模型客户端，再启动巡检和 Web 服务。监控和容器操作会调用系统命令；通知使用配置中的渠道。`/health` 提供公开的最小存活响应，`/api/health` 返回需要认证的部署诊断信息。

`NewHandler` 创建独立路由，不注册进程级默认路由，也不启动采集任务。`Run` 管理监听、采集和退出；启动失败会返回调用者，取消会传播到系统命令、HTTP 检查和模型请求。未知的 `/api`、`/ws` 路径返回 `404`，不会被当作前端页面。

## 目录怎么分工

| 目录 | 职责 |
| --- | --- |
| `cmd/qwq` | 命令注册、初始化和巡检调度；保持启动相关逻辑集中 |
| `internal/server` | 独立路由、服务生命周期、静态资源、认证及业务 HTTP 适配 |
| `internal/config` | JSON 配置、环境变量覆盖，以及部署配置检查 |
| `internal/agent`、`executor`、`security` | 对话、命令执行和风险判断 |
| `internal/monitor`、`notify`、`logger`、`utils` | 系统检查、通知、日志和底层工具 |
| `internal/gateway` | 可选的控制台反向代理、认证和确定性路由 |
| `internal/deployment`、`platform` | 已生效配置、嵌入资源和运行环境的诊断 |
| `frontend/src/views` | 路由直接加载的页面 |
| `frontend/src/components` | 共享界面组件，例如功能状态提示 |
| `frontend/src/lib` | 错误信息、轮询、终端会话和安全内容渲染 |
| `frontend/src/i18n` | 中文和英文界面文案 |
| `scripts` | 跨平台构建入口 |
| `docs` | 当前中英文操作与架构指南 |
| `.github/workflows` | 测试、二进制构建和镜像构建 |

新增能力时，把服务实现与 HTTP 处理分开，再由 Web 处理函数做输入校验、认证和响应转换。涉及数据的功能应先明确存储和重启后的行为；菜单中的功能状态不能代替服务实现。

## 当前功能边界

| Web 入口 | 当前行为与依赖 |
| --- | --- |
| 仪表盘、日志、巡检 | 使用当前运行环境中的系统命令；完整流程依赖 Linux 工具 |
| 容器列表和操作 | 调用 Docker 命令，需要客户端、可达的守护进程及相应权限 |
| 文件浏览与编辑 | 存在 `/hostfs` 时以其为根，否则使用进程文件系统的 `/`；受路径和禁止目录检查限制。文本编辑限 UTF-8 普通文件、2 MiB，保存保留已有文件权限及 Unix UID/GID；未保存内容关闭或离开时提示 |
| 终端对话 | 调用兼容模型接口；自动执行只允许单条只读诊断命令，仍可能读取服务进程有权访问的信息 |
| 网站、用户、角色 | 维护内存配置记录，重启会丢失；不会发布反向代理或建立用户授权。证书签发、续期与用户权限读写返回 `501` |
| 应用商店、数据库页面 | 展示当前接入条件；管理写接口返回 `501`，不会声称部署或连接成功 |
| 运行时修复 | `/api/deployment/repair` 返回 `501`；配置变更和资源重建由操作者按部署流程执行 |

Web 入口使用共享的 HTTP Basic Auth。用户管理页的数据与入口凭据不是同一个认证系统。需要多用户、持久化权限或租户隔离时，应先完成认证和存储接入，再把工作台用于这些场景。

改变状态的操作使用写请求：`/api/container/action`、`/api/files/action` 和 `/api/trigger` 只接受 `POST`，使用 `GET` 会返回 `405`。浏览器写请求和 WebSocket 连接会检查 `Origin`：来源地址中的主机与端口必须和请求的 `Host` 一致，非法或外域来源返回 `403`。不带 `Origin` 的命令行客户端保持兼容，仍需通过入口认证。

反向代理终止 HTTPS 时，应保留浏览器发送的 `Host`；服务接受 `http` 和 `https` 来源，不依赖转发头恢复来源。代理重写 `Host` 会造成正常写请求被拒绝。

内置网关只把请求转发到 `127.0.0.1:WEB_UI_PORT`，保留原始 `Host` 和认证信息，支持普通 HTTP 与 WebSocket。它不注册预留微服务、不生成未实现的接口文档，也不另开注册管理端口。

自动命令策略拒绝管道、重定向、命令拼接和状态修改子命令；例如 `docker ps` 可自动诊断，`docker restart` 需要操作者走明确的容器操作。模型回复按经过清理的 Markdown 显示，文件内容以纯文本返回。这些约束不能代替入口认证或主机权限管理。

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

具体命令见[开发指南](development.md)，部署限制见[部署指南](deployment.md)。
