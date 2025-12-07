@echo off
REM ============================================
REM qwq AIOps 平台 - 重新构建脚本（Windows）
REM ============================================

echo.
echo ========================================
echo   qwq AIOps 平台重新构建脚本
echo ========================================
echo.

REM 检查 Docker 是否运行
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Docker 未运行，请先启动 Docker Desktop
    pause
    exit /b 1
)

echo [1/5] 检查 Docker 环境... OK
echo.

REM 停止并删除容器
echo [2/5] 停止并删除现有容器...
docker-compose down
echo.

REM 清理 Docker 缓存
echo [3/5] 清理 Docker 缓存...
docker system prune -f
echo.

REM 重新构建（不使用缓存）
echo [4/5] 重新构建镜像（不使用缓存）...
echo [提示] 这可能需要 5-10 分钟，请耐心等待...
echo.
docker-compose build --no-cache --progress=plain

if %errorlevel% neq 0 (
    echo.
    echo [错误] 构建失败
    echo.
    echo 可能的原因：
    echo 1. 网络问题 - 查看 NETWORK_FIX.md
    echo 2. 磁盘空间不足 - 运行 docker system df 检查
    echo 3. 依赖问题 - 查看上面的错误信息
    echo.
    pause
    exit /b 1
)

echo.
echo [5/5] 启动服务...
docker-compose up -d

if %errorlevel% neq 0 (
    echo.
    echo [错误] 启动失败
    pause
    exit /b 1
)

echo.
echo ========================================
echo   重新构建成功！
echo ========================================
echo.
echo 访问地址：
echo   前端界面: http://localhost:8081
echo   API 文档: http://localhost:8081/api/docs
echo.
echo 查看日志：
echo   docker-compose logs -f qwq
echo.
echo ========================================

REM 等待服务启动
echo 等待服务启动...
timeout /t 10 /nobreak >nul

REM 健康检查
echo 正在检查服务状态...
curl -s http://localhost:8081/api/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [成功] 服务运行正常
) else (
    echo [警告] 服务可能还在启动中
    echo 运行: docker-compose logs -f qwq 查看日志
)

echo.
pause
