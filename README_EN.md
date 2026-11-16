# qwq Ops

[简体中文](README.md) | English

qwq Ops is a Go and Vue 3 operations workbench for system status, Docker containers, files, scheduled checks, notifications, and model conversations. Automatic terminal execution is limited to single read-only diagnostic commands. File and container actions modify the service's environment, so restrict access to trusted operators.

## What is connected today

The Web service uses Go's standard `net/http` package. Vite builds the frontend, which is then embedded in the Go binary. Dashboard, containers, files, logs, and terminal are the main entry points.

Website, user, and role pages currently maintain configuration records in memory. They lose data on restart and do not publish reverse proxies or establish permission isolation. Certificate issuance, user permission management, and runtime repair are unavailable. The application store and database pages show their feature status. Read the [architecture and feature boundaries](docs/architecture.en.md) before extending them; implement the service and storage before connecting the page.

## Run from source

Install Go 1.26.11 or later, Node.js 22.13+ on the 22.x line or 24+, and npm. System checks depend on Linux commands. Windows supports builds and frontend development; verify complete system checks in a Linux environment.

```bash
git clone https://github.com/QwQBiG/qwq-aiops.git
cd qwq-aiops
node scripts/build.mjs
```

The build script installs the locked frontend dependencies, builds and copies static assets, and writes `bin/qwq` (`bin/qwq.exe` on Windows). The frontend must be built before Go compilation so the binary contains the current pages.

Set the service configuration in the same terminal, then start it. This example uses Bash; see the [development guide](docs/development.en.md) for PowerShell commands.

```bash
export OPENAI_API_KEY='your-api-key'
export OPENAI_BASE_URL='https://api.openai.com/v1'
export OPENAI_MODEL='your-model-name'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
./bin/qwq web
```

Set `OPENAI_BASE_URL` to the provider's compatible endpoint and choose a model that provider serves. The binary does not load `.env` automatically: export variables or pass a JSON file with `--config`. See [deployment](docs/deployment.en.md) for local models and Docker Compose.

Open <http://localhost:8080>. The browser requests the HTTP Basic Auth credentials you configured. The server listens on `:8080` by default; use HTTPS and a controlled network entry point for remote access.

```bash
curl --fail http://localhost:8080/health
curl --fail --user operator http://localhost:8080/api/health
```

`/health` returns minimal JSON liveness information without authentication. It does not test database, Docker, or external model availability. `/api/health` is an authenticated deployment diagnostics endpoint.

Optional `gateway` mode forwards the same console through `PORT`. Its internal Web service listens only on `127.0.0.1:WEB_UI_PORT` (default `8899`) and uses the same entry point credentials. It does not start reserved microservices or expose a registry management port. See [deployment](docs/deployment.en.md) for an example.

## Configure authentication before use

Set `WEB_USER` and `WEB_PASSWORD` together. Configuring only one prevents startup. Leaving both empty keeps anonymous access available for existing local workflows. Set credentials before starting the server, and keep an anonymous instance off public networks.

File access is constrained by its mount and blocked directories, but the workbench still has operations privileges. Grant Docker socket, host directory, and terminal access only to trusted operators. The user management page does not replace entry point authentication.

## Repository layout

```text
cmd/qwq/            CLI entry point and startup orchestration
internal/           Web service, configuration, diagnostics, notifications, execution
frontend/           Vue views, routes, translations, dependency lockfile
scripts/            Shared build entry point
docs/               Current Chinese and English guides
.github/workflows/  Automated checks and builds
```

See [architecture](docs/architecture.en.md) for package responsibilities, request flow, and generated assets.

## Documentation and contributions

| Topic | 中文 | English |
| --- | --- | --- |
| Deployment, configuration, troubleshooting | [部署指南](docs/deployment.md) | [Deployment](docs/deployment.en.md) |
| Local development and verification | [开发指南](docs/development.md) | [Development](docs/development.en.md) |
| Structure and feature boundaries | [架构说明](docs/architecture.md) | [Architecture](docs/architecture.en.md) |
| Reporting issues and proposing changes | [贡献指南](CONTRIBUTING.md) | [Contributing](CONTRIBUTING_EN.md) |

Report problems through [Issues](https://github.com/QwQBiG/qwq-aiops/issues), including reproduction steps, your environment, and logs with secrets removed.
