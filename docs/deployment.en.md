# Deployment and configuration

[简体中文](deployment.md) · [Back to README](../README_EN.md)

Start in a trusted test environment, then grant only the host directory, Docker, and terminal access you need. Workbench actions use the service process's privileges. Treat HTTP Basic Auth credentials as administrative access to the workbench.

## Run the binary

Build with the [development guide](development.en.md), then run these commands from the repository root on Linux:

```bash
export OPENAI_API_KEY='your-api-key'
export OPENAI_BASE_URL='https://api.openai.com/v1'
export OPENAI_MODEL='your-model-name'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
export PORT='8080'
./bin/qwq web
```

The binary does not read `.env`. Variables must be present in the process environment. The server listens on `:PORT`, defaulting to `8080`. Put HTTPS in front of remote access and restrict the network entry point.

Preserve the browser's `Host` through a reverse proxy so writes and WebSocket connections pass origin checks. Foreign or invalid `Origin` values return `403`. The service does not substitute forwarding headers such as `X-Forwarded-Host` for the original `Host`.

Alternatively, load a JSON file with `--config /path/to/config.json`. The `Config` struct in `internal/config/config.go` defines its fields, including `api_key`, `base_url`, `model`, `web_user`, `web_password`, `patrol_rules`, and `http_rules`. Environment variables override corresponding values. Keep real credentials out of version control.

## Connect a local model

A service running on the host can use this Bash configuration:

```bash
export AI_PROVIDER='ollama'
export OLLAMA_HOST='http://localhost:11434'
export OLLAMA_MODEL='your-installed-model'
export WEB_USER='operator'
export WEB_PASSWORD='replace-with-a-long-unique-password'
./bin/qwq web
```

This mode sends requests through the local service's `/v1` compatible interface. Set `OLLAMA_HOST` to its root address and install the model beforehand. Accepted configuration does not establish that a model is installed or reachable.

Inside a container, `localhost` refers to that container. Docker Desktop commonly exposes the host through `host.docker.internal`. On Linux, configure a reachable host address or put the model service on the same container network. A bridge address from another machine may not apply to yours.

## Use Docker Compose

Install Docker with Linux container support and Compose V2. Copy the template from the repository root, then edit `.env` with the endpoint, model, and entry point credentials:

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

In PowerShell, use `Copy-Item -LiteralPath '.env.example' -Destination '.env'`. Compose reads interpolation values from `.env`, then passes the variables listed in `docker-compose.yml` into the container. The binary itself still does not load `.env`.

```bash
docker compose config --quiet
docker compose up -d --build qwq
docker compose ps
docker compose logs --tail 100 qwq
curl --fail http://localhost:8081/health
```

Open <http://localhost:8081>. Only the workbench starts by default, and service ports bind to the host's `127.0.0.1`. MySQL, Redis, and monitoring services use the `database`, `cache`, and `monitoring` profiles. For example, `docker compose --profile monitoring up -d` also starts Prometheus and Grafana. Configure access controls before exposing supporting services. Starting a database does not make in-memory website, user, or role data persistent.

Set `MYSQL_ROOT_PASSWORD` and `MYSQL_PASSWORD` before enabling `database`, and `GRAFANA_ADMIN_PASSWORD` before enabling `monitoring`. The current Prometheus configuration scrapes only its own metrics. Starting these containers does not connect workbench metrics or provision Grafana dashboards.

The image runs as a non-root user. Mounted `data`, `logs`, and `backups` directories must permit that user to write. Docker actions also require daemon access. Configure actual user and group permissions rather than opening access to the entire host. Mounting the Docker socket with `:ro` does not restrict mutating Docker API calls.

## Check the actual behavior

```bash
curl --fail http://localhost:8081/health
curl --fail --user operator http://localhost:8081/api/health
```

The first request should return JSON liveness information. The second prompts for a password. Protected endpoints should return `401` without credentials. Liveness does not test Docker, models, databases, or business data. Exercise the feature you intend to use and inspect the logs as well.

Compose requires nonempty `WEB_USER` and `WEB_PASSWORD`. When running the binary directly, configure them together: one missing value prevents startup, while both empty retain anonymous access. Recreate a container after changing Compose environment variables, for example with `docker compose up -d qwq`. A plain `restart` does not refresh its environment.

## Troubleshooting

| Symptom | First checks |
| --- | --- |
| Startup reports a missing key | Confirm the process environment; direct execution does not load `.env` |
| Login fails after setting a password | Pass both credentials, recreate the container, and check credentials cached by the browser |
| Missing pages or placeholder assets | Run the full build and recompile Go; `npm run build` alone cannot update an existing binary |
| Docker actions fail | Check the client, daemon address, and container user's socket permissions |
| Model requests fail | Check endpoint, model name, key, and network access from the container |
| Data disappears after restart | Check the [feature boundaries](architecture.en.md) for in-memory endpoints |

For source runs, inspect `qwq.log` in the working directory. For containers, start with `docker compose logs`. Preserve data and logs while investigating; clearing volumes or global build caches is not a prerequisite for diagnosis.
