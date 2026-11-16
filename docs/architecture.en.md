# Architecture and directory responsibilities

[简体中文](architecture.md) · [Back to README](../README_EN.md)

A single Go program provides the CLI and Web workbench. The `web` command starts a `net/http` server. The `gateway` command forwards requests to a Web service in the same process that listens only on loopback. Both modes use the same handlers and entry point credentials. Start with `cmd/qwq/main.go` and `internal/server` when tracing a request.

## Follow a Web request

```text
Vue page in the browser
  ├─ HTTP /api/... → entry point authentication → internal/server handler
  ├─ WebSocket /ws/chat → authentication and origin checks → conversation and commands
  └─ pages and assets → internal/server/dist embedded in the Go binary
```

The entry point loads configuration, initializes logging and the model client, then starts system checks and the Web server. Monitoring and container actions invoke system commands. Notifications use the configured channels. Public `/health` gives a minimal liveness response; `/api/health` exposes deployment diagnostics through authentication.

`NewHandler` creates an independent router without registering process-wide default routes or starting collectors. `Run` owns the listener, collection, and shutdown. Startup failures reach the caller, and cancellation reaches system commands, HTTP checks, and model requests. Unknown `/api` and `/ws` paths return `404` instead of a frontend page.

## Where code belongs

| Directory | Responsibility |
| --- | --- |
| `cmd/qwq` | Command registration, initialization, and system check scheduling |
| `internal/server` | Independent routes, service lifecycle, assets, authentication, and business HTTP adapters |
| `internal/config` | JSON configuration, environment overrides, and deployment configuration checks |
| `internal/agent`, `executor`, `security` | Conversations, command execution, and risk checks |
| `internal/monitor`, `notify`, `logger`, `utils` | System checks, notifications, logging, and low level utilities |
| `internal/gateway` | Optional console reverse proxy, authentication, and deterministic routing |
| `internal/deployment`, `platform` | Diagnostics for effective configuration, embedded assets, and runtime environment |
| `frontend/src/views` | Pages loaded directly by the router |
| `frontend/src/components` | Shared interface components such as feature status notices |
| `frontend/src/lib` | Error messages, polling, terminal sessions, and safe content rendering |
| `frontend/src/i18n` | Chinese and English interface text |
| `scripts` | Shared build entry point |
| `docs` | Current Chinese and English guides |
| `.github/workflows` | Tests, binary builds, and container builds |

Keep service implementation separate from HTTP handling, then adapt it through a handler that validates input and returns an accurate response. For features with data, define storage and restart behavior first. A menu's feature status does not replace a service implementation.

## Current feature boundaries

| Web entry point | Behavior and dependencies |
| --- | --- |
| Dashboard, logs, system checks | Use commands in the service's runtime environment; complete checks require Linux tools. Failed dashboard measurements show “Unavailable” and the cause, while measured zero values remain zero |
| Container list and actions | Invoke Docker commands; require the client, a reachable daemon, and permissions |
| File browsing and editing | Use `/hostfs` as the root when present, otherwise `/`; enforce path and blocked directory checks. Editing accepts regular UTF-8 text files up to 2 MiB and preserves permissions and Unix UID/GID. Closing or leaving with unsaved edits requires confirmation |
| Terminal conversations | Call a compatible model endpoint; automatic execution permits single read-only diagnostic commands, which may still read information available to the service process |
| Websites, users, roles | Maintain configuration records in memory and lose them on restart; do not publish reverse proxies or establish user authorization. Certificate issuance, renewal, and user permission requests return `501` |
| Application store and database pages | Describe current integration requirements; management writes return `501` rather than claiming deployment or connection success |
| Runtime repair | `/api/deployment/repair` returns `501`; operators change configuration and rebuild assets through the deployment workflow |

Web entry points share HTTP Basic Auth. The user management page and entry point credentials are separate mechanisms. Persistent permissions, multiple authenticated users, and tenant isolation require further authentication and storage integration.

State changes use write requests: `/api/container/action`, `/api/files/action`, and `/api/trigger` accept only `POST`; `GET` returns `405`. Browser writes and WebSocket connections check `Origin`. Its host and port must match the request's `Host`; invalid or foreign origins return `403`. Command-line clients may omit `Origin` but must still pass authentication.

A reverse proxy terminating HTTPS must preserve the browser's `Host`. The service accepts `http` and `https` origins and does not reconstruct them from forwarding headers. Rewriting `Host` can cause valid browser writes to be rejected.

The built-in gateway forwards only to `127.0.0.1:WEB_UI_PORT`, preserving the original `Host` and authentication for HTTP and WebSocket requests. It does not register reserved microservices, generate unimplemented API documentation, or open a registry management port.

Automatic command policy rejects pipes, redirection, command composition, and state-changing subcommands. For example, `docker ps` can run as a diagnostic, while `docker restart` requires the explicit container action. Model replies render as sanitized Markdown, and file content is served as plain text. These boundaries do not replace entry point authentication or host permissions.

Resource detail handlers accept only defined complete paths. Unknown website, user, or role subpaths return `404` without updating or deleting the parent record. Pages lock inputs and closing while a save is pending, and preserve input after failed submissions.

Deployment diagnostics inspect effective configuration, embedded assets, and platform information. They do not run repair templates or contact Docker, model, or notification services. Frontend checks verify that local scripts and styles referenced by the entry page exist and can be read. Verify external services through their actual workflows as well.

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

See [development](development.en.md) for commands and [deployment](deployment.en.md) for runtime constraints.
