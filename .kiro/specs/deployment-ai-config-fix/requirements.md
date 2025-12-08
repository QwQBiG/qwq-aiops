# 部署和 AI 配置修复需求文档

## 简介

qwq AIOps 平台的部署流程存在 AI 配置不一致的问题,导致用户无法正常部署。本需求旨在修复部署脚本、配置文件和文档,确保用户能够顺利完成部署。

## 术语表

- **qwq**: qwq AIOps 智能运维平台
- **AI Provider**: AI 服务提供商,如 OpenAI、Ollama
- **docker-compose.yml**: Docker Compose 配置文件
- **一键部署脚本**: 自动化部署脚本 (一键部署.sh)
- **配置脚本**: AI 服务配置脚本 (配置AI服务.sh)
- **环境变量**: Docker 容器的环境变量配置

## 需求

### 需求 1: docker-compose.yml 配置规范化

**用户故事**: 作为部署人员,我希望 docker-compose.yml 中的 AI 配置清晰明确,以便我能够正确配置 AI 服务。

#### 验收标准

1. WHEN docker-compose.yml 被创建 THEN 系统应当将所有 AI 相关环境变量设置为注释状态,并提供清晰的配置示例
2. WHEN 用户查看 docker-compose.yml THEN 系统应当在注释中明确说明 OpenAI 和 Ollama 两种配置方式的区别
3. WHEN docker-compose.yml 包含 AI 配置 THEN 系统应当使用统一的注释格式,便于脚本进行替换操作
4. WHEN 用户需要配置 OpenAI THEN docker-compose.yml 应当包含 AI_PROVIDER、OPENAI_API_KEY、OPENAI_BASE_URL 三个环境变量的配置示例
5. WHEN 用户需要配置 Ollama THEN docker-compose.yml 应当包含 AI_PROVIDER、OLLAMA_HOST、OLLAMA_MODEL 三个环境变量的配置示例

### 需求 2: 一键部署脚本修复

**用户故事**: 作为部署人员,我希望一键部署脚本能够正确更新 docker-compose.yml 中的 AI 配置,以便我选择的 AI 服务能够生效。

#### 验收标准

1. WHEN 用户选择 OpenAI 并输入 API Key THEN 脚本应当正确取消 AI_PROVIDER 和 OPENAI_API_KEY 的注释,并填入用户提供的值
2. WHEN 用户选择 Ollama 并输入服务地址 THEN 脚本应当正确取消 AI_PROVIDER 和 OLLAMA_HOST 的注释,并填入用户提供的值
3. WHEN 用户选择跳过配置 THEN 脚本应当保持所有 AI 配置为注释状态,并在启动后给出明确的配置提示
4. WHEN 脚本更新配置文件 THEN 系统应当确保只修改 AI 相关的环境变量,不影响其他配置
5. WHEN 脚本执行完成 THEN 系统应当验证 docker-compose.yml 的语法正确性

### 需求 3: 配置AI服务脚本修复

**用户故事**: 作为运维人员,我希望配置AI服务.sh 脚本能够正确更新已部署系统的 AI 配置,以便我可以随时切换 AI 服务。

#### 验收标准

1. WHEN 用户运行配置脚本并选择 OpenAI THEN 脚本应当正确更新 docker-compose.yml 中的 OpenAI 配置,并注释掉 Ollama 配置
2. WHEN 用户运行配置脚本并选择 Ollama THEN 脚本应当正确更新 docker-compose.yml 中的 Ollama 配置,并注释掉 OpenAI 配置
3. WHEN 用户运行配置脚本并选择自定义 API THEN 脚本应当正确添加 OPENAI_BASE_URL 环境变量
4. WHEN 脚本更新配置后 THEN 系统应当提示用户重启服务以使配置生效
5. WHEN 脚本测试 Ollama 连接失败 THEN 系统应当给出明确的错误提示和解决方案

### 需求 4: 部署文档统一和简化

**用户故事**: 作为新用户,我希望部署文档清晰简洁,以便我能够快速理解如何部署和配置 AI 服务。

#### 验收标准

1. WHEN 用户查看快速开始文档 THEN 系统应当在文档开头明确说明 AI 配置是必需的
2. WHEN 用户查看部署指南 THEN 系统应当提供 OpenAI 和 Ollama 两种配置方式的对比表格
3. WHEN 用户遇到 AI 配置问题 THEN 文档应当提供常见问题和解决方案
4. WHEN 用户需要切换 AI 服务 THEN 文档应当提供明确的步骤说明
5. WHEN 用户查看 AI配置说明.md THEN 系统应当提供完整的配置示例和测试方法

### 需求 5: 配置验证和错误提示

**用户故事**: 作为部署人员,我希望系统能够验证 AI 配置的正确性,以便我能够及时发现和修复配置错误。

#### 验收标准

1. WHEN 服务启动时未配置 AI THEN 系统应当在日志中输出明确的错误信息和配置指引
2. WHEN OpenAI API Key 无效 THEN 系统应当在启动时检测并给出明确的错误提示
3. WHEN Ollama 服务无法连接 THEN 系统应当在启动时检测并给出明确的错误提示
4. WHEN 用户访问健康检查接口 THEN 系统应当返回 AI 服务的连接状态
5. WHEN AI 配置正确 THEN 系统应当在日志中输出 AI 服务类型和模型信息

### 需求 6: 端口配置优化

**用户故事**: 作为部署人员,我希望系统使用不常用的端口,以便避免与其他服务发生端口冲突。

#### 验收标准

1. WHEN docker-compose.yml 配置服务端口 THEN 系统应当使用不常用的端口号,避免与常见服务冲突
2. WHEN 用户部署 qwq 主服务 THEN 系统应当使用 8081 端口而非 8080 端口
3. WHEN 用户部署 MySQL 服务 THEN 系统应当使用 3307 端口而非 3306 端口
4. WHEN 用户部署 Redis 服务 THEN 系统应当使用 6380 端口而非 6379 端口
5. WHEN 用户部署 Prometheus 服务 THEN 系统应当使用 9091 端口而非 9090 端口
6. WHEN 端口被占用 THEN 部署文档应当提供清晰的端口修改指引
7. WHEN 用户查看服务访问地址 THEN 所有文档应当使用统一的端口号(8081)
