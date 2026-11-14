# 本地开发与验证

[English](development.en.md) · [返回 README](../README.md)

使用 Go 1.23 或更高版本、Node.js 22 和 npm。完整巡检依赖 Bash 与 Linux 系统工具；Windows 可以检查构建、HTTP 接口和前端，但不能据此认定 Linux 巡检已经验证。

## 构建完整程序

在仓库根目录运行：

```bash
node scripts/build.mjs
```

该入口在 Windows、Linux 和 macOS 上使用同一套顺序：`npm ci`、前端构建、同步嵌入资源、`go build`。输出为 `bin/qwq`，Windows 为 `bin/qwq.exe`。脚本从自身位置确定仓库目录，不要求调用者手动复制前端文件。

需要只准备前端资源时使用：

```bash
node scripts/build.mjs --frontend-only
```

仓库保留的资源占位文件允许 Go 包编译，但不能代替完整前端。直接执行 `go build` 前，应先运行上述资源构建。修改 `frontend/src` 后也需要重新编译 Go，才能更新二进制内的页面。

## 启动后端

以下 PowerShell 示例在当前终端设置配置。请换成实际服务商支持的地址和模型：

```powershell
$env:OPENAI_API_KEY = 'your-api-key'
$env:OPENAI_BASE_URL = 'https://api.openai.com/v1'
$env:OPENAI_MODEL = 'your-model-name'
$env:WEB_USER = 'operator'
$env:WEB_PASSWORD = 'replace-with-a-long-unique-password'
go run ./cmd/qwq web
```

Bash 使用 `export NAME='value'`；完整示例见[部署指南](deployment.md)。`.env` 只供 Compose 插值使用，直接运行程序时不会加载。后端默认端口为 `8080`。

## 启动前端开发服务器

保持后端运行，另开终端执行：

```bash
cd frontend
npm ci
npm run dev
```

打开 <http://localhost:5173>。Vite 把 `/api` 和 `/ws` 转发到 `localhost:8080`，界面请求使用后端的入口凭据。开发代理与生产二进制嵌入页面是两条路径；修改代理配置时应分别验证。

## 按改动验证

```bash
go test ./internal/config ./internal/server
go test ./...
```

第一条覆盖配置和 Web 包，适合定位相关改动；第二条检查整个 Go 模块。测试失败时应保留包名、命令和原始错误，区分编译错误、断言失败与本机缺少外部依赖。

前端变更至少执行 `npm run build`；改变嵌入或打包流程时，执行完整的 `node scripts/build.mjs`。配置与路由变更还要启动实际程序，确认正常路径和拒绝路径：

```bash
curl --fail http://localhost:8080/health
curl --include http://localhost:8080/api/health
curl --fail --user operator http://localhost:8080/api/health
```

Windows PowerShell 请使用 `curl.exe`，避免旧版本中的命令别名。启用认证后，第二条请求应返回 `401`；第三条会提示输入密码。打开页面并确认至少一个静态资源请求成功，随后查看 `qwq.log`。存活检查不调用 Docker 或模型，相关功能需要另走一次实际操作。

Docker 改动可执行：

```bash
docker compose config --quiet
docker compose build qwq
docker compose up -d qwq
docker compose logs --tail 100 qwq
```

这些命令需要先完成[部署配置](deployment.md)。本机没有 Docker 时，应明确记录尚未验证镜像运行，并检查对应的远程构建结果；不能以配置可解析替代镜像运行验证。

## 持续集成检查什么

`.github/workflows/build.yml` 在 Ubuntu 上使用 Go 1.23 和 Node.js 22，先构建真实前端资源，再执行整个模块的竞争检查与覆盖率测试。后续二进制构建复用这份前端资源，交叉编译时关闭 CGO。镜像流程位于 `docker-publish.yml`，负责 Linux 多架构构建；PR 构建不推送镜像。

这些是工作流配置，不代表任意提交都已经通过。提交 PR 后，查看本次提交对应的检查结果；失败时根据原始日志定位，不通过排除失败包来宣布整个模块已通过。

早期部署和诊断脚本保存在 [`scripts/legacy`](../scripts/legacy/README.md)，它们依赖旧目录和配置，仅用于查阅。当前构建使用 `scripts/build.mjs`，部署使用仓库根目录的 Compose 配置。

## 文件与提交约定

文本使用 UTF-8。`.gitattributes` 统一源码、Markdown 和 Shell 文件为 LF，Windows `.bat` 文件为 CRLF。构建输出、依赖目录和运行数据不进入提交。

新增行为放进负责该领域的包，测试紧邻包。修改公开操作步骤或配置时，同步检查中文与英文；两种语言应描述同一个行为和限制。提交范围与验证结果的写法见[贡献指南](../CONTRIBUTING.md)。
