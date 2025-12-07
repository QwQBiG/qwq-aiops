# 端口修改指南

## 问题说明

如果您的本地 8080 端口被占用，qwq 平台无法正常启动，您可以按照以下方法修改端口。

## 解决方案

### ✅ 方案 1：修改 docker-compose.yml（推荐）

我已经帮您修改了 `docker-compose.yml` 文件，将端口映射从 `8080:8080` 改为 `8081:8080`。

**使用方法**：

```bash
# 1. 停止现有服务（如果正在运行）
docker-compose down

# 2. 构建并启动服务（首次运行需要构建镜像）
docker-compose up -d --build

# 3. 查看构建和启动日志
docker-compose logs -f

# 4. 访问系统
# 前端界面: http://localhost:8081
# API 文档: http://localhost:8081/api/docs
```

**注意**：首次启动会自动构建 Docker 镜像，这可能需要几分钟时间。

### 方案 2：使用其他端口

如果 8081 也被占用，您可以修改为任意可用端口：

**编辑 `docker-compose.yml`**：

```yaml
services:
  qwq:
    ports:
      - "8082:8080"  # 改为 8082 或其他可用端口
```

然后重启服务：

```bash
docker-compose down
docker-compose up -d
```

### 方案 3：查找并停止占用端口的进程

**Windows 系统**：

```cmd
# 查找占用 8080 端口的进程
netstat -ano | findstr :8080

# 停止进程（替换 <PID> 为实际进程ID）
taskkill /PID <PID> /F
```

**Linux/macOS 系统**：

```bash
# 查找占用 8080 端口的进程
lsof -i :8080

# 停止进程（替换 <PID> 为实际进程ID）
kill -9 <PID>
```

## 验证端口是否可用

**Windows**：

```cmd
netstat -ano | findstr :8081
```

如果没有输出，说明端口可用。

**Linux/macOS**：

```bash
lsof -i :8081
```

如果没有输出，说明端口可用。

## 常见端口占用情况

- **8080**: 通常被 Tomcat、Jenkins 等应用占用
- **8081**: 通常可用
- **8082-8089**: 通常可用

## 更新后的访问地址

修改端口后，请使用新的地址访问：

- **前端界面**: http://localhost:8081
- **API 文档**: http://localhost:8081/api/docs
- **健康检查**: http://localhost:8081/api/health

## 需要帮助？

如果遇到问题，请查看：

- [部署指南](docs/deployment-guide.md) - 完整的部署说明
- [故障排查指南](docs/troubleshooting-guide.md) - 常见问题解决方案
- [GitHub Issues](https://github.com/QwQBiG/qwq-aiops/issues) - 提交问题

---

**提示**：修改端口后，记得更新 Nginx 配置（如果使用）和防火墙规则。
