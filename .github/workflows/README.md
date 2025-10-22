# GitHub Actions Workflows

本项目使用 GitHub Actions 实现自动化 CI/CD 流程，针对简化的部署环境（本地开发 + 单个生产服务器）进行了优化。

## 📋 目录

- [工作流概览](#工作流概览)
- [环境配置](#环境配置)
- [Secrets 配置](#secrets-配置)
- [使用指南](#使用指南)
- [故障排查](#故障排查)
- [最佳实践](#最佳实践)

---

## 工作流概览

### 1. **ping-runner.yml** - Runner 连通性测试

- **触发方式**: 手动触发 或 每6小时自动执行
- **用途**: 检查生产服务器和 GitHub Runner 的连通性
- **运行时间**: ~1 分钟
- **检查内容**:
  - SSH 连接测试
  - 系统资源状态（CPU、内存、磁盘）
  - IAM 服务运行状态
  - GitHub Runner 状态

### 2. **db-ops.yml** - 数据库操作

- **触发方式**: 手动触发
- **运行时间**: 视操作而定
- **支持操作**:
  - `backup`: 备份数据库（保留最近10次备份）
  - `restore`: 从备份恢复数据库
  - `migrate`: 运行数据库迁移
  - `status`: 查看数据库状态和可用备份

**使用方法**:

```bash
# 在 GitHub 页面操作
Actions → Database Operations → Run workflow → 选择操作类型
# 如需恢复，输入备份文件名（如：iam_backup_20231022_120000.sql.gz）
```

### 3. **server-check.yml** - 服务器健康检查

- **触发方式**: 手动触发 或 每30分钟自动执行
- **运行时间**: ~2-3 分钟
- **检查内容**:
  - 系统健康状态（CPU、内存、磁盘、负载）
  - IAM 服务状态（自动重启失败的服务）
  - 网络状态和端口监听
  - 数据库连接测试
  - 磁盘空间预警（>80% 触发警告）

### 4. **cicd.yml** - 主 CI/CD 流程

- **触发方式**:
  - Push 到 main/develop 分支
  - Pull Request 到 main 分支
  - 手动触发
- **运行时间**: ~10-15 分钟
- **流程**:

```text
Test (3-5分钟)
  ↓
Lint (2-3分钟)
  ↓
Build (2-3分钟)
  ↓
Docker (3-4分钟) ← 仅 main 分支
  ↓
Deploy (2-3分钟) ← 仅 main 分支
  ↓
Verify (验证部署)
```

**部署步骤**:

1. 备份当前版本
2. 拉取最新代码
3. 构建 Docker 镜像
4. 停止服务
5. 运行数据库迁移
6. 启动服务
7. 健康检查
8. 验证部署

---

## 环境配置

### 当前架构

```text
开发环境: MacBook (本地开发)
    ↓
  GitHub
    ↓
生产环境: SVRA (服务器 A)
```

### 服务器要求

- **操作系统**: Linux (推荐 Ubuntu 20.04+)
- **Go 版本**: 1.21+
- **Docker**: 用于容器化部署
- **MySQL**: 5.7+ 或 8.0+
- **Redis**: 5.0+
- **systemd**: 用于服务管理

---

## Secrets 配置

### 配置步骤

1. **进入 Settings**
   - **Repository Secrets**: `Settings` → `Secrets and variables` → `Actions`
   - **Organization Secrets**: 组织设置 → `Secrets and variables` → `Actions`

2. **点击 `New repository secret` 或 `New organization secret`**

3. **添加以下 Secrets**

### 必需的 Secrets

#### Organization Secrets（组织级别，8个）

| Secret 名称 | 说明 | 示例值 |
|------------|------|--------|
| `SVRA_HOST` | 生产服务器 IP 或域名 | `192.168.1.100` 或 `svra.example.com` |
| `SVRA_USERNAME` | SSH 登录用户名 | `deploy` 或 `root` |
| `SVRA_SSH_KEY` | SSH 私钥（完整内容） | 见下方 SSH 配置 |
| `SVRA_SSH_PORT` | SSH 端口 | `22`（默认） |
| `MYSQL_HOST` | MySQL 服务器地址 | `192.168.1.101` |
| `MYSQL_PORT` | MySQL 端口 | `3306` |
| `REDIS_HOST` | Redis 服务器地址 | `192.168.1.102` |
| `REDIS_PORT` | Redis 端口 | `6379` |

#### Repository Secrets（仓库级别，5个）

| Secret 名称 | 说明 | 示例值 |
|------------|------|--------|
| `MYSQL_USERNAME` | MySQL 用户名 | `iam_user` |
| `MYSQL_PASSWORD` | MySQL 密码 | `your_secure_password` |
| `MYSQL_DBNAME` | 数据库名称 | `iam_db` |
| `REDIS_PASSWORD` | Redis 密码 | `your_redis_password` |
| `REDIS_DB` | Redis 数据库编号 | `0` |

### SSH 密钥配置

#### 1. 生成 SSH 密钥对

```bash
# 在本地生成密钥
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy

# 会生成两个文件:
# ~/.ssh/github_actions_deploy      (私钥)
# ~/.ssh/github_actions_deploy.pub  (公钥)
```

#### 2. 配置生产服务器

```bash
# 将公钥添加到 SVRA 服务器
ssh-copy-id -i ~/.ssh/github_actions_deploy.pub user@svra-host

# 或手动添加
cat ~/.ssh/github_actions_deploy.pub | ssh user@svra-host "cat >> ~/.ssh/authorized_keys"

# 在服务器上设置权限
ssh user@svra-host "chmod 600 ~/.ssh/authorized_keys"
```

#### 3. 添加私钥到 GitHub

```bash
# 复制私钥内容
cat ~/.ssh/github_actions_deploy

# 在 GitHub 上添加:
# Settings → Secrets → New secret
# Name: SVRA_SSH_KEY
# Value: 粘贴完整的私钥内容（包括 -----BEGIN 和 -----END 行）
```

#### 4. 测试连接

```bash
# 使用私钥测试连接
ssh -i ~/.ssh/github_actions_deploy user@svra-host
```

### ✅ 验证配置

配置完成后，运行以下工作流验证：

```bash
# 1. 测试 SSH 连通性
Actions → Ping Runner → Run workflow

# 2. 查看数据库状态
Actions → Database Operations → Run workflow → 选择 "status"

# 3. 健康检查
Actions → Server Health Check → Run workflow
```

---

## 使用指南

### 首次部署

#### 1. 配置生产服务器

```bash
# SSH 登录到 SVRA
ssh user@svra-host

# 创建必要目录
sudo mkdir -p /opt/iam
sudo mkdir -p /opt/backups/iam
sudo chown -R $USER:$USER /opt/iam /opt/backups/iam

# 克隆仓库
cd /opt
git clone https://github.com/FangcunMount/iam-contracts.git
cd iam-contracts

# 安装依赖
go mod download
```

#### 2. 配置 systemd 服务

```bash
# 复制服务文件
sudo cp build/systemd/iam-apiserver.service /etc/systemd/system/

# 重载 systemd
sudo systemctl daemon-reload

# 启用服务
sudo systemctl enable iam-apiserver

# 启动服务
sudo systemctl start iam-apiserver

# 检查状态
sudo systemctl status iam-apiserver
```

#### 3. 配置应用

```bash
# 编辑配置文件
vim configs/apiserver.yaml

# 或使用环境变量
vim configs/env/config.prod.env
```

### 日常使用

#### 自动部署（推荐）

```bash
# 提交代码到 develop 分支（测试）
git checkout develop
git add .
git commit -m "feat: add new feature"
git push origin develop

# 合并到 main 分支（生产部署）
git checkout main
git merge develop
git push origin main
# 自动触发 CI/CD → 测试 → 构建 → 部署
```

#### 手动触发部署

```bash
# 在 GitHub 页面
Actions → CI/CD Pipeline → Run workflow → 选择分支 → Run
```

#### 数据库备份

```bash
# 方式1: GitHub Actions（推荐）
Actions → Database Operations → Run workflow → 选择 "backup"

# 方式2: 服务器上手动备份
ssh user@svra-host
mysqldump -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USERNAME -p$MYSQL_PASSWORD $MYSQL_DBNAME > backup.sql
gzip backup.sql
```

#### 数据库恢复

```bash
# 1. 查看可用备份
Actions → Database Operations → Run workflow → 选择 "status"

# 2. 恢复指定备份
Actions → Database Operations → Run workflow → 选择 "restore"
# 输入备份文件名: iam_backup_20231022_120000.sql.gz
```

#### 查看服务状态

```bash
# 方式1: GitHub Actions
Actions → Server Health Check → Run workflow

# 方式2: SSH 登录查看
ssh user@svra-host
systemctl status iam-apiserver
journalctl -u iam-apiserver -f
```

### 监控和告警

#### 查看工作流状态

- 访问: `https://github.com/FangcunMount/iam-contracts/actions`
- 每个工作流执行都有详细日志

#### 自动健康检查时间表

- **Runner 连通性**: 每6小时自动检查
- **服务器健康**: 每30分钟自动检查
- **部署验证**: 每次部署后自动验证

#### 添加状态徽章

在项目 README.md 中添加：

```markdown
![CI/CD](https://github.com/FangcunMount/iam-contracts/workflows/CI/CD%20Pipeline/badge.svg)
![Health Check](https://github.com/FangcunMount/iam-contracts/workflows/Server%20Health%20Check/badge.svg)
```

---

## 故障排查

### 常见问题

#### 1. SSH 连接失败

**错误信息**: `Permission denied (publickey)`

**解决方案**:

```bash
# 1. 检查私钥是否正确配置在 GitHub Secrets
# 2. 验证公钥在服务器上
ssh user@svra-host "cat ~/.ssh/authorized_keys | grep github-actions"

# 3. 测试本地连接
ssh -i ~/.ssh/github_actions_deploy user@svra-host

# 4. 检查服务器 SSH 配置
ssh user@svra-host "sudo cat /etc/ssh/sshd_config | grep PubkeyAuthentication"
# 确保: PubkeyAuthentication yes

# 5. 查看 SSH 日志
ssh user@svra-host "sudo journalctl -u ssh -n 50"
```

#### 2. 部署失败

**排查步骤**:

```bash
# 1. 查看 GitHub Actions 日志
Actions → 失败的 workflow → 查看详细日志

# 2. SSH 登录服务器检查
ssh user@svra-host

# 3. 检查服务状态
sudo systemctl status iam-apiserver

# 4. 查看应用日志
sudo journalctl -u iam-apiserver -n 100 --no-pager

# 5. 检查磁盘空间
df -h

# 6. 检查内存使用
free -h
```

#### 3. 服务未启动

```bash
# 检查服务状态
sudo systemctl status iam-apiserver

# 查看错误日志
sudo journalctl -u iam-apiserver -n 100

# 检查配置文件
cat /opt/iam/configs/apiserver.yaml

# 手动启动服务
sudo systemctl start iam-apiserver

# 如果仍失败，查看详细错误
sudo systemctl start iam-apiserver -l
```

#### 4. 数据库连接失败

**错误信息**: `Access denied for user` 或 `Can't connect to MySQL server`

**解决方案**:

```bash
# 1. 测试数据库连接
mysql -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USERNAME -p

# 2. 检查数据库用户权限
mysql -u root -p
> SELECT user, host FROM mysql.user WHERE user='iam_user';
> SHOW GRANTS FOR 'iam_user'@'%';

# 3. 检查防火墙
sudo ufw status
sudo iptables -L -n | grep 3306

# 4. 检查 MySQL 绑定地址
sudo cat /etc/mysql/mysql.conf.d/mysqld.cnf | grep bind-address
# 应该是: bind-address = 0.0.0.0 或注释掉
```

#### 5. 回滚到之前版本

```bash
# 1. SSH 登录服务器
ssh user@svra-host

# 2. 查看可用备份
ls -lh /opt/backups/iam/deployments/

# 3. 停止服务
sudo systemctl stop iam-apiserver

# 4. 恢复备份
cd /opt/iam
BACKUP_FILE="backup_20231022_120000.tar.gz"
tar -xzf /opt/backups/iam/deployments/$BACKUP_FILE

# 5. 启动服务
sudo systemctl start iam-apiserver

# 6. 验证服务
sudo systemctl status iam-apiserver
curl http://localhost:8080/healthz
```

### 日志查看

```bash
# GitHub Actions 日志
GitHub → Actions → 选择 workflow run → 查看每个 job 的日志

# 服务器系统日志
sudo journalctl -u iam-apiserver -f          # 实时查看
sudo journalctl -u iam-apiserver -n 100       # 查看最近100行
sudo journalctl -u iam-apiserver --since today # 查看今天的日志

# 应用日志（如果配置了文件日志）
tail -f /var/log/iam/apiserver.log
```

---

## 最佳实践

### 开发流程

#### 1. 提交前本地测试

```bash
# 运行测试
make test

# 代码检查
make lint

# 本地构建
make build

# 运行服务
./_output/bin/iam-apiserver
```

#### 2. 使用有意义的提交信息

```bash
# 推荐的提交格式
feat: 新功能
fix: 修复bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建或辅助工具变动

# 好的例子
git commit -m "feat: add user authentication"
git commit -m "fix: resolve database connection timeout"
git commit -m "docs: update API documentation"

# 避免
git commit -m "update"
git commit -m "fix bug"
git commit -m "changes"
```

#### 3. 分支管理

```bash
# 功能开发
git checkout -b feature/user-management
git push origin feature/user-management
# 创建 PR → 代码审查 → 合并到 develop

# 紧急修复
git checkout -b hotfix/critical-bug
git push origin hotfix/critical-bug
# 创建 PR → 测试 → 合并到 main
```

### 部署策略

#### 1. 定期备份数据库

```bash
# 自动备份（已配置）
# - GitHub Actions 手动触发
# - 保留最近10次备份

# 重要操作前手动备份
Actions → Database Operations → backup

# 定期测试恢复流程（每月一次）
Actions → Database Operations → restore
```

#### 2. 监控服务器资源

```bash
# 自动监控（已配置）
# - 每30分钟健康检查
# - 磁盘空间 >80% 告警

# 手动检查
ssh user@svra-host
df -h                    # 磁盘空间
free -h                  # 内存使用
top                      # CPU 和进程
systemctl status iam-apiserver
```

#### 3. 日志管理

```bash
# 定期清理日志（建议每月）
ssh user@svra-host
sudo journalctl --vacuum-time=30d  # 保留30天
sudo journalctl --vacuum-size=1G   # 限制1GB
```

### 安全实践

#### 1. 定期更新密钥

```bash
# 建议每3-6个月更新
# - SSH 密钥
# - 数据库密码
# - Redis 密码
# - API tokens
```

#### 2. 最小权限原则

```bash
# 数据库用户只授予必要权限
CREATE USER 'iam_user'@'%' IDENTIFIED BY 'password';
GRANT SELECT, INSERT, UPDATE, DELETE ON iam_db.* TO 'iam_user'@'%';
# 不要授予 DROP, CREATE, ALTER 等权限
```

#### 3. 审计日志

```bash
# 定期检查（建议每周）
# - GitHub Actions 执行历史
# - 失败的部署记录
# - 服务器登录日志
# - 数据库访问日志
```

### 性能优化

#### 1. 构建缓存

```yaml
# GitHub Actions 已配置 Go 模块缓存
# Docker 层缓存
# 减少构建时间 30-50%
```

#### 2. 并行执行

```yaml
# test 和 lint 可以并行执行
# 多个健康检查并行运行
```

#### 3. 工作流优化

```bash
# 只在必要时触发完整流程
# PR: 只运行 test + lint
# Push to develop: test + lint + build
# Push to main: 完整 CI/CD 流程
```

---

## 附加资源

### 相关文档

- [架构概览](../../docs/architecture-overview.md)
- [部署检查清单](../../docs/DEPLOYMENT_CHECKLIST.md)
- [API 参考](../../docs/authn/API_REFERENCE.md)

### 外部链接

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [GitHub Secrets 安全指南](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Docker 最佳实践](https://docs.docker.com/develop/dev-best-practices/)

### 命令行工具

```bash
# GitHub CLI (gh)
brew install gh
gh auth login
gh workflow list
gh run list
gh run view <run-id> --log

# Docker
docker ps
docker logs iam-apiserver
docker system prune -a  # 清理未使用的镜像

# systemd
systemctl status iam-apiserver
journalctl -u iam-apiserver -f
systemctl restart iam-apiserver
```

---

## 🎯 快速参考

### 常用命令速查

```bash
# 触发部署
git push origin main

# 查看服务状态
ssh user@svra "systemctl status iam-apiserver"

# 查看日志
ssh user@svra "journalctl -u iam-apiserver -n 50"

# 备份数据库
Actions → Database Operations → backup

# 健康检查
Actions → Server Health Check → Run workflow

# 回滚部署
ssh user@svra "cd /opt/iam && git checkout <commit-hash>"
ssh user@svra "systemctl restart iam-apiserver"
```

### 工作流执行时间

| 工作流 | 平均时间 | 触发方式 |
|--------|---------|---------|
| Ping Runner | ~1分钟 | 手动/每6小时 |
| CI/CD Pipeline | ~10-15分钟 | push/PR/手动 |
| Database Operations | 1-5分钟 | 手动 |
| Server Health Check | ~2-3分钟 | 手动/每30分钟 |

### Secrets 清单

**✅ Organization Secrets (8个)**:
SVRA_HOST, SVRA_USERNAME, SVRA_SSH_KEY, SVRA_SSH_PORT,
MYSQL_HOST, MYSQL_PORT, REDIS_HOST, REDIS_PORT

**✅ Repository Secrets (5个)**:
MYSQL_USERNAME, MYSQL_PASSWORD, MYSQL_DBNAME,
REDIS_PASSWORD, REDIS_DB

---

## 获取帮助

### 问题排查顺序

1. **查看 GitHub Actions 日志** - 最详细的错误信息
2. **检查服务器日志** - `journalctl -u iam-apiserver`
3. **验证 Secrets 配置** - 确保所有 Secrets 正确配置
4. **测试连通性** - 运行 ping-runner 工作流
5. **查看本文档** - 查找相关故障排查步骤

### 支持渠道

- **GitHub Issues**: 提交问题和功能请求
- **Pull Requests**: 提交改进和修复
- **文档**: 查阅项目文档目录

---

**最后更新**: 2025年10月22日

**环境**: 开发环境（MacBook）+ 生产环境（SVRA 服务器 A）

**CI/CD**: GitHub Actions + Docker + systemd
