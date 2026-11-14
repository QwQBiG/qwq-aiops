# 部署与配置

[English](deployment.en.md) · [返回 README](../README.md)

先在可信的测试环境中运行，再根据实际需要授予主机目录、Docker 和终端权限。工作台中的操作使用服务进程的权限；HTTP Basic Auth 凭据相当于工作台入口的管理凭据。

## 直接运行二进制

先按[开发指南](development.md)构建。Linux 中从仓库根目录执行：

```bash
export OPENAI_API_KEY='your-api-key'
export OPENAI_BASE_URL='https://api.openai.com/v1'
export OPENAI_MODEL='your-model-name'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
export PORT='8080'
./bin/qwq web
```

二进制不会读取 `.env`，这些变量必须存在于启动进程的环境中。服务监听 `:PORT`，默认端口为 `8080`。跨网络访问时，请在前面配置 HTTPS，并限制网络访问范围。

反向代理需要保留浏览器的 `Host`，以便写请求和 WebSocket 通过来源检查。外域或非法 `Origin` 会返回 `403`；服务不使用 `X-Forwarded-Host` 等转发头代替原始 `Host`。

也可以用 `--config /path/to/config.json` 加载 JSON。配置字段以 `internal/config/config.go` 中的 `Config` 为准，例如 `api_key`、`base_url`、`model`、`web_user`、`web_password`、`patrol_rules` 和 `http_rules`。环境变量会覆盖对应字段；请勿将真实凭据提交到仓库。

## 运行可选网关

在同一终端配置好上述凭据和模型地址后，可以改用：

```bash
export PORT='8080'
export WEB_UI_PORT='8899'
./bin/qwq gateway
```

访问入口仍是 <http://localhost:8080>。网关使用入口凭据认证，并把 HTTP、WebSocket 转发到内部的 `127.0.0.1:8899`，保留浏览器的 `Host`。`WEB_UI_PORT` 应选未占用的端口，并与 `PORT` 不同。任何监听器启动失败都会结束本次启动；停止进程时，两端服务一起退出。该模式只代理当前控制台，不会启动其他业务包对应的微服务。

## 连接本地模型

运行在宿主机上的服务可以使用以下 Bash 配置：

```bash
export AI_PROVIDER='ollama'
export OLLAMA_HOST='http://localhost:11434'
export OLLAMA_MODEL='your-installed-model'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
./bin/qwq web
```

该模式通过本地服务的 `/v1` 兼容接口发送请求，`OLLAMA_HOST` 应填写服务根地址。模型需要预先安装；配置成功并不表示模型已经下载或接口已经可达。

容器中的 `localhost` 指向容器本身。Docker Desktop 通常可以通过 `host.docker.internal` 访问宿主机；Linux 上需要根据部署网络配置宿主机地址或将模型服务接入同一网络。不要直接照抄其他机器的网桥地址。

## 使用 Docker Compose

需要可运行 Linux 容器的 Docker 和 Compose V2。在仓库根目录复制模板，然后编辑 `.env`，明确配置服务地址、模型和入口凭据：

```bash
cp .env.example .env
```

```dotenv
AI_PROVIDER=openai
OPENAI_API_KEY=your-api-key
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=your-model-name
WEB_USER=operator
WEB_PASSWORD=replace-with-a-long-unique-password
```

PowerShell 复制命令为 `Copy-Item -LiteralPath '.env.example' -Destination '.env'`。Compose 从 `.env` 读取插值值，再把 `docker-compose.yml` 中列出的变量传入容器；这与二进制自动加载 `.env` 是两回事。

```bash
docker compose config --quiet
docker compose up -d --build qwq
docker compose ps
docker compose logs --tail 100 qwq
curl --fail http://localhost:8081/health
```

访问 <http://localhost:8081>。默认只启动工作台，服务端口绑定宿主机的 `127.0.0.1`。MySQL、Redis 和监控服务分别放在 `database`、`cache`、`monitoring` profile 中；例如 `docker compose --profile monitoring up -d` 会额外启动 Prometheus 和 Grafana。开放辅助服务的端口前，请先配置各自的访问控制。启动数据库服务不会自动把内存中的网站、用户和角色数据改成持久化数据。

启用 `database` 前设置 `MYSQL_ROOT_PASSWORD` 和 `MYSQL_PASSWORD`，启用 `monitoring` 前设置 `GRAFANA_ADMIN_PASSWORD`。当前 Prometheus 配置只抓取自身指标；启动这些容器不会自动接入工作台指标或生成 Grafana 面板。

镜像以非 root 用户运行。宿主机挂载的 `data`、`logs`、`backups` 目录需要允许该用户写入；Docker 操作还要求容器用户能访问守护进程。请按实际用户和组配置权限，不要为了消除报错开放整个宿主机目录。将 Docker socket 挂载为 `:ro` 也不能限制 Docker API 的修改操作。

## 确认服务实际可用

```bash
curl --fail http://localhost:8081/health
curl --fail --user operator http://localhost:8081/api/health
```

第一个请求应返回 JSON 存活状态；第二个请求会提示输入密码。未提供凭据访问受保护接口应返回 `401`。存活检查不验证 Docker、模型、数据库或业务数据，请另走一次你将使用的实际功能，并检查服务日志。

部署诊断读取本进程已生效的配置和嵌入资源；通知未配置表示该可选功能关闭。模型能否回答、Docker 能否管理容器，仍需各自的实际请求确认。证书签发、用户权限管理和运行时自动修复尚未接入，对应操作返回 `501`；请按部署流程修改配置、重新构建和重启。

Compose 要求 `WEB_USER`、`WEB_PASSWORD` 均为非空。直接运行二进制时，两者必须成对设置：只配置其中一个会拒绝启动，两者都为空则保留匿名访问。改动 Compose 的环境变量后，应重新创建容器使配置生效，例如 `docker compose up -d qwq`；单纯 `restart` 不会刷新容器环境。

## 常见问题

| 现象 | 先检查什么 |
| --- | --- |
| 启动提示缺少密钥 | 确认变量在当前进程环境中；直接运行不会加载 `.env` |
| 配置了密码却无法登录 | 确认用户名和密码一起传入，重新创建容器，并检查浏览器缓存的凭据 |
| 页面不存在或只有占位资源 | 运行完整构建，再重新编译 Go；仅执行 `npm run build` 不会更新旧二进制 |
| Docker 操作失败 | 检查 Docker 客户端、守护进程地址和容器用户的 socket 权限 |
| 模型请求失败 | 检查兼容接口根地址、模型名称、密钥和容器到服务端的网络 |
| 写请求返回 `403` | 检查入口凭据与 `Origin`，并确认代理保留原始 `Host` |
| API 返回 `404` 或 `501` | `404` 表示没有该接口；`501` 表示对应管理能力尚未接入，查看[功能边界](architecture.md) |
| 文件保存失败或格式异常 | 编辑限 2 MiB 的 UTF-8 普通文本；检查进程权限，JSON 和 HTML 均按原始文本编辑 |
| 数据在重启后消失 | 查看[功能边界](architecture.md)，确认该接口是否仍使用进程内存 |

源码方式运行时查看工作目录中的 `qwq.log`；容器方式先查看 `docker compose logs`。排障时保留原始数据和日志，确认原因后再处理，不需要先清空卷或全局构建缓存。
