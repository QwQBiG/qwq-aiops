# Docker 构建修复说明

## 问题

Docker 构建失败，错误信息：
```
npm error The `npm ci` command can only install with an existing package-lock.json
```

## 原因

前端目录缺少 `package-lock.json` 文件，而 `npm ci` 命令需要这个文件。

## 解决方案

### 方案 1：使用 npm install（已应用）✅

Dockerfile 已修改为使用 `npm install` 代替 `npm ci`。

**优点**：
- 立即可用，无需额外操作
- 不需要 package-lock.json

**缺点**：
- 构建速度稍慢
- 依赖版本可能不完全一致

### 方案 2：生成 package-lock.json（推荐）

在本地生成 `package-lock.json` 并提交到 Git。

**步骤**：

```bash
# 进入前端目录
cd frontend

# 生成 package-lock.json
npm install

# 返回项目根目录
cd ..

# 提交到 Git
git add frontend/package-lock.json
git commit -m "chore: add package-lock.json for reproducible builds"
```

然后将 Dockerfile 改回使用 `npm ci`：

```dockerfile
# 安装依赖（使用 npm ci 更快更可靠）
RUN npm ci
```

**优点**：
- 构建速度更快
- 依赖版本完全一致
- 更符合最佳实践

**缺点**：
- 需要额外的步骤

## 当前状态

✅ Dockerfile 已修改为使用 `npm install`，可以立即构建。

## 建议

建议采用方案 2，生成 `package-lock.json` 文件：

1. 在本地运行 `cd frontend && npm install`
2. 提交生成的 `package-lock.json`
3. 将 Dockerfile 改回使用 `npm ci`

这样可以获得更快的构建速度和更好的依赖管理。
