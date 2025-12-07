@echo off
REM ============================================
REM 测试 Docker 镜像拉取 (Windows)
REM 快速验证网络和镜像源配置
REM ============================================

echo.
echo ========================================
echo   测试 Docker 镜像拉取
echo ========================================
echo.

REM 测试镜像
set TEST_IMAGE=alpine:latest

echo [1/3] 检查 Docker 服务状态...
docker info >nul 2>&1
if errorlevel 1 (
    echo [失败] Docker 服务未运行
    echo 请先启动 Docker Desktop
    pause
    exit /b 1
)
echo [成功] Docker 服务正常
echo.

echo [2/3] 测试网络连接...
echo   测试 Docker Hub...
curl -s --connect-timeout 5 https://registry-1.docker.io/v2/ >nul 2>&1
if errorlevel 1 (
    echo   [警告] Docker Hub 无法访问
) else (
    echo   [成功] Docker Hub 可访问
)

echo   测试 USTC 镜像源...
curl -s --connect-timeout 5 https://docker.mirrors.ustc.edu.cn/v2/ >nul 2>&1
if errorlevel 1 (
    echo   [警告] USTC 镜像源无法访问
) else (
    echo   [成功] USTC 镜像源可访问
)
echo.

echo [3/3] 测试拉取镜像...
echo   尝试拉取: %TEST_IMAGE%
docker pull %TEST_IMAGE% >nul 2>&1
if errorlevel 1 (
    echo [失败] 镜像拉取失败
    echo.
    echo ========================================
    echo   测试失败
    echo ========================================
    echo.
    echo 建议的解决方案：
    echo.
    echo 1. 配置镜像源：
    echo    - 打开 Docker Desktop
    echo    - Settings -^> Docker Engine
    echo    - 添加镜像源配置
    echo.
    echo 2. 手动拉取镜像：
    echo    手动拉取镜像.bat
    echo.
    echo 3. 查看详细文档：
    echo    镜像拉取失败解决方案.md
    echo.
    pause
    exit /b 1
) else (
    echo [成功] 镜像拉取成功
    
    REM 清理测试镜像
    docker rmi %TEST_IMAGE% >nul 2>&1
    
    echo.
    echo ========================================
    echo   测试通过！
    echo ========================================
    echo.
    echo 您的 Docker 配置正常，可以开始部署：
    echo   start.bat
    echo.
)

pause
