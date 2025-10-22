# ✅ GitHub Actions CI/CD 实施检查清单

## 📋 文件创建验证

### ✅ 工作流文件（4个）

- [x] `cicd.yml` (15 KB) - 主 CI/CD 流程
- [x] `server-check.yml` (10 KB) - 服务器健康检查
- [x] `db-ops.yml` (7.9 KB) - 数据库操作
- [x] `ping-runner.yml` (3.9 KB) - Runner 连通性测试

### ✅ 文档文件（7个）

- [x] `README.md` (9.6 KB) - 完整使用文档
- [x] `QUICKSTART.md` (5.5 KB) - 快速启动指南
- [x] `MANIFEST.md` (7.0 KB) - 文件清单
- [x] `DIAGRAMS.md` (19 KB) - 流程图和架构图
- [x] `BADGES.md` (4.2 KB) - 状态徽章配置
- [x] `SUMMARY.md` (9.8 KB) - 实施总结
- [x] `secrets.example` (3.0 KB) - Secrets 配置模板

**文件总数：** 11 个  
**总大小：** ~94 KB

---

## 🔍 实施前检查清单

### 1. GitHub 仓库配置

- [ ] 确认仓库已创建
- [ ] 确认有管理员权限
- [ ] 确认 Actions 已启用（Settings → Actions → General）

### 2. Secrets 配置（必需）

#### Docker Hub（必需，用于镜像推送）
- [ ] `DOCKER_USERNAME` - Docker Hub 用户名
- [ ] `DOCKER_PASSWORD` - Docker Hub Token

#### API URLs（必需，至少配置开发环境）
- [ ] `DEV_API_URL` - 开发环境 API 地址
- [ ] `STAGING_API_URL` - 预发布环境 API 地址（可选）
- [ ] `PROD_API_URL` - 生产环境 API 地址（可选）

#### 数据库配置（可选，用于 db-ops 工作流）
- [ ] `DB_HOST_dev` - 开发环境数据库主机
- [ ] `DB_PORT_dev` - 开发环境数据库端口
- [ ] `DB_NAME_dev` - 开发环境数据库名称
- [ ] `DB_USER_dev` - 开发环境数据库用户
- [ ] `DB_PASSWORD_dev` - 开发环境数据库密码

#### SSH 配置（可选，用于服务器健康检查）
- [ ] `SERVER_HOST_dev` - 开发服务器主机
- [ ] `SERVER_USER_dev` - SSH 用户名
- [ ] `SSH_PRIVATE_KEY_dev` - SSH 私钥

### 3. Runner 配置

- [ ] 决定使用 GitHub-hosted 还是 self-hosted runner
- [ ] 如使用 self-hosted，已安装并配置 runner
- [ ] Runner 有必要的权限（Docker、文件系统）

### 4. 分支策略

- [ ] `main` 分支已创建（生产）
- [ ] `develop` 分支已创建（开发）
- [ ] 了解 `release/**` 分支用途（预发布）
- [ ] 了解版本标签 `v*.*.*` 用途（生产部署）

---

## 🧪 实施后测试清单

### 第一步：测试 Runner（预计 2 分钟）

- [ ] 进入 Actions 页面
- [ ] 选择 "Ping Runner" 工作流
- [ ] 点击 "Run workflow"
- [ ] 等待运行完成
- [ ] 检查是否显示绿色✅
- [ ] 查看日志确认系统信息正确

### 第二步：测试 CI（预计 10 分钟）

- [ ] 创建测试分支：`git checkout -b test/ci-setup`
- [ ] 添加测试文件：`echo "test" > test.txt`
- [ ] 提交更改：`git add . && git commit -m "test: CI setup"`
- [ ] 推送到远程：`git push origin test/ci-setup`
- [ ] 在 GitHub 创建 PR 到 `develop`
- [ ] 观察 CI 工作流自动运行
- [ ] 确认 Lint、Test、Build 步骤都通过
- [ ] 检查是否生成了构建产物

### 第三步：测试部署（预计 15 分钟）

#### 测试开发环境部署
- [ ] 合并测试 PR 到 `develop` 分支
- [ ] 推送 `develop` 分支：`git push origin develop`
- [ ] 观察部署工作流启动
- [ ] 确认部署到开发环境成功
- [ ] 访问 `DEV_API_URL/health` 验证服务运行

#### 测试预发布环境部署（可选）
- [ ] 创建发布分支：`git checkout -b release/v0.1.0 develop`
- [ ] 推送发布分支：`git push origin release/v0.1.0`
- [ ] 观察部署到预发布环境
- [ ] 访问 `STAGING_API_URL/health` 验证

#### 测试生产环境部署（谨慎！）
- [ ] 确认准备好部署到生产
- [ ] 切换到 `main` 分支
- [ ] 合并发布分支：`git merge --no-ff release/v0.1.0`
- [ ] 创建标签：`git tag -a v0.1.0 -m "Release v0.1.0"`
- [ ] 推送标签：`git push origin main --tags`
- [ ] 观察生产部署流程（蓝绿部署）
- [ ] 访问 `PROD_API_URL/health` 验证
- [ ] 确认 GitHub Release 已创建

### 第四步：测试健康检查（预计 5 分钟）

- [ ] 进入 Actions 页面
- [ ] 选择 "Server Health Check" 工作流
- [ ] 点击 "Run workflow"
- [ ] 选择环境：`dev`
- [ ] 选择检查类型：`quick`
- [ ] 运行并查看结果
- [ ] 确认所有检查项通过

### 第五步：测试数据库操作（可选，5 分钟）

- [ ] 进入 Actions 页面
- [ ] 选择 "Database Operations" 工作流
- [ ] 点击 "Run workflow"
- [ ] 选择操作：`health-check`
- [ ] 选择环境：`dev`
- [ ] 运行并查看结果
- [ ] 确认数据库连接正常

---

## 📊 功能验证矩阵

| 工作流 | 触发方式 | 测试状态 | 备注 |
|--------|---------|---------|------|
| Ping Runner | 手动 | ⬜ 待测试 | 测试 Runner 连通性 |
| CI/CD - Lint | PR | ⬜ 待测试 | 代码检查 |
| CI/CD - Test | PR | ⬜ 待测试 | 单元测试 |
| CI/CD - Build | PR | ⬜ 待测试 | 构建二进制 |
| CI/CD - Docker | Push | ⬜ 待测试 | Docker 镜像 |
| Deploy Dev | Push develop | ⬜ 待测试 | 开发环境部署 |
| Deploy Staging | Push release | ⬜ 待测试 | 预发布部署 |
| Deploy Prod | Tag v*.*.* | ⬜ 待测试 | 生产部署 |
| Server Check | 手动/定时 | ⬜ 待测试 | 健康检查 |
| DB Operations | 手动 | ⬜ 待测试 | 数据库操作 |

**状态说明：**
- ⬜ 待测试
- ✅ 已通过
- ❌ 失败
- ⚠️ 部分通过

---

## 🔧 故障排查检查清单

### CI 工作流失败

#### Lint 失败
- [ ] 检查代码格式：运行 `make lint` 或 `gofmt -l .`
- [ ] 修复格式问题：`gofmt -w .`
- [ ] 重新提交并推送

#### Test 失败
- [ ] 本地运行测试：`make test` 或 `go test ./...`
- [ ] 检查测试依赖（MySQL、Redis）
- [ ] 修复测试代码
- [ ] 重新运行

#### Build 失败
- [ ] 检查依赖：`go mod verify`
- [ ] 本地构建：`go build ./cmd/apiserver`
- [ ] 检查 Go 版本是否匹配（1.24）
- [ ] 修复编译错误

### Docker 构建失败

- [ ] 检查 Dockerfile 语法
- [ ] 验证 Docker Hub 凭据
- [ ] 检查镜像名称格式
- [ ] 查看详细错误日志

### 部署失败

- [ ] 验证 Secrets 配置正确
- [ ] 检查服务器可访问性
- [ ] 验证部署路径权限
- [ ] 检查服务配置文件
- [ ] 查看部署日志

### 健康检查失败

- [ ] 确认服务正在运行
- [ ] 测试 API 端点可访问
- [ ] 检查数据库连接
- [ ] 验证防火墙规则
- [ ] 检查 SSL 证书

---

## 📝 部署后配置

### 1. 添加状态徽章到 README

在项目根目录 `README.md` 顶部添加：

```markdown
[![CI/CD](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg)](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml)
[![Server Health](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml/badge.svg)](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml)
```

**注意：** 将 `FangcunMount` 替换为你的 GitHub 用户名/组织名

### 2. 配置分支保护规则

- [ ] 进入 Settings → Branches
- [ ] 为 `main` 分支添加保护规则
- [ ] 启用 "Require a pull request before merging"
- [ ] 启用 "Require status checks to pass before merging"
- [ ] 选择必需的检查：Lint、Test、Build
- [ ] 启用 "Require branches to be up to date"

### 3. 配置通知（可选）

- [ ] 配置 Slack/企业微信通知
- [ ] 设置邮件告警
- [ ] 配置 Issue 自动分配

### 4. 启用定时任务

定时任务会自动运行，确认：
- [ ] Server Health Check - 每小时
- [ ] Ping Runner - 每天凌晨 1:00 UTC

### 5. 文档维护

- [ ] 在团队中分享工作流文档
- [ ] 添加项目特定的配置说明
- [ ] 更新团队开发流程文档

---

## 📚 团队培训检查清单

### 开发人员需要了解

- [ ] 分支策略（main/develop/feature/release）
- [ ] 提交规范（Conventional Commits）
- [ ] PR 流程和 CI 检查
- [ ] 如何查看 Actions 日志
- [ ] 本地测试方法

### 运维人员需要了解

- [ ] 如何配置 Secrets
- [ ] 如何手动触发部署
- [ ] 如何运行健康检查
- [ ] 如何执行数据库操作
- [ ] 回滚流程

### 团队负责人需要了解

- [ ] 整体 CI/CD 流程
- [ ] 环境部署策略
- [ ] 监控和告警机制
- [ ] 安全最佳实践
- [ ] 成本和性能优化

---

## 🎯 成功标准

### 基础目标（必须达成）

- [ ] 所有工作流文件无语法错误
- [ ] CI 流程（Lint、Test、Build）正常运行
- [ ] 至少一个环境部署成功
- [ ] 健康检查工作正常
- [ ] 团队成员了解基本使用方法

### 进阶目标（建议达成）

- [ ] 三个环境都部署成功
- [ ] 定时健康检查正常运行
- [ ] 数据库操作工作流测试通过
- [ ] 状态徽章显示在 README
- [ ] 分支保护规则已配置

### 优化目标（持续改进）

- [ ] 工作流执行时间优化到 10 分钟内
- [ ] 代码覆盖率达到 80% 以上
- [ ] 集成了外部监控工具
- [ ] 配置了自动化通知
- [ ] 建立了完整的文档体系

---

## 📅 实施时间线

| 阶段 | 任务 | 预计时间 | 状态 |
|------|------|----------|------|
| 准备 | 阅读文档、理解流程 | 30 分钟 | ⬜ |
| 配置 | 设置 Secrets | 10 分钟 | ⬜ |
| 测试 | 运行基础测试 | 20 分钟 | ⬜ |
| 部署 | 测试环境部署 | 30 分钟 | ⬜ |
| 验证 | 完整功能验证 | 30 分钟 | ⬜ |
| 优化 | 调优和文档完善 | 1 小时 | ⬜ |

**总计：** 约 2.5 - 3 小时

---

## 🎊 完成标志

当以下所有项都完成时，即表示 CI/CD 实施成功：

- [x] ✅ 所有文件已创建
- [ ] ✅ Secrets 已配置
- [ ] ✅ CI 测试通过
- [ ] ✅ 至少一个环境部署成功
- [ ] ✅ 健康检查正常运行
- [ ] ✅ 团队成员已培训
- [ ] ✅ 文档已更新

---

## 📞 获取支持

如需帮助，请按顺序尝试：

1. 📖 查看相关文档（QUICKSTART.md / README.md）
2. 🔍 检查工作流日志中的详细错误信息
3. 📊 参考 DIAGRAMS.md 理解流程
4. 💬 在仓库创建 Issue 并附上：
   - 问题描述
   - 工作流日志截图
   - 已尝试的解决方法

---

**检查清单版本：** 1.0.0  
**最后更新：** 2025-10-21  
**维护者：** DevOps Team

祝实施顺利！🚀
