@echo off
REM ============================================
REM 手动拉取 Docker 镜像脚本 (Windows)
REM 解决镜像拉取失败问题
REM ============================================

echo.
echo ========================================
echo   手动拉取 Docker 镜像
echo ========================================
echo.

REM 需要拉取的镜像列表
set IMAGES=node:18-alpine golang:1.23-alpine alpine:3.19 mysql:8.0 redis:7-alpine prom/prometheus:latest grafana/grafana:latest

echo 准备拉取镜像...
echo.

REM 拉取每个镜像
for %%I in (%IMAGES%) do (
    echo [拉取] %%I
    docker pull %%I
    if errorlevel 1 (
        echo [失败] %%I 拉取失败，请检查网络连接
    ) else (
        echo [成功] %%I 拉取成功
    )
    echo.
)

echo.
echo ========================================
echo   镜像拉取完成！
echo ========================================
echo.
echo 已拉取的镜像：
docker images | findstr /I "node golang alpine mysql redis prometheus grafana"
echo.
echo 下一步：
echo   运行部署脚本: start.bat
echo   或直接构建: docker compose build
echo.

pause
