# qwq AIOps 跨平台部署指南

## Windows 开发 → Ubuntu 部署

本指南帮助你在 Windows 上开发，然后部署到 Ubuntu 服务器。

### 前置要求

#### Windows 开发环境
- Windows 10/11
- Docker Desktop for Windows
- Git
- 代码编辑器（VS Code 推荐）

#### Ubuntu 部署环境
- Ubuntu 20.04+ 或 Debian 11+
- Docker Engine 20.10+
- Docker Compose V2
- 至少 2GB 内存，10GB 磁盘空间

---

## 方式一：使用 Docker 部署（推荐）

### 1. 在 Windows 上准备代码

```powershell
# 克隆或准备代码
git clone <your-repo>
cd qwqOps

# 确保代码已提交（可选，用于版本控制）
git add .
git commit -m "准备部署"
```

### 2. 传输代码到 Ubuntu 服务器

**方式 A：使用 Git（推荐）**
```bash
# 在 Ubuntu 服务器上
git clone <your-repo>
cd qwqOps
```

**方式 B：使用 SCP**
```powershell
# 在 Windows PowerShell 中
scp -r . user@ubuntu-server:/path/to/qwqOps
```

**方式 C：使用压缩包**
```powershell
# 在 Windows 上打包
Compress-Archive -Path . -DestinationPath qwqOps.zip -Exclude node_modules,dist,.git

# 传输到 Ubuntu
scp qwqOps.zip user@ubuntu-server:/tmp/

# 在 Ubuntu 上解压
ssh user@ubuntu-server
cd /path/to
unzip /tmp/qwqOps.zip
```

### 3. 在 Ubuntu 上部署

```bash
# 进入项目目录
cd qwqOps

# 给部署脚本执行权限
chmod +x deploy.sh

# 运行部署脚本
./deploy.sh
```

部署脚本会自动完成：
- ✅ 环境检查
- ✅ 配置创建
- ✅ 镜像构建
- ✅ 服务启动
- ✅ 健康检查

### 4. 访问服务

部署成功后，访问：
- 前端界面: `http://ubuntu-server-ip:8081`
- API 文档: `http://ubuntu-server-ip:8081/api/docs`

---

## 方式二：本地构建 + 传输镜像

如果 Ubuntu 服务器网络较慢，可以在 Windows 上构建镜像，然后传输到 Ubuntu。

### 1. 在 Windows 上构建镜像

```powershell
# 在 Windows PowerShell 中
docker build -t qwq-aiops:latest .

# 导出镜像
docker save qwq-aiops:latest -o qwq-aiops.tar
```

### 2. 传输镜像到 Ubuntu

```powershell
# 在 Windows PowerShell 中
scp qwq-aiops.tar user@ubuntu-server:/tmp/
```

### 3. 在 Ubuntu 上加载镜像

```bash
# SSH 到 Ubuntu 服务器
ssh user@ubuntu-server

# 加载镜像
docker load -i /tmp/qwq-aiops.tar

# 修改 docker-compose.yml，使用本地镜像
# 将 build: . 改为 image: qwq-aiops:latest

# 启动服务
docker compose up -d
```

---

## 方式三：使用 CI/CD 自动部署

### GitHub Actions 示例

创建 `.github/workflows/deploy.yml`:

```yaml
name: Build and Deploy

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: docker build -t qwq-aiops:latest .
      
      - name: Deploy to server
        uses: appleboy/scp-action@master
        with:
          host: ${{ secrets.HOST }}
          username: ${{ secrets.USERNAME }}
          key: ${{ secrets.SSH_KEY }}
          source: "."
          target: "/path/to/qwqOps"
```

---

## 跨平台注意事项

### 1. 路径分隔符

代码中已使用 Go 的 `filepath` 包，自动处理路径分隔符：
- Windows: `\`
- Linux/Mac: `/`

### 2. Docker Socket

`docker-compose.yml` 中的 Docker socket 配置：
```yaml
# Linux/Mac
- /var/run/docker.sock:/var/run/docker.sock:ro

# Windows Docker Desktop（自动处理）
# Docker Desktop 会自动映射
```

### 3. 文件权限

在 Ubuntu 上确保脚本有执行权限：
```bash
chmod +x deploy.sh
chmod +x rebuild.sh
chmod +x start.sh
```

### 4. 行尾符

如果遇到脚本执行问题，可能是 Windows CRLF vs Linux LF：
```bash
# 在 Ubuntu 上转换
dos2unix deploy.sh
# 或使用 sed
sed -i 's/\r$//' deploy.sh
```

---

## 常见问题

### Q: Windows 上构建失败？
A: 确保使用 Docker Desktop，并且 WSL2 已启用。

### Q: Ubuntu 上端口被占用？
A: 修改 `docker-compose.yml` 中的端口映射，或停止占用端口的服务。

### Q: 权限错误？
A: 确保 Docker 用户有权限访问 Docker socket：
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Q: 网络问题？
A: 检查防火墙设置：
```bash
# Ubuntu 防火墙
sudo ufw allow 8081/tcp
sudo ufw allow 3308/tcp
sudo ufw allow 6380/tcp
```

---

## 生产环境建议

1. **使用反向代理**（Nginx/Caddy）
2. **配置 HTTPS**（Let's Encrypt）
3. **设置自动备份**
4. **监控和日志**（Prometheus + Grafana）
5. **资源限制**（已在 docker-compose.yml 中配置）

---

## 快速命令参考

```bash
# 查看日志
docker compose logs -f qwq

# 重启服务
docker compose restart qwq

# 停止服务
docker compose down

# 更新代码并重新部署
git pull
docker compose build --no-cache
docker compose up -d

# 查看服务状态
docker compose ps

# 进入容器
docker compose exec qwq sh
```

---

## 技术支持

如遇问题，请查看：
- 日志文件: `logs/qwq.log`
- Docker 日志: `docker compose logs`
- GitHub Issues: [项目地址]

