# 历史脚本 / Historical scripts

这里保存早期部署、重建和诊断脚本，便于查阅原来的做法。这些脚本依赖旧目录结构、配置和资源名称，未按当前流程逐项适配；不要把它们作为当前操作指南。

当前入口：

- 构建：在仓库根目录运行 `node scripts/build.mjs`。
- 部署：按[部署指南](../../docs/deployment.md)配置后，运行 `docker compose up -d --build qwq`。
- 开发与验证：[开发指南](../../docs/development.md)。

These earlier deployment, rebuild, and diagnostic scripts preserve the old approach for reference. They depend on the former directory layout, configuration, and asset names, and have not been adapted to every current path. Use the current entry points instead:

- Build: run `node scripts/build.mjs` from the repository root.
- Deploy: follow [deployment](../../docs/deployment.en.md), then run `docker compose up -d --build qwq`.
- Develop and verify: see [development](../../docs/development.en.md).
