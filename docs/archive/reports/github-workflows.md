# GitHub Workflows 说明

本项目包含 2 个 GitHub Actions 工作流，用于自动化构建、测试和发布。

## 工作流列表

### 1. Build and Test (`.github/workflows/build.yml`)

**触发条件**:
- Push 到 `main` 或 `develop` 分支
- Pull Request 到 `main` 分支

**功能**:
- 运行所有单元测试和属性测试
- 生成测试覆盖率报告
- 构建多平台二进制文件（Linux、Windows、macOS）
- 支持 amd64 和 arm64 架构

**产物**:
- 测试覆盖率报告（上传到 Codecov）
- 多平台可执行文件（作为 Artifacts）

### 2. Docker Build and Publish (`.github/workflows/docker-publish.yml`)

**触发条件**:
- Push 到 `main` 或 `develop` 分支
- 创建版本标签（`v*.*.*`）
- Pull Request 到 `main` 分支
- 手动触发（workflow_dispatch）

**功能**:
- 构建多架构 Docker 镜像（linux/amd64, linux/arm64）
- 发布到 GitHub Container Registry (ghcr.io)
- 自动生成镜像标签（分支名、PR 号、版本号、SHA、latest）
- 使用 GitHub Actions 缓存加速构建
- 生成构建证明（attestation）

**镜像标签规则**:
- `main` 分支: `latest`, `main`, `main-<sha>`
- `develop` 分支: `develop`, `develop-<sha>`
- 版本标签: `v1.0.0`, `1.0`, `1`
- Pull Request: `pr-<number>`

## 使用说明

### 查看工作流运行状态

访问 [Actions](https://github.com/yourusername/qwq/actions) 页面查看所有工作流的运行状态。

### 手动触发 Docker 构建

1. 进入 [Actions](https://github.com/yourusername/qwq/actions) 页面
2. 选择 "Docker Build and Publish" 工作流
3. 点击 "Run workflow" 按钮
4. 选择分支并点击 "Run workflow"

### 拉取 Docker 镜像

```bash
# 拉取最新版本
docker pull ghcr.io/yourusername/qwq:latest

# 拉取特定版本
docker pull ghcr.io/yourusername/qwq:v1.0.0

# 拉取开发版本
docker pull ghcr.io/yourusername/qwq:develop
```

### 下载构建产物

1. 进入 [Actions](https://github.com/yourusername/qwq/actions) 页面
2. 选择一个成功的 "Build and Test" 运行
3. 在 "Artifacts" 部分下载对应平台的二进制文件

## 工作流优化

### 缓存策略

- **Go 模块缓存**: 使用 `actions/cache` 缓存 `~/go/pkg/mod`
- **Docker 构建缓存**: 使用 GitHub Actions 缓存（`type=gha`）