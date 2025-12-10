@echo off
chcp 65001 >nul
echo ============================================
echo qwq AIOps 平台 - 配置诊断工具
echo ============================================
echo.

REM 检查 .env 文件
if not exist ".env" (
    echo [警告] .env 配置文件不存在
    echo.
    echo 是否要从 .env.example 创建 .env 文件？
    set /p CREATE_ENV="输入 Y 创建，其他键跳过: "
    if /i "%CREATE_ENV%"=="Y" (
        copy .env.example .env
        echo [成功] .env 文件已创建，请编辑配置
    )
) else (
    echo [成功] .env 配置文件存在
)
echo.

REM 检查必需的配置项
echo 检查必需的配置项...
echo.

REM 检查 JWT_SECRET
findstr /C:"JWT_SECRET=" .env >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] JWT_SECRET 未配置
) else (
    echo [成功] JWT_SECRET 已配置
)

REM 检查 ENCRYPTION_KEY
findstr /C:"ENCRYPTION_KEY=" .env >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] ENCRYPTION_KEY 未配置
) else (
    echo [成功] ENCRYPTION_KEY 已配置
)

REM 检查钉钉配置
findstr /C:"DINGTALK_WEBHOOK=" .env >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] DINGTALK_WEBHOOK 未配置，钉钉通知功能不可用
) else (
    echo [成功] DINGTALK_WEBHOOK 已配置
)

REM 检查 AI 配置
findstr /C:"AI_PROVIDER=" .env >nul 2>&1
if %errorlevel% neq 0 (
    echo [提示] AI_PROVIDER 未配置，AI 功能不可用
) else (
    echo [成功] AI_PROVIDER 已配置
)

echo.
echo ============================================
echo 诊断完成
echo ============================================
echo.
echo 如需更详细的诊断，请运行: go run cmd/qwq/main.go --diagnose
echo.
pause
