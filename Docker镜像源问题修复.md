# Docker 镜像源问题修复

## 问题

无法从 Docker Hub 拉取镜像：
```
failed to resolve source metadata for docker.io/library/node:18-alpine: notfound
```

## 快速解决方案

### Linux 用户（推荐）⭐

运行自动配置脚本：

```bash
chmod +x fix-docker-mirror.sh
sudo ./fix-docker-mirror.sh
```

然后重新构建：

```bash
docker-compose build --no-cache
docker-compose up -d
```

### 手动配置

#### 1. 编辑 Docker 配置

```bash
sudo nano /etc/docker/daemon.json
```

#### 2. 添加以下内容

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

#### 3. 重启 Docker

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

#### 4. 验证配置

```bash
docker info | grep -A 5 "Registry Mirrors"
```

#### 5. 重新构建

```bash
docker-compose build --no-cache
docker-compose up -d
```

## macOS / Windows 用户

### Docker Desktop 配置

1. 打开 Docker Desktop
2. 进入 Settings → Docker Engine
3. 添加配置：

```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com"
  ]
}
```

4. 点击 "Apply & Restart"
5. 重新构建：

```bash
docker-compose build --no-cache
docker-compose up -d
```

## 可用的国内镜像源

| 镜像源 | 地址 | 速度 |
|--------|------|------|
| 中科大 | https://docker.mirrors.ustc.edu.cn | ⭐⭐⭐⭐⭐ |
| 网易 | https://hub-mirror.c.163.com | ⭐⭐⭐⭐ |
| 百度云 | https://mirror.baidubce.com | ⭐⭐⭐⭐ |
| 腾讯云 | https://ccr.ccs.tencentyun.com | ⭐⭐⭐⭐ |

## 验证修复

配置完成后，测试拉取镜像：

```bash
# 测试拉取
docker pull node:18-alpine

# 应该看到从镜像源下载
# Pulling from docker.mirrors.ustc.edu.cn/library/node
```

## 如果还是失败

### 方法 1：手动拉取镜像

```bash
# 手动拉取所需镜像
docker pull node:18-alpine
docker pull golang:1.23-alpine
docker pull alpine:3.19

# 然后构建
docker-compose build
```

### 方法 2：使用代理

```bash
# 设置代理（如果有）
export HTTP_PROXY=http://proxy:8080
export HTTPS_PROXY=http://proxy:8080

# 构建
docker-compose build
```

### 方法 3：检查网络

```bash
# 测试网络连接
ping docker.io
curl -I https://registry-1.docker.io/v2/

# 测试镜像源
curl -I https://docker.mirrors.ustc.edu.cn/v2/
```

## 创建的文件

1. ✅ `fix-docker-mirror.sh` - 自动配置脚本
2. ✅ `docker-daemon-config.md` - 详细配置指南
3. ✅ `Docker镜像源问题修复.md` - 本文档

## 相关文档

- **[docker-daemon-config.md](docker-daemon-config.md)** - 详细配置指南
- **[NETWORK_FIX.md](NETWORK_FIX.md)** - 网络问题修复
- **[START_HERE.md](START_HERE.md)** - 快速开始

---

**修复时间**: 2025-12-07  
**状态**: ✅ 已提供解决方案  
**下一步**: 配置镜像源后重新构建
