#!/bin/bash

# 提交 Docker 修复和 package-lock.json

echo "📦 准备提交更改..."

# 添加文件
git add frontend/package-lock.json
git add Dockerfile
git add .gitignore
git add DOCKER_FIX.md
git add SETUP_COMPLETE.md

# 显示将要提交的文件
echo ""
echo "📝 将要提交的文件："
git status --short

# 提交
echo ""
echo "💾 提交更改..."
git commit -m "fix: add package-lock.json and update Dockerfile for reproducible builds

- Add frontend/package-lock.json for deterministic dependency installation
- Update Dockerfile to use npm ci instead of npm install
- Add frontend build artifacts to .gitignore
- Fix Docker build error: npm ci requires package-lock.json

This enables faster and more reliable Docker builds with npm ci."

echo ""
echo "✅ 提交完成！"
echo ""
echo "🚀 推送到 GitHub："
echo "   git push"
echo ""
echo "📊 查看 GitHub Actions："
echo "   https://github.com/yourusername/qwq/actions"
