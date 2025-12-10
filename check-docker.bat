@echo off
echo 正在检查 Docker 状态...
echo.

docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Docker 未运行
    echo.
    echo 请执行以下操作：
    echo 1. 启动 Docker Desktop
    echo 2. 等待 Docker 完全启动（系统托盘图标变绿）
    echo 3. 重新运行此脚本
    pause
    exit /b 1
)

echo [成功] Docker 运行正常
echo.

echo Docker 版本信息：
docker version
echo.

echo 前端构建状态：
if exist "frontend\dist\index.html" (
    echo [成功] 前端已构建
    dir /b frontend\dist\assets\*plugin*.js 2>nul
) else (
    echo [警告] 前端未构建
    echo 运行: cd frontend ^&^& npm run build
)
echo.

pause
