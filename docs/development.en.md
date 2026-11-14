# Local development and verification

[简体中文](development.md) · [Back to README](../README_EN.md)

Use Go 1.23 or later, Node.js 22, and npm. Complete system checks require Bash and Linux tools. Windows can verify builds, HTTP behavior, and the frontend, but those results do not establish that Linux system checks work.

## Build the complete program

From the repository root, run:

```bash
node scripts/build.mjs
```

Windows, Linux, and macOS use the same sequence: `npm ci`, frontend build, embedded asset sync, then `go build`. Output is `bin/qwq`, or `bin/qwq.exe` on Windows. The script locates the repository from its own path; you do not need to copy frontend files manually.

To prepare only the frontend assets:

```bash
node scripts/build.mjs --frontend-only
```

Tracked resource placeholders let Go packages compile, but do not supply the full frontend. Prepare assets before running `go build` directly. Recompile Go after changes to `frontend/src` so the binary contains the updated pages.

## Start the backend

This PowerShell example configures the current terminal. Replace the endpoint and model with values your provider supports:

```powershell
$env:OPENAI_API_KEY = 'your-api-key'
$env:OPENAI_BASE_URL = 'https://api.openai.com/v1'
$env:OPENAI_MODEL = 'your-model-name'
$env:WEB_USER = 'operator'
$env:WEB_PASSWORD = 'replace-with-a-long-unique-password'
go run ./cmd/qwq web
```

In Bash, use `export NAME='value'`; see [deployment](deployment.en.md) for a full example. `.env` supplies Compose interpolation values and is not loaded when running the program directly. The backend defaults to port `8080`.

## Start the frontend development server

Leave the backend running and open another terminal:

```bash
cd frontend
npm ci
npm run dev
```

Open <http://localhost:5173>. Vite proxies `/api` and `/ws` to `localhost:8080`; requests use the backend's entry point credentials. The development proxy and embedded production pages are separate paths. Check both when changing proxy behavior.

## Verify according to the change

```bash
go test ./internal/config ./internal/server
go test ./...
```

The first command targets configuration and Web changes. The second checks the entire Go module. When tests fail, preserve the package name, command, and original error. Distinguish compilation failures, failed assertions, and unavailable external dependencies.

Run at least `npm run build` for frontend changes. Run the complete `node scripts/build.mjs` for embedding or packaging changes. Configuration and route changes also require starting the program and checking accepted and rejected requests:

```bash
curl --fail http://localhost:8080/health
curl --include http://localhost:8080/api/health
curl --fail --user operator http://localhost:8080/api/health
```

Use `curl.exe` in Windows PowerShell to avoid older command aliases. With authentication enabled, the second request should return `401`; the third prompts for a password. Open the page, confirm that at least one static asset loads, then inspect `qwq.log`. Liveness does not exercise Docker or model requests; test those features separately.

For Docker changes:

```bash
docker compose config --quiet
docker compose build qwq
docker compose up -d qwq
docker compose logs --tail 100 qwq
```

Complete the [deployment configuration](deployment.en.md) first. If Docker is unavailable locally, record that container execution remains unverified and inspect the remote build result. A valid configuration is not a substitute for a running image.

## What continuous integration checks

`.github/workflows/build.yml` uses Go 1.23 and Node.js 22 on Ubuntu. It builds real frontend assets before running race detection and coverage tests for the entire module. Binary jobs reuse those assets and disable CGO for cross compilation. `docker-publish.yml` builds Linux images for multiple architectures; PR builds do not push images.

This describes the workflow configuration, not the outcome for every commit. Inspect checks for the actual PR commit. Use original failure logs to locate problems rather than excluding failing packages and claiming the complete module passed.

Earlier deployment and diagnostic scripts remain in [`scripts/legacy`](../scripts/legacy/README.md). They depend on the old layout and configuration and are retained only for reference. Use `scripts/build.mjs` for current builds and the root Compose configuration for deployment.

## Files and contributions

Use UTF-8 text. `.gitattributes` sets LF for source, Markdown, and Shell files, and CRLF for Windows `.bat` files. Exclude build output, dependencies, and runtime data from commits.

Put new behavior in its business package and keep tests beside it. Update Chinese and English together when changing public setup steps or configuration. Both versions should describe the same behavior and limits. See [contributing](../CONTRIBUTING_EN.md) for change scope and verification reports.
