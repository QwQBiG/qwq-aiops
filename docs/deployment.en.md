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

Alternatively, load a JSON file with `--config /path/to/config.json`. The `Config` struct in `internal/config/config.go` defines its fields, including `api_key`, `base_url`, `model`, `web_user`, `web_password`, `patrol_rules`, and `http_rules`. Nonempty environment variables override corresponding values. Explicit `--user`, `--password`, `--webhook`, and `--knowledge` flags take final precedence. Keep real credentials out of version control.

Startup checks static validity without probing external services. Model endpoints must be HTTP(S) base URLs without embedded credentials, a query, or a fragment. HTTP rules require absolute HTTP(S) URLs and a `code` of `0` (check for `200` by default) or `100` through `599`. Custom patrol commands must be nonempty. An explicitly configured knowledge file must be readable; startup reports an error rather than continuing without the expected context.

## Run the optional gateway

After setting the credentials and model endpoint above in the same terminal, you can instead run:

```bash
export PORT='8080'
export WEB_UI_PORT='8899'
./bin/qwq gateway
```

The entry point remains <http://localhost:8080>. The gateway authenticates with the configured credentials and forwards HTTP and WebSocket requests to internal `127.0.0.1:8899`, preserving the browser's `Host`. Choose an unused `WEB_UI_PORT` different from `PORT`. A failure to bind either listener ends startup; stopping the process stops both services. This mode proxies the current console and does not start microservices for other business packages.

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

Open <http://localhost:8081>. Compose manages only the workbench, binds its port to the host's `127.0.0.1`, and writes logs to the mounted `logs` directory. Website, user, and role records remain in memory and are lost on restart.

The old `database`, `cache`, and `monitoring` profiles and templates have been removed because the current workbench did not use them. Updating does not delete existing supporting containers or volumes. If you still need them, manage them with a separate Compose file and preserve their volumes. Unconnected business drafts remain available in Git history.

The image runs as a non-root user with UID/GID `1000:1000`. The mounted `logs` directory must permit that user to write. Docker actions also require daemon access. Configure actual user and group permissions rather than opening access to the entire host. Mounting the Docker socket with `:ro` does not restrict mutating Docker API calls.

## Notifications and one-shot commands

Configure DingTalk with `DINGTALK_WEBHOOK`, or set both `TELEGRAM_TOKEN` and `TELEGRAM_CHAT_ID`. The JSON fields are `webhook`, `telegram_token`, and `telegram_chat_id`. The Compose template passes these variables through. When both channels are configured, both receive the message; any failure returns an error. Sends are not retried automatically.

```bash
./bin/qwq status
./bin/qwq run 'printf "done\n"'
```

`status` waits for notification delivery confirmation and exits with a nonzero status if no channel is configured or delivery fails. After risk confirmation, `run` executes once, uses that attempt's output for failure diagnostics, and returns a nonzero status on failure. Suggested commands are not executed. Ctrl+C cancels the active command and diagnostic request.

Status reports and the dashboard share the system metric collector. Missing commands, execution failures, or unrecognized output produce `N/A` values and a list of causes in reports. Measured `0` values remain zero. Successful delivery means the notification service accepted the message; it does not establish that every metric was available.

The patrol button requests a manual patrol and status report. `202` means accepted; inspect the logs for progress and results. Repeated requests return `409` while that manual batch is running, or `503` if the service is not ready. Scheduled patrols and reports retain their own schedules.

Patrols distinguish observed anomalies from incomplete checks. Missing tools, insufficient permissions, failed commands, and unrecognized output appear as failures in the logs. A custom rule should exit with status `0` and no output when there is no anomaly. Nonempty output from a successful command becomes an anomaly; a nonzero exit is a failed check. Handle an expected “no matches” result explicitly in the script.

## Check the actual behavior

```bash
curl --fail http://localhost:8081/health
curl --fail --user operator http://localhost:8081/api/health
```

The first request should return JSON liveness information. The second prompts for a password. Protected endpoints should return `401` without credentials. Liveness does not test Docker, models, databases, or business data. Exercise the feature you intend to use and inspect the logs as well.

Deployment diagnostics inspect the process's effective configuration and embedded assets. Missing notification configuration means that optional feature is disabled. Actual model and Docker requests are still needed to verify those services. Certificate issuance, user permission management, and automatic runtime repair are not connected and return `501`; change configuration, rebuild, and restart through the deployment workflow.

Compose requires nonempty `WEB_USER` and `WEB_PASSWORD`. When running the binary directly, configure them together: one missing value prevents startup, while both empty retain anonymous access. Recreate a container after changing Compose environment variables, for example with `docker compose up -d qwq`. A plain `restart` does not refresh its environment.

## Troubleshooting

| Symptom | First checks |
| --- | --- |
| Startup reports a missing key | Confirm the process environment; direct execution does not load `.env` |
| Startup reports invalid settings or a knowledge file error | Check the named field, endpoint, status code, command, or file path. The service user must be able to read the knowledge file |
| Login fails after setting a password | Pass both credentials, recreate the container, and check credentials cached by the browser |
| Missing pages or placeholder assets | Run the full build and recompile Go; `npm run build` alone cannot update an existing binary |
| Docker actions fail | Check the client, daemon address, and container user's socket permissions |
| Model requests fail | Check endpoint, model name, key, and network access from the container |
| Dashboard metrics show “Unavailable” | Read the collection error and check the required Linux tools. Container metrics describe the service's runtime environment |
| A write returns `403` | Check credentials and `Origin`, and confirm the proxy preserves the original `Host` |
| An API returns `404` or `501` | `404` means no such endpoint; `501` means the management capability is not connected. See [feature boundaries](architecture.en.md) |
| File editing fails or changes format | Editing accepts regular UTF-8 text files up to 2 MiB; check process permissions. JSON and HTML are edited as raw text |
| Data disappears after restart | Check the [feature boundaries](architecture.en.md) for in-memory endpoints |

For source runs, inspect `logs/qwq.log` in the working directory. Containers use the same mounted path and `docker compose logs`. Existing root-level `qwq.log` files remain untouched; new logs go to `logs`. Preserve data and logs while investigating; clearing volumes or global build caches is not a prerequisite for diagnosis.
