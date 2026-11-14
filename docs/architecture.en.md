# Architecture and directory responsibilities

[简体中文](architecture.md) · [Back to README](../README_EN.md)

A single Go program provides the CLI and Web workbench. The `web` command starts a `net/http` server. The separate `gateway` command starts a `net/http` gateway and a Web service. Start with `cmd/qwq/main.go` and `internal/server` when tracing the `web` command.

## Follow a Web request

```text
Vue page in the browser
  ├─ HTTP /api/... → entry point authentication → internal/server handler
  ├─ WebSocket /ws/chat → authentication and origin checks → conversation and commands
  └─ pages and assets → internal/server/dist embedded in the Go binary
```

The entry point loads configuration, initializes logging and the model client, then starts system checks and the Web server. Monitoring and container actions invoke system commands. Notifications use the configured channels. Public `/health` gives a minimal liveness response; `/api/health` exposes deployment diagnostics through authentication.

## Where code belongs

| Directory | Responsibility |
| --- | --- |
| `cmd/qwq` | Command registration, initialization, and system check scheduling |
| `internal/server` | Current Web routes, embedded assets, authentication, health checks, and HTTP adapters |
| `internal/config` | JSON configuration, environment overrides, and deployment configuration checks |
| `internal/agent`, `executor`, `security` | Conversations, command execution, and risk checks |
| `internal/monitor`, `notify`, `logger`, `utils` | System checks, notifications, logging, and low level utilities |
| `internal/gateway`, `registry` | Optional gateway, routing, and service discovery |
| Other `internal` business packages | Container deployment, databases, websites, backup, clusters, and related services |
| `frontend/src/views` | Pages loaded directly by the router |
| `frontend/src/components` | Components and earlier page implementations; follow the router to find active views |
| `frontend/src/i18n` | Chinese and English interface text |
| `scripts` | Shared build entry point; `legacy` preserves scripts that depend on the old layout |
| `config` | Configuration for optional Prometheus, MySQL, and other supporting services |
| `docs` | Current guides; `archive` contains historical documents |
| `.github/workflows` | Tests, binary builds, and container builds |

Implement new behavior in the relevant business package, then adapt it through a Web handler that validates input and returns an accurate response. A package's presence does not mean the `web` command uses it. A menu entry does not establish that its service is connected.

## Current feature boundaries

| Web entry point | Behavior and dependencies |
| --- | --- |
| Dashboard, logs, system checks | Use commands in the service's runtime environment; complete checks require Linux tools |
| Container list and actions | Invoke Docker commands; require the client, a reachable daemon, and permissions |
| File browsing and editing | Use `/hostfs` as the root when present, otherwise `/` in the process filesystem; apply path and blocked directory checks |
| Terminal conversations | Call a compatible endpoint and may execute commands; restrict access to trusted operators |
| Websites, users, roles | Some endpoints use process memory and lose data on restart; complete tenant and permission isolation is absent |
| Application store and database pages | Not connected to their business services; lists are empty and management writes return `501` rather than claiming success |
| Backup, clusters, extended monitoring | Independent packages exist; inspect each entry point and its tests before use |

Web entry points share HTTP Basic Auth. The user management page and entry point credentials are separate mechanisms. Persistent permissions, multiple authenticated users, and tenant isolation require further authentication and storage integration.

State changes use write requests: `/api/container/action`, `/api/files/action`, and `/api/trigger` accept only `POST`; `GET` returns `405`. Browser writes and WebSocket connections check `Origin`. Its host and port must match the request's `Host`; invalid or foreign origins return `403`. Command-line clients may omit `Origin` but must still pass authentication.

A reverse proxy terminating HTTPS must preserve the browser's `Host`. The service accepts `http` and `https` origins and does not reconstruct them from forwarding headers. Rewriting `Host` can cause valid browser writes to be rejected.

## How the frontend reaches the binary

```text
frontend/package-lock.json
  → npm ci → Vite build → frontend/dist
  → copy to internal/server/dist → go build → bin/qwq
```

`internal/server/dist` contains generated files. Tracked placeholders let the package compile; they do not provide a complete workbench. `node scripts/build.mjs` performs this sequence, while `--frontend-only` prepares embedded assets without compiling Go. Recompile the Go binary after frontend changes: an existing binary cannot read newly built pages.

Generated output belongs in `frontend/dist`, `internal/server/dist`, and `bin`. Do not hand edit hashed asset files. Commit dependency lockfiles and source; keep runtime credentials, data, and logs outside version control.

## Verify the connected behavior

Build the complete program, then follow the actual request with a browser or HTTP client. Check authentication, the service called by the handler, the resulting state in that service, and the error response. A business package's tests establish the behavior they exercise. They do not establish that menus, HTTP handlers, external dependencies, and storage are connected.

See [development](development.en.md) for commands and [deployment](deployment.en.md) for runtime constraints. Historical implementation reports remain in the [archive](archive/README.md); they are not evidence of current feature completion.
