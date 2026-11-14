# qwq Ops

简体中文 | [English](README_EN.md)

qwq Ops 是一个 Go + Vue 3 运维工作台，提供系统状态查看、Docker 容器操作、文件管理、巡检通知和模型对话。它适合在受控环境中运行和继续开发；终端与容器操作会直接影响运行服务的机器。

## 先了解当前范围

Web 服务使用 Go 标准库 `net/http`，前端由 Vite 构建后嵌入 Go 二进制。常用入口包括仪表盘、容器、文件、日志和终端。

仓库也包含数据库、应用商店、网站、备份、集群等业务包。它们的实现程度和 Web 接入程度不同：网站、用户和角色等部分接口使用进程内存保存数据，重启后不会保留，用户与角色页面也不构成完整的权限隔离。请先阅读[架构与功能边界](docs/architecture.md)，再决定接入哪些模块。

## 从源码运行

需要 Go 1.23 或更高版本、Node.js 22 和 npm。系统巡检依赖 Linux 命令；Windows 可以构建和开发前端，运行完整巡检时请使用 Linux 环境。

```bash
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops
node scripts/build.mjs
```

构建脚本安装锁文件指定的前端依赖、生成并同步静态资源，最后输出 `bin/qwq`；Windows 输出 `bin/qwq.exe`。先构建前端再编译 Go，才能把当前页面嵌入程序。

在同一个终端中设置服务配置，然后启动。以下是 Bash 示例；[开发指南](docs/development.md)中有 PowerShell 命令。

```bash
export OPENAI_API_KEY='your-api-key'
export OPENAI_BASE_URL='https://api.openai.com/v1'
export OPENAI_MODEL='your-model-name'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
./bin/qwq web
```

`OPENAI_BASE_URL` 应指向服务商的兼容接口，模型名称必须与该服务商一致。二进制不会自动读取 `.env`；直接运行时需要导出环境变量，或通过 `--config` 指定 JSON 配置。使用本地模型和 Docker Compose 的配置见[部署指南](docs/deployment.md)。

打开 <http://localhost:8080>，浏览器会请求上面设置的 HTTP Basic Auth 用户名和密码。服务默认监听 `:8080`；跨网络访问时请使用 HTTPS 和受控的网络入口。

```bash
curl --fail http://localhost:8080/health
curl --fail --user operator http://localhost:8080/api/health
```

`/health` 返回最小 JSON 存活状态，不要求登录；它不检查数据库、Docker 或外部模型是否可用。`/api/health` 是需要认证的部署诊断接口。

## 使用前配置认证

`WEB_USER` 和 `WEB_PASSWORD` 必须一起设置，只设置一个会启动失败。两者都为空时，为兼容现有本地用法，服务仍允许匿名访问。请在启动服务前设置凭据，尤其不要把匿名服务暴露到公网。

文件管理受挂载目录和禁止目录约束，但整个工作台仍具有运维权限。Docker socket、宿主机目录和终端权限只应授予可信的操作者。页面中的用户列表不能替代入口认证。

## 项目目录

```text
cmd/qwq/          命令行入口和启动编排
internal/         Web 接入、配置、监控与业务包
frontend/         Vue 页面、路由、翻译与依赖锁文件
scripts/          统一构建入口及历史脚本
config/           可选基础服务配置
docs/             当前指南；archive/ 保存历史记录
.github/workflows/  自动化检查与构建
```

目录职责、请求流程和静态资源的生成方式见[架构说明](docs/architecture.md)。

## 文档与贡献

| 内容 | 中文 | English |
| --- | --- | --- |
| 部署、配置与排障 | [部署指南](docs/deployment.md) | [Deployment](docs/deployment.en.md) |
| 本地开发与验证 | [开发指南](docs/development.md) | [Development](docs/development.en.md) |
| 目录职责与功能边界 | [架构说明](docs/architecture.md) | [Architecture](docs/architecture.en.md) |
| 提交问题与修改 | [贡献指南](CONTRIBUTING.md) | [Contributing](CONTRIBUTING_EN.md) |

旧的部署说明、实现报告和测试记录保存在[文档归档](docs/archive/README.md)。归档中的完成状态和验证结论只适用于当时的代码，请以当前源码和实际运行结果为准。

遇到问题请提交 [Issue](https://github.com/QwQBiG/qwq-aiops/issues)，附上复现步骤、运行环境和脱敏日志。
