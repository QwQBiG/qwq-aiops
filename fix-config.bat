@echo off
REM ============================================
REM qwq AIOps 平台 - 配置文件修复脚本（Windows）
REM ============================================

echo.
echo ========================================
echo   qwq AIOps 配置文件修复脚本
echo ========================================
echo.

REM 创建 config 目录
if not exist config (
    echo [1/3] 创建 config 目录...
    mkdir config
    echo √ config 目录创建成功
) else (
    echo [1/3] config 目录已存在
)

echo.

REM 创建 prometheus.yml
if not exist config\prometheus.yml (
    echo [2/3] 创建 prometheus.yml...
    (
        echo # Prometheus 配置文件
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
    echo √ prometheus.yml 创建成功
) else (
    echo [2/3] prometheus.yml 已存在
)

echo.

REM 创建 mysql.cnf
if not exist config\mysql.cnf (
    echo [3/3] 创建 mysql.cnf...
    (
        echo [mysqld]
        echo character-set-server=utf8mb4
        echo collation-server=utf8mb4_unicode_ci
        echo default-authentication-plugin=mysql_native_password
        echo.
        echo [client]
        echo default-character-set=utf8mb4
    ) > config\mysql.cnf
    echo √ mysql.cnf 创建成功
) else (
    echo [3/3] mysql.cnf 已存在
)

echo.
echo ========================================
echo   配置文件修复完成！
echo ========================================
echo.
echo 现在可以启动服务：
echo   docker-compose up -d
echo.
pause
