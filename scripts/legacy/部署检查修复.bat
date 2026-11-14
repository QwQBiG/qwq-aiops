@echo off
REM ============================================
REM qwq AIOps 平台 - 部署检查和修复脚本
REM ============================================

setlocal enabledelayedexpansion

echo.
echo ========================================
echo   qwq AIOps 平台 - 部署检查和修复
echo ========================================
echo.

REM 颜色定义（Windows 10+）
set "GREEN=[92m"
set "YELLOW=[93m"
set "RED=[91m"
set "BLUE=[94m"
set "NC=[0m"

set ERROR_COUNT=0
set WARNING_COUNT=0

REM ============================================
REM 1. 检查 Docker 环境
REM ============================================
echo %BLUE%[1/8] 检查 Docker 环境...%NC%
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo %RED%✗ Docker 未运行或未安装%NC%
    echo   请先安装并启动 Docker Desktop
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ Docker 运行正常%NC%
)
echo.

REM ============================================
REM 2. 检查 Docker Compose
REM ============================================
echo %BLUE%[2/8] 检查 Docker Compose...%NC%
docker compose version >nul 2>&1
if %errorlevel% neq 0 (
    echo %YELLOW%⚠ Docker Compose V2 未找到，尝试 V1...%NC%
    docker-compose --version >nul 2>&1
    if %errorlevel% neq 0 (
        echo %RED%✗ Docker Compose 未安装%NC%
        set /a ERROR_COUNT+=1
    ) else (
        echo %GREEN%✓ Docker Compose V1 可用%NC%
        echo %YELLOW%⚠ 建议升级到 Docker Compose V2%NC%
        set /a WARNING_COUNT+=1
    )
) else (
    echo %GREEN%✓ Docker Compose V2 可用%NC%
)
echo.

REM ============================================
REM 3. 检查必要文件
REM ============================================
echo %BLUE%[3/8] 检查必要文件...%NC%

REM 检查 docker-compose.yml
if not exist docker-compose.yml (
    echo %RED%✗ docker-compose.yml 不存在%NC%
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ docker-compose.yml 存在%NC%
)

REM 检查 Dockerfile
if not exist Dockerfile (
    echo %RED%✗ Dockerfile 不存在%NC%
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ Dockerfile 存在%NC%
)

REM 检查 go.mod
if not exist go.mod (
    echo %RED%✗ go.mod 不存在%NC%
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ go.mod 存在%NC%
)

REM 检查前端目录
if not exist frontend\package.json (
    echo %RED%✗ frontend/package.json 不存在%NC%
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ frontend/package.json 存在%NC%
)

REM 检查后端入口
if not exist cmd\qwq\main.go (
    echo %RED%✗ cmd/qwq/main.go 不存在%NC%
    set /a ERROR_COUNT+=1
) else (
    echo %GREEN%✓ cmd/qwq/main.go 存在%NC%
)

echo.

REM ============================================
REM 4. 检查和创建 .env 文件
REM ============================================
echo %BLUE%[4/8] 检查 .env 配置文件...%NC%

if not exist .env (
    echo %YELLOW%⚠ .env 文件不存在，正在创建...%NC%
    if exist .env.example (
        copy .env.example .env >nul
        echo %GREEN%✓ 已从 .env.example 创建 .env 文件%NC%
        echo %YELLOW%⚠ 请编辑 .env 文件配置 AI 服务%NC%
        set /a WARNING_COUNT+=1
    ) else (
        echo %RED%✗ .env.example 不存在，无法创建 .env%NC%
        set /a ERROR_COUNT+=1
    )
) else (
    echo %GREEN%✓ .env 文件存在%NC%
    
    REM 检查 AI 配置
    findstr /C:"AI_PROVIDER=" .env | findstr /V /C:"#" >nul 2>&1
    if %errorlevel% neq 0 (
        echo %YELLOW%⚠ AI_PROVIDER 未配置%NC%
        set /a WARNING_COUNT+=1
    ) else (
        echo %GREEN%✓ AI_PROVIDER 已配置%NC%
    )
)

echo.

REM ============================================
REM 5. 检查配置目录
REM ============================================
echo %BLUE%[5/8] 检查配置目录...%NC%

if not exist config (
    echo %YELLOW%⚠ config 目录不存在，正在创建...%NC%
    mkdir config
    echo %GREEN%✓ config 目录已创建%NC%
) else (
    echo %GREEN%✓ config 目录存在%NC%
)

REM 检查 prometheus.yml
if not exist config\prometheus.yml (
    echo %YELLOW%⚠ prometheus.yml 不存在，正在创建...%NC%
    (
        echo global:
        echo   scrape_interval: 15s
        echo   evaluation_interval: 15s
        echo.
        echo scrape_configs:
        echo   - job_name: 'prometheus'
        echo     static_configs:
        echo       - targets: ['localhost:9090']
        echo.
        echo   - job_name: 'qwq'
        echo     static_configs:
        echo       - targets: ['qwq:8899']
        echo     metrics_path: '/metrics'
    ) > config\prometheus.yml
    echo %GREEN%✓ prometheus.yml 已创建%NC%
) else (
    echo %GREEN%✓ prometheus.yml 存在%NC%
)

REM 检查 mysql.cnf
if not exist config\mysql.cnf (
    echo %YELLOW%⚠ mysql.cnf 不存在，正在创建...%NC%
    (
        echo [mysqld]
        echo character-set-server=utf8mb4
        echo collation-server=utf8mb4_unicode_ci
        echo default-authentication-plugin=mysql_native_password
        echo.
        echo [client]
        echo default-character-set=utf8mb4
    ) > config\mysql.cnf
    echo %GREEN%✓ mysql.cnf 已创建%NC%
) else (
    echo %GREEN%✓ mysql.cnf 存在%NC%
)

echo.

REM ============================================
REM 6. 检查数据目录
REM ============================================
echo %BLUE%[6/8] 检查数据目录...%NC%

for %%d in (data logs backups) do (
    if not exist %%d (
        echo %YELLOW%⚠ %%d 目录不存在，正在创建...%NC%
        mkdir %%d
        echo %GREEN%✓ %%d 目录已创建%NC%
    ) else (
        echo %GREEN%✓ %%d 目录存在%NC%
    )
)

echo.

REM ============================================
REM 7. 检查端口占用
REM ============================================
echo %BLUE%[7/8] 检查端口占用...%NC%

set PORTS=8081 3308 6380 9091 3000

for %%p in (%PORTS%) do (
    netstat -ano | findstr ":%%p " | findstr "LISTENING" >nul 2>&1
    if %errorlevel% equ 0 (
        echo %YELLOW%⚠ 端口 %%p 已被占用%NC%
        set /a WARNING_COUNT+=1
    ) else (
        echo %GREEN%✓ 端口 %%p 可用%NC%
    )
)

echo.

REM ============================================
REM 8. 检查网络连接
REM ============================================
echo %BLUE%[8/8] 检查网络连接...%NC%

REM 测试 Docker Hub
curl -s -I --connect-timeout 5 https://hub.docker.com >nul 2>&1
if %errorlevel% equ 0 (
    echo %GREEN%✓ Docker Hub 可访问%NC%
) else (
    echo %YELLOW%⚠ Docker Hub 访问失败%NC%
    set /a WARNING_COUNT+=1
)

REM 测试 Go 代理
curl -s -I --connect-timeout 5 https://goproxy.cn >nul 2>&1
if %errorlevel% equ 0 (
    echo %GREEN%✓ Go 代理可访问%NC%
) else (
    echo %YELLOW%⚠ Go 代理访问失败%NC%
    set /a WARNING_COUNT+=1
)

REM 测试 npm 镜像
curl -s -I --connect-timeout 5 https://registry.npmmirror.com >nul 2>&1
if %errorlevel% equ 0 (
    echo %GREEN%✓ npm 镜像可访问%NC%
) else (
    echo %YELLOW%⚠ npm 镜像访问失败%NC%
    set /a WARNING_COUNT+=1
)

echo.

REM ============================================
REM 检查结果汇总
REM ============================================
echo ========================================
echo   检查结果汇总
echo ========================================
echo.

if %ERROR_COUNT% equ 0 (
    if %WARNING_COUNT% equ 0 (
        echo %GREEN%✓ 所有检查通过，可以开始部署！%NC%
        echo.
        echo 运行以下命令开始部署：
        echo   start.bat
    ) else (
        echo %YELLOW%⚠ 发现 %WARNING_COUNT% 个警告%NC%
        echo.
        echo 可以继续部署，但建议先处理警告项
        echo.
        echo 运行以下命令开始部署：
        echo   start.bat
    )
) else (
    echo %RED%✗ 发现 %ERROR_COUNT% 个错误，%WARNING_COUNT% 个警告%NC%
    echo.
    echo 请先修复上述错误后再部署
)

echo.
echo ========================================

pause
