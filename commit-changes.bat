@echo off
chcp 65001 >nul
echo.
echo 📦 准备提交更改...
echo.

REM 添加文件
git add frontend/package-lock.json
git add Dockerfile
git add .gitignore
git add go.mod
git add DOCKER_FIX.md
git add SETUP_COMPLETE.md

REM 显示将要提交的文件
echo 📝 将要提交的文件：
git status --short

REM 提交
echo.
echo 💾 提交更改...
git commit -m "fix: add package-lock.json and fix Go version for Docker builds" -m "- Add frontend/package-lock.json for deterministic dependency installation" -m "- Update Dockerfile to use npm ci instead of npm install" -m "- Fix go.mod: downgrade from Go 1.24.0 to Go 1.23 (1.24 not released yet)" -m "- Add frontend build artifacts to .gitignore" -m "- Fix Docker build errors" -m "" -m "This enables faster and more reliable Docker builds."

echo.
echo ✅ 提交完成！
echo.
echo 🚀 推送到 GitHub：
echo    git push
echo.
echo 📊 查看 GitHub Actions：
echo    https://github.com/yourusername/qwq/actions
echo.
pause
