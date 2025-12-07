# Docker 镜像源配置指南

## 问题

无法从 Docker Hub 拉取镜像：
```
failed to resolve source metadata for docker.io/library/node:18-alpine: notfound
```

## 解决方案：配置国内镜像源

### Linux 系统

1. **创建或编辑 Docker 配置文件**：

```bash
sudo mkdir -p /etc/docker
sudo nano /etc/docker/daemon.json
```

2. **添加以下内容**：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://ccr.ccs.tencentyun.com"
  ],
  "dns": ["8.8.8.8", "8.8.4.4"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

3. **重启 Docker 服务**：

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

4. **验证配置**：

```bash
docker info | grep -A 10 "Registry Mirrors"
```

### macOS / Windows (Docker Desktop)

1. **打开 Docker Desktop**

2. **进入设置**：
   - macOS: Docker Desktop → Preferences → Docker Engine
   - Windows: Docker Desktop → Settings → Docker Engine

3. **添加镜像源配置**：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

4. **点击 "Apply & Restart"**

## 可用的国内镜像源

| 镜像源 | 地址 | 提供商 |
|--------|------|--------|
| 中科大 | https://docker.mirrors.ustc.edu.cn | 中国科学技术大学 |
| 网易 | https://hub-mirror.c.163.com | 网易 |
| 百度云 | https://mirror.baidubce.com | 百度 |
| 腾讯云 | https://ccr.ccs.tencentyun.com | 腾讯 |
| 阿里云 | https://[your-id].mirror.aliyuncs.com | 阿里（需注册） |

## 验证配置

配置完成后，测试拉取镜像：

```bash
# 测试拉取镜像
docker pull node:18-alpine

# 查看镜像
docker images | grep node
```

## 重新构建

配置镜像源后，重新构建：

```bash
# 清理缓存
docker system prune -f

# 重新构建
docker-compose build --no-cache

# 启动服务
docker-compose up -d
```

## 如果还是失败

### 方法 1：使用代理

如果您有 HTTP 代理：

```bash
# 临时设置代理
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080

# 构建
docker-compose build
```

### 方法 2：手动下载镜像

```bash
# 手动拉取所需的基础镜像
docker pull node:18-alpine
docker pull golang:1.23-alpine
docker pull alpine:3.19

# 然后构建
docker-compose build
```

### 方法 3：使用已有镜像

如果之前构建成功过，直接启动：

```bash
# 不重新构建，使用已有镜像
docker-compose up -d
```

## 网络诊断

### 检查 DNS

```bash
# 测试 DNS 解析
nslookup docker.io
nslookup registry-1.docker.io

# 或使用 dig
dig docker.io
```

### 检查网络连接

```bash
# 测试连接 Docker Hub
curl -I https://registry-1.docker.io/v2/

# 测试连接国内镜像源
curl -I https://docker.mirrors.ustc.edu.cn/v2/
```

### 检查防火墙

```bash
# 检查防火墙规则
sudo iptables -L

# 临时关闭防火墙测试（不推荐生产环境）
sudo systemctl stop firewalld  # CentOS/RHEL
sudo ufw disable  # Ubuntu
```

## 完整配置示例

### /etc/docker/daemon.json

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://ccr.ccs.tencentyun.com"
  ],
  "insecure-registries": [],
  "debug": false,
  "experimental": false,
  "features": {
    "buildkit": true
  },
  "builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "20GB"
    }
  },
  "dns": ["8.8.8.8", "8.8.4.4"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ]
}
```

## 阿里云镜像加速器（推荐）

阿里云提供个人专属的镜像加速器：

1. **注册阿里云账号**：https://www.aliyun.com

2. **获取加速器地址**：
   - 访问：https://cr.console.aliyun.com/cn-hangzhou/instances/mirrors
   - 复制您的专属加速器地址

3. **配置**：

```json
{
  "registry-mirrors": [
    "https://[your-id].mirror.aliyuncs.com"
  ]
}
```

## 故障排查步骤

1. **检查 Docker 服务状态**：
   ```bash
   sudo systemctl status docker
   ```

2. **查看 Docker 日志**：
   ```bash
   sudo journalctl -u docker -f
   ```

3. **测试网络连接**：
   ```bash
   ping docker.io
   curl -I https://registry-1.docker.io/v2/
   ```

4. **检查 Docker 配置**：
   ```bash
   docker info
   ```

5. **重启 Docker**：
   ```bash
   sudo systemctl restart docker
   ```

---

**配置完成后，重新运行**：
```bash
docker-compose build --no-cache
docker-compose up -d
```
