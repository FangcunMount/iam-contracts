# GitHub Actions CI/CD 工作流说明

本项目使用 GitHub Actions 实现完整的 CI/CD 流程。以下是各个工作流的说明和使用方法。

## 📋 工作流列表

### 1. Ping Runner (`ping-runner.yml`)

**用途：** 测试 GitHub Actions Runner 的连通性和可用性

**触发方式：**
- 手动触发（workflow_dispatch）
- 定时任务（每天凌晨 1:00 UTC，北京时间 9:00）

**功能：**
- 显示 Runner 系统信息（OS、架构、CPU、内存）
- 测试网络连通性
- 检查 Docker 可用性
- 检查 Go 环境
- 生成 Runner 健康报告

**手动触发：**
```bash
# 在 GitHub UI 中：
Actions -> Ping Runner -> Run workflow
# 可选择 runner label（默认：self-hosted）
```

---

### 2. Database Operations (`db-ops.yml`)

**用途：** 数据库操作和维护

**触发方式：**
- 手动触发（workflow_dispatch）

**支持的操作：**

#### a) 健康检查 (health-check)
- 检查数据库连接
- 列出数据库表
- 显示数据库大小

#### b) 备份 (backup)
- 创建数据库完整备份
- 压缩备份文件
- 自动清理 30 天前的备份

#### c) 迁移 (migrate)
- 执行数据库迁移脚本
- 更新数据库架构

#### d) 数据填充 (seed)
- 填充测试数据（仅限非生产环境）

**手动触发示例：**
```bash
# 在 GitHub UI 中：
Actions -> Database Operations -> Run workflow
# 选择操作类型：backup/restore/migrate/seed/health-check
# 选择环境：dev/staging/prod
```

**所需 Secrets：**
```
DB_HOST_dev, DB_HOST_staging, DB_HOST_prod
DB_PORT_dev, DB_PORT_staging, DB_PORT_prod
DB_NAME_dev, DB_NAME_staging, DB_NAME_prod
DB_USER_dev, DB_USER_staging, DB_USER_prod
DB_PASSWORD_dev, DB_PASSWORD_staging, DB_PASSWORD_prod
```

---

### 3. Server Health Check (`server-check.yml`)

**用途：** 服务器和 API 健康检查

**触发方式：**
- 手动触发（workflow_dispatch）
- 定时任务（每小时运行一次）

**检查类型：**

#### a) 完整检查 (full)
- API 端点可访问性
- API 版本信息
- API 响应时间
- 数据库连接
- SSL 证书有效期
- 服务器资源使用情况
- 认证流程测试

#### b) 快速检查 (quick)
- API 健康端点
- 基本连通性测试

#### c) API 专项检查 (api-only)
- API 端点详细检查
- 响应时间测量
- 认证流程测试

#### d) 数据库专项检查 (database-only)
- 数据库连接测试
- 数据库性能检查

**功能特性：**
- 自动生成健康报告
- 生产环境失败时自动创建 Issue
- SSL 证书过期预警（30 天内）

**手动触发示例：**
```bash
# 在 GitHub UI 中：
Actions -> Server Health Check -> Run workflow
# 选择环境：dev/staging/prod
# 选择检查类型：full/quick/api-only/database-only
```

**所需 Secrets：**
```
API_URL_dev, API_URL_staging, API_URL_prod
SERVER_HOST_dev, SERVER_HOST_staging
SERVER_USER_dev, SERVER_USER_staging
SSH_PRIVATE_KEY_dev, SSH_PRIVATE_KEY_staging
```

---

### 4. CI/CD Pipeline (`cicd.yml`)

**用途：** 完整的持续集成和持续部署流程

**触发方式：**
- Push 到 `main`、`develop` 或 `release/**` 分支
- Pull Request 到 `main` 或 `develop`
- 创建版本标签 `v*.*.*`
- 手动触发（workflow_dispatch）

**CI 阶段：**

#### 1. 代码检查 (lint)
- 运行 golangci-lint
- 检查代码格式
- 确保代码质量

#### 2. 单元测试 (test)
- 运行所有单元测试
- 生成代码覆盖率报告
- 上传到 Codecov
- 使用 MySQL 和 Redis 服务

#### 3. 构建二进制 (build)
- 构建 Linux amd64 和 arm64 版本
- 嵌入版本信息
- 上传构建产物

#### 4. Docker 镜像构建 (docker-build)
- 构建多架构 Docker 镜像
- 推送到 Docker Hub
- 自动标签管理

**CD 阶段：**

#### 1. 开发环境部署 (deploy-dev)
- **触发条件：** Push 到 `develop` 分支
- 部署到开发服务器
- 验证部署成功

#### 2. 预发布环境部署 (deploy-staging)
- **触发条件：** Push 到 `release/**` 分支
- 使用 Docker Compose 部署
- 运行冒烟测试

#### 3. 生产环境部署 (deploy-prod)
- **触发条件：** 创建版本标签 `v*.*.*`
- 创建备份
- 蓝绿部署策略
- 健康检查
- 失败自动回滚
- 创建 GitHub Release

**部署策略：**

| 环境 | 触发条件 | 部署方式 | 特性 |
|------|---------|---------|------|
| Development | `develop` 分支 | 二进制部署 | 快速迭代 |
| Staging | `release/**` 分支 | Docker Compose | 预发布验证 |
| Production | `v*.*.*` 标签 | 蓝绿部署 | 零停机、自动回滚 |

**手动触发示例：**
```bash
# 在 GitHub UI 中：
Actions -> CI/CD Pipeline -> Run workflow
# 选择部署环境：dev/staging/prod
# 可选：跳过测试
```

**所需 Secrets：**
```
# Docker Hub
DOCKER_USERNAME
DOCKER_PASSWORD

# Codecov (可选)
CODECOV_TOKEN

# 开发环境
DEV_API_URL

# 预发布环境
STAGING_API_URL

# 生产环境
PROD_API_URL
```

---

## 🚀 快速开始

### 1. 配置 GitHub Secrets

在 GitHub 仓库中配置以下 Secrets：

```bash
Settings -> Secrets and variables -> Actions -> New repository secret
```

**必需的 Secrets：**

```yaml
# Docker 凭据
DOCKER_USERNAME: your_dockerhub_username
DOCKER_PASSWORD: your_dockerhub_token

# API URLs
DEV_API_URL: http://dev.example.com
STAGING_API_URL: http://staging.example.com
PROD_API_URL: https://api.example.com

# 数据库配置（针对每个环境）
DB_HOST_dev: localhost
DB_PORT_dev: 3306
DB_NAME_dev: iam_dev
DB_USER_dev: root
DB_PASSWORD_dev: password

# 重复为 staging 和 prod 环境配置
```

### 2. 配置 Self-hosted Runner（可选）

如果需要使用自托管 Runner：

```bash
# 在服务器上
cd ~
mkdir actions-runner && cd actions-runner
curl -o actions-runner-linux-x64-2.311.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.311.0.tar.gz

# 配置 Runner
./config.sh --url https://github.com/YOUR_ORG/iam-contracts --token YOUR_TOKEN

# 作为服务运行
sudo ./svc.sh install
sudo ./svc.sh start
```

### 3. 开发工作流

#### 功能开发
```bash
# 创建功能分支
git checkout -b feature/your-feature develop

# 开发和提交
git add .
git commit -m "feat: add new feature"

# 推送到远程
git push origin feature/your-feature

# 创建 Pull Request 到 develop
# CI 会自动运行测试和代码检查
```

#### 发布流程
```bash
# 从 develop 创建发布分支
git checkout -b release/v1.2.0 develop

# 推送发布分支
git push origin release/v1.2.0
# 自动部署到 staging 环境

# 测试通过后，合并到 main 并打标签
git checkout main
git merge --no-ff release/v1.2.0
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin main --tags
# 自动部署到 production 环境
```

---

## 📊 监控和通知

### 工作流状态

所有工作流的状态可以在以下位置查看：
- GitHub Actions 页面：`https://github.com/YOUR_ORG/iam-contracts/actions`
- 提交状态检查：每个提交旁边会显示检查状态

### 失败通知

- **CI/CD 失败：** 自动创建 Issue
- **生产健康检查失败：** 自动创建 Issue 并添加 `production` 标签
- **邮件通知：** GitHub 会向相关人员发送邮件

### 监控 Badge

在 README 中添加状态徽章：

```markdown
![CI/CD](https://github.com/YOUR_ORG/iam-contracts/actions/workflows/cicd.yml/badge.svg)
![Server Health](https://github.com/YOUR_ORG/iam-contracts/actions/workflows/server-check.yml/badge.svg)
```

---

## 🔧 故障排查

### 常见问题

#### 1. Runner 连接失败
```bash
# 运行 ping-runner 工作流检查 Runner 状态
# 检查 Runner 服务是否运行
sudo systemctl status actions.runner.*
```

#### 2. 部署失败
```bash
# 检查 Secrets 配置是否正确
# 查看工作流日志了解详细错误
# 运行 server-check 工作流验证服务器状态
```

#### 3. 数据库操作失败
```bash
# 运行 db-ops 健康检查
# 验证数据库连接配置
# 检查数据库服务器防火墙规则
```

#### 4. Docker 镜像构建失败
```bash
# 检查 Dockerfile 语法
# 验证 Docker Hub 凭据
# 检查磁盘空间
```

---

## 📝 最佳实践

### 1. 分支管理
- `main` - 生产环境代码
- `develop` - 开发环境代码
- `feature/*` - 功能开发
- `release/*` - 发布准备
- `hotfix/*` - 紧急修复

### 2. 提交规范
使用 Conventional Commits：
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试
chore: 构建/工具链
```

### 3. 版本标签
遵循语义化版本：
```
v1.0.0 - 主版本.次版本.修订号
v1.0.0-beta.1 - 预发布版本
v1.0.0-rc.1 - 候选版本
```

### 4. 安全建议
- 不要在代码中硬编码敏感信息
- 使用 GitHub Secrets 管理凭据
- 定期轮换 Secrets
- 限制 Runner 访问权限
- 审查第三方 Actions

---

## 🔄 工作流维护

### 定期维护任务

- [ ] 每月检查并更新 Actions 版本
- [ ] 审查和清理旧的构建产物
- [ ] 检查 Secret 有效性
- [ ] 优化工作流执行时间
- [ ] 更新文档

### 性能优化

1. **使用缓存：** Go 依赖、Docker 层缓存
2. **并行执行：** 独立的 job 并行运行
3. **矩阵构建：** 多平台并行构建
4. **条件执行：** 跳过不必要的步骤

---

## 📚 参考资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [工作流语法](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [Self-hosted Runner](https://docs.github.com/en/actions/hosting-your-own-runners)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
- [Codecov Action](https://github.com/codecov/codecov-action)

---

## 📞 支持

如有问题，请：
1. 查看工作流日志
2. 搜索现有 Issues
3. 创建新的 Issue 并附上详细信息

---

**最后更新：** 2025-10-21
