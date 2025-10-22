# GitHub Actions CI/CD 快速启动指南

## 🚀 5 分钟快速配置

### 步骤 1: 配置 GitHub Secrets

1. 进入你的 GitHub 仓库
2. 点击 `Settings` -> `Secrets and variables` -> `Actions`
3. 点击 `New repository secret`
4. 添加以下必需的 Secrets：

```bash
# Docker Hub（必需，用于镜像推送）
DOCKER_USERNAME: 你的 Docker Hub 用户名
DOCKER_PASSWORD: 你的 Docker Hub Token

# 开发环境 API URL（必需）
DEV_API_URL: http://dev.yourdomain.com:8080

# 预发布环境 API URL（必需）
STAGING_API_URL: http://staging.yourdomain.com:8080

# 生产环境 API URL（必需）
PROD_API_URL: https://api.yourdomain.com
```

参考 `.github/workflows/secrets.example` 文件查看完整的配置模板。

---

### 步骤 2: 测试 CI 流程

创建一个测试分支并推送：

```bash
# 创建测试分支
git checkout -b test/ci-setup

# 做一个小改动
echo "# CI/CD Test" >> CICD_TEST.md
git add CICD_TEST.md
git commit -m "test: verify CI/CD setup"

# 推送到远程
git push origin test/ci-setup
```

然后在 GitHub 上创建 Pull Request 到 `develop` 分支，观察 CI 工作流是否运行。

---

### 步骤 3: 测试 Runner 连通性

1. 进入 GitHub Actions 页面
2. 选择 `Ping Runner` 工作流
3. 点击 `Run workflow`
4. 选择 `ubuntu-latest` 或你的 `self-hosted` runner
5. 点击 `Run workflow` 按钮

查看运行结果，确认 Runner 正常工作。

---

### 步骤 4: 配置数据库 Secrets（可选）

如果需要使用数据库操作工作流，添加以下 Secrets：

```bash
# 开发环境数据库
DB_HOST_dev: localhost
DB_PORT_dev: 3306
DB_NAME_dev: iam_dev
DB_USER_dev: root
DB_PASSWORD_dev: your_password

# 重复为 staging 和 prod 环境配置
```

---

### 步骤 5: 测试完整的 CI/CD 流程

#### 开发环境部署测试

```bash
# 切换到 develop 分支
git checkout develop

# 做一个改动
echo "Test deployment" > test.txt
git add test.txt
git commit -m "chore: test dev deployment"

# 推送（会自动触发部署到开发环境）
git push origin develop
```

#### 预发布环境部署测试

```bash
# 创建发布分支
git checkout -b release/v0.1.0 develop

# 推送（会自动触发部署到预发布环境）
git push origin release/v0.1.0
```

#### 生产环境部署测试

```bash
# 确保在 main 分支
git checkout main

# 合并发布分支
git merge --no-ff release/v0.1.0

# 创建版本标签（会自动触发生产部署）
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin main --tags
```

---

## 📊 验证部署

### 检查工作流状态

访问 GitHub Actions 页面查看工作流运行状态：
```
https://github.com/YOUR_ORG/iam-contracts/actions
```

### 使用健康检查工作流

1. 进入 GitHub Actions 页面
2. 选择 `Server Health Check` 工作流
3. 点击 `Run workflow`
4. 选择环境和检查类型
5. 点击 `Run workflow` 按钮

---

## 🔧 故障排查

### CI 工作流失败

**问题：** lint 或 test 失败

**解决方案：**
```bash
# 本地运行检查
make lint
make test

# 修复问题后重新推送
git add .
git commit -m "fix: resolve CI issues"
git push
```

---

### Docker 镜像推送失败

**问题：** 无法推送到 Docker Hub

**解决方案：**
1. 检查 DOCKER_USERNAME 和 DOCKER_PASSWORD 是否正确
2. 确认 Docker Hub Token 有推送权限
3. 验证镜像名称格式正确

---

### 部署失败

**问题：** 部署到服务器失败

**解决方案：**
1. 检查 API_URL Secrets 是否正确
2. 确认服务器可访问
3. 验证 SSH 密钥配置（如使用 self-hosted runner）
4. 查看工作流日志了解详细错误

---

## 📝 最佳实践建议

### 1. 分支策略

```
main (生产)
  ├── release/v1.0.0 (预发布)
  │     └── develop (开发)
  │           ├── feature/login (功能)
  │           ├── feature/auth (功能)
  │           └── bugfix/issue-123 (修复)
  └── hotfix/critical-bug (紧急修复)
```

### 2. 提交消息规范

```bash
feat: 新功能
fix: Bug 修复
docs: 文档更新
style: 代码格式（不影响代码运行）
refactor: 重构
test: 测试相关
chore: 构建过程或辅助工具的变动
perf: 性能优化
ci: CI/CD 配置
```

### 3. 版本号规范

遵循语义化版本 (Semantic Versioning)：

```
v主版本号.次版本号.修订号

例如：
v1.0.0 - 首次正式发布
v1.1.0 - 添加新功能（向后兼容）
v1.1.1 - Bug 修复
v2.0.0 - 破坏性更新
```

### 4. 安全注意事项

- ✅ 使用 GitHub Secrets 存储敏感信息
- ✅ 定期轮换密码和 Token
- ✅ 为不同环境使用不同的凭据
- ✅ 限制 self-hosted runner 的访问权限
- ❌ 不要在代码中硬编码密码
- ❌ 不要提交 .env 文件到 Git

---

## 🎯 下一步

配置完成后，你可以：

1. **配置通知：** 集成 Slack、企业微信等通知
2. **增强安全：** 添加 SAST、依赖扫描等安全检查
3. **性能监控：** 集成 APM 工具（如 Datadog、New Relic）
4. **环境管理：** 使用 GitHub Environments 增强部署控制
5. **自动化更多：** 添加自动化测试、性能测试等

---

## 📚 参考文档

- [GitHub Actions 完整文档](https://docs.github.com/en/actions)
- [工作流详细说明](.github/workflows/README.md)
- [Secrets 配置模板](.github/workflows/secrets.example)

---

## 💬 需要帮助？

如果遇到问题：

1. 查看 [工作流详细文档](.github/workflows/README.md)
2. 检查 GitHub Actions 工作流日志
3. 在仓库中创建 Issue 描述问题

---

**祝你使用愉快！🎉**
