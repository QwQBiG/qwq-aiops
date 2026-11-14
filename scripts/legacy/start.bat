@echo off
REM ============================================
REM qwq AIOps 平台 - Windows 启动脚本
REM ============================================

echo.
echo ========================================
echo   qwq AIOps 平台启动脚本
echo ========================================
echo.

REM 检查 Docker 是否运行
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] Docker 未运行，请先启动 Docker Desktop
    pause
    exit /b 1
)

echo [1/4] 检查 Docker 环境... OK
echo.

REM 检查是否存在 .env 文件
if not exist .env (
    echo [提示] 未找到 .env 文件，正在创建...
    copy .env.example .env
    echo [提示] 请编辑 .env 文件配置 AI API Key
    echo.
)

echo [2/4] 检查配置文件... OK
echo.

REM 测试网络连接
echo [提示] 测试网络连接...
curl -s -I https://goproxy.cn >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 无法访问 goproxy.cn，构建可能较慢
    echo [提示] 如果构建失败，请查看 NETWORK_FIX.md
) else (
    echo [提示] 网络连接正常
)
echo.

REM 停止现有容器
echo [3/4] 停止现有容器...
docker-compose down
echo.

REM 构建并启动
echo [4/4] 构建并启动服务（首次运行需要 5-10 分钟）...
echo.
docker-compose up -d --build

if %errorlevel% neq 0 (
    echo.
    echo [错误] 启动失败，请查看错误信息
    pause
    exit /b 1
)

echo.
echo ========================================
echo   启动成功！
echo ========================================
echo.
echo 访问地址：
echo   前端界面: http://localhost:8081
echo   API 文档: http://localhost:8081/api/docs
echo   健康检查: http://localhost:8081/api/health
echo.
echo 默认账号：
echo   用户名: admin
echo   密码: admin123
echo.
echo 查看日志：
echo   docker-compose logs -f qwq
echo.
echo 停止服务：
echo   docker-compose down
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
    echo.
    echo 是否在浏览器中打开？(Y/N)
    set /p open="请选择: "
    if /i "%open%"=="Y" (
        start http://localhost:8081
    )
) else (
    echo [警告] 服务可能还在启动中，请稍后访问
    echo 或运行: docker-compose logs -f qwq 查看日志
)

echo.
pause
