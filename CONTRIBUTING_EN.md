# Contributing

[简体中文](CONTRIBUTING.md) · [Back to README](README_EN.md)

Before changing code, identify the affected path: the CLI, the `web` service, `gateway`, or an independent business package that is not connected to a page. Menu entries and historical completion reports do not replace reproduction.

## Report a problem

Include the following in an [Issue](https://github.com/QwQBiG/qwq-aiops/issues):

- Commit or branch, operating system, Go and Node.js versions, or Docker/Compose versions.
- Minimal steps from startup to the failure, expected behavior, and actual behavior.
- Original errors, relevant logs, and configuration field names, with keys, passwords, tokens, and business data removed.
- Verification commands you tried and their results, including anything you could not verify.

For authentication bypasses, host access, or other exploitable issues, keep real credentials, host details, and active exploitation paths out of public issues.

## Prepare a change

Create a separate branch from the current `main`. Build using the [development guide](docs/development.en.md), reproduce the original behavior, then change the package responsible for it. Update Chinese and English documentation together when configuration or user steps change.

Keep the scope focused. The [architecture guide](docs/architecture.en.md) describes directory responsibilities: Web handlers own the request boundary, while business rules belong in the relevant service package. Exclude generated assets, dependency directories, runtime data, and credentials from commits.

Use UTF-8 text and follow `.gitattributes` for line endings. Format Go files with `gofmt` and follow neighboring frontend code conventions.

## Verify and open a PR

Run tests directly related to the change, extending to `go test ./...` when needed. Build frontend changes. For configuration, HTTP routes, or embedded assets, start the complete program and exercise both accepted and rejected paths. Docker changes need image builds and startup log checks.

A PR should make sense to someone who has not read the issue discussion. Describe the concrete problem and resulting behavior first, then give commands actually run, results, and unverified limits. Link the issue and leave abandoned approaches out of the final description.

Review the complete diff before committing and preserve unrelated existing work. Historical reports live in the [documentation archive](docs/archive/README.md). State which code and environment new verification results cover.
