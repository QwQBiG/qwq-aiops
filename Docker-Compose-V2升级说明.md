# Docker Compose V2 升级说明

## 问题说明

您遇到的 `KeyError: 'ContainerConfig'` 错误是因为使用了旧版的 `docker-compose` (V1)。

### 版本差异

| 版本 | 命令 | 语言 | 状态 |
|------|------|------|------|
| V1 | `docker-compose` | Python | ❌ 已废弃 |
| V2 | `docker compose` | Go | ✅ 推荐使用 |

## ✅ 已修复

我已经将所有脚本和文档中的命令从 `docker-compose` 更新为 `docker compose`（V2 版本）。

## 🚀 现在使用新命令

### 所有命令都改为

```bash
# 旧命令（V1，不要用）
docker-compose up -d        ❌

# 新命令（V2，推荐）
docker compose up -d        ✅
```

### 常用命令对照

| 功能 | 旧命令 (V1) | 新命令 (V2) |
|------|-------------|-------------|
| 启动 | `docker-compose up -d` | `docker compose up -d` |
| 停止 | `docker-compose down` | `docker compose down` |
| 构建 | `docker-compose build` | `docker compose build` |
| 查看日志 | `docker-compose logs -f` | `docker compose logs -f` |
| 查看状态 | `docker-compose ps` | `docker compose ps` |
| 重启 | `docker-compose restart` | `docker compose restart` |

## 📝 已更新的文件

所有脚本和文档都已更新为使用 `docker compose`：

### 核心脚本
1. ✅ `一键部署.sh`
2. ✅ `rebuild.sh` / `rebuild.bat`
3. ✅ `start.sh` / `start.bat`
4. ✅ `fix-config.sh` / `fix-config.bat`

### 文档
5. ✅ `README.md`
6. ✅ `README_EN.md`
7. ✅ `docs/deployment-guide.md`
8. ✅ `快速开始.md`
9. ✅ `一键部署说明.md`
10. ✅ `START_HERE.md`
11. ✅ 所有其他相关文档

## 🔧 如何检查您的版本

```bash
# 检查 Docker Compose 版本
docker compose version

# 应该看到类似输出：
# Docker Compose version v2.x.x
```

## 💡 如果您还在使用 V1

### 方法 1：安装 Docker Compose V2（推荐）

Docker Compose V2 已经集成在 Docker Desktop 和最新的 Docker Engine 中。

**Linux 系统**：

```bash
# 更新 Docker
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 验证安装
docker compose version
```

**macOS / Windows**：

更新 Docker Desktop 到最新版本即可，V2 已经内置。

### 方法 2：创建别名（临时方案）

如果暂时无法升级，可以创建别名：

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
alias docker-compose='docker compose'

# 重新加载配置
source ~/.bashrc
```

## 🎯 使用一键部署脚本

现在直接运行一键部署脚本即可，它已经使用了新命令：

```bash
chmod +x 一键部署.sh
sudo ./一键部署.sh
```

## ⚠️ 注意事项

1. **不要混用**：不要在同一个项目中混用 V1 和 V2 命令
2. **配置文件兼容**：`docker-compose.yml` 文件在两个版本中都可以使用
3. **推荐升级**：建议升级到 V2，V1 已经不再维护

## 📚 更多信息

- Docker Compose V2 文档：https://docs.docker.com/compose/
- 迁移指南：https://docs.docker.com/compose/migrate/

---

**所有命令已更新为 V2 版本，不会再出现 KeyError 问题！** ✅
