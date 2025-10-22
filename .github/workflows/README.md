# GitHub Actions Workflows

本项目使用 GitHub Actions 实现自动化 CI/CD 流程，针对简化的部署环境（本地开发 + 单个生产服务器）进行了优化。

## 📋 工作流概览

### 1. **ping-runner.yml** - Runner 连通性测试
- **触发方式**: 手动触发 或 每6小时自动执行
- **用途**: 检查生产服务器 A 和 GitHub Runner 的连通性
- **检查内容**:
  - SSH 连接测试
  - 系统资源状态（CPU、内存、磁盘）
  - IAM 服务运行状态
  - GitHub Runner 状态

### 2. **db-ops.yml** - 数据库操作
- **触发方式**: 手动触发
- **支持操作**:
  - `backup`: 备份数据库（保留7天）
  - `restore`: 从备份恢复数据库
  - `migrate`: 运行数据库迁移
  - `status`: 查看数据库状态和可用备份
- **使用方法**:
  ```bash
  # 在 GitHub Actions 页面选择 "Database Operations"
  # 选择操作类型，如需恢复则输入备份文件名
  ```

### 3. **server-check.yml** - 服务器健康检查
- **触发方式**: 手动触发 或 每30分钟自动执行
- **检查内容**:
  - 系统健康状态（CPU、内存、磁盘、负载）
  - IAM 服务状态（自动重启失败的服务）
  - 网络状态和端口监听
  - 数据库连接
  - 磁盘空间预警（>80% 触发警告）

### 4. **cicd.yml** - 主 CI/CD 流程
- **触发方式**: 
  - Push 到 main/develop 分支
  - Pull Request 到 main 分支
  - 手动触发
- **流程**:
  ```
  Test → Lint → Build → Docker → Deploy (仅 main 分支)
  ```
- **部署步骤**:
  1. 备份当前版本
  2. 拉取最新代码
  3. 构建新版本
  4. 停止服务
  5. 运行数据库迁移
  6. 启动服务
  7. 健康检查
  8. 验证部署

## 🔐 必需的 Secrets 配置

在 GitHub 仓库设置中添加以下 Secrets：

### 服务器连接
```
PRODUCTION_HOST=your-server-ip
PRODUCTION_USERNAME=deploy-user
PRODUCTION_SSH_KEY=<your-private-ssh-key>
PRODUCTION_SSH_PORT=22  # 可选，默认22
```

### 数据库配置
```
DB_HOST=localhost
DB_USERNAME=iam_user
DB_PASSWORD=your-db-password
DB_DATABASE=iam
```

## 🚀 使用指南

### 首次配置

1. **设置 GitHub Secrets**
   ```bash
   # 生成 SSH 密钥对
   ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions
   
   # 将公钥添加到生产服务器
   ssh-copy-id -i ~/.ssh/github_actions.pub user@server-ip
   
   # 在 GitHub 仓库设置中添加私钥内容到 PRODUCTION_SSH_KEY
   ```

2. **配置生产服务器**
   ```bash
   # 在生产服务器上创建必要目录
   sudo mkdir -p /opt/iam-contracts
   sudo mkdir -p /opt/backups/iam/{deployments,database}
   sudo chown -R deploy-user:deploy-user /opt/iam-contracts /opt/backups/iam
   
   # 克隆仓库
   cd /opt
   git clone https://github.com/FangcunMount/iam-contracts.git
   cd iam-contracts
   ```

3. **设置 systemd 服务**
   ```bash
   sudo cp build/systemd/iam-apiserver.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable iam-apiserver
   ```

### 日常使用

#### 自动部署（推荐）
```bash
# 提交代码到 main 分支会自动触发部署
git add .
git commit -m "feat: add new feature"
git push origin main
```

#### 手动部署
1. 访问 GitHub Actions 页面
2. 选择 "CI/CD Pipeline"
3. 点击 "Run workflow"
4. 选择分支并运行

#### 数据库备份
```bash
# 方式1: 通过 GitHub Actions（推荐）
# Actions → Database Operations → Run workflow → 选择 "backup"

# 方式2: 在服务器上手动执行
ssh user@server-ip
mysqldump -h localhost -u iam_user -p iam > /opt/backups/iam/manual_backup.sql
gzip /opt/backups/iam/manual_backup.sql
```

#### 数据库恢复
```bash
# 1. 在 GitHub Actions 查看可用备份
# Actions → Database Operations → Run workflow → 选择 "status"

# 2. 恢复数据库
# Actions → Database Operations → Run workflow → 选择 "restore"
# 输入备份文件名: iam_backup_20231022_120000.sql.gz
```

## 📊 监控和告警

### 查看工作流状态
- 访问: `https://github.com/FangcunMount/iam-contracts/actions`
- 每个工作流执行都有详细日志

### 健康检查时间表
- **Runner 连通性**: 每6小时
- **服务器健康**: 每30分钟
- **部署验证**: 每次部署后

### 状态徽章
在项目 README.md 中添加：

```markdown
![CI/CD](https://github.com/FangcunMount/iam-contracts/workflows/CI/CD%20Pipeline/badge.svg)
![Health Check](https://github.com/FangcunMount/iam-contracts/workflows/Server%20Health%20Check/badge.svg)
```

## 🔧 故障排查

### 部署失败
1. 查看 GitHub Actions 日志
2. SSH 到生产服务器检查
   ```bash
   ssh user@server-ip
   sudo journalctl -u iam-apiserver -f
   sudo systemctl status iam-apiserver
   ```

### 服务未启动
```bash
# 检查服务状态
sudo systemctl status iam-apiserver

# 查看日志
sudo journalctl -u iam-apiserver -n 100

# 手动启动
sudo systemctl start iam-apiserver
```

### 回滚到之前版本
```bash
ssh user@server-ip
cd /opt/iam-contracts

# 查看可用备份
ls -lh /opt/backups/iam/deployments/

# 恢复备份
BACKUP_FILE="backup_20231022_120000.tar.gz"
sudo systemctl stop iam-apiserver
tar -xzf /opt/backups/iam/deployments/$BACKUP_FILE -C /opt/iam-contracts
sudo systemctl start iam-apiserver
```

## 📝 最佳实践

1. **提交前本地测试**
   ```bash
   make test
   make lint
   make build
   ```

2. **使用有意义的提交信息**
   ```bash
   # 好的例子
   git commit -m "feat: add user authentication"
   git commit -m "fix: resolve database connection issue"
   git commit -m "docs: update API documentation"
   
   # 避免
   git commit -m "update"
   git commit -m "fix bug"
   ```

3. **定期备份数据库**
   - 自动备份保留7天
   - 重要操作前手动备份
   - 定期测试恢复流程

4. **监控服务器资源**
   - 关注磁盘空间警告
   - 检查服务健康状态
   - 定期清理日志和临时文件

## 🔄 工作流更新

修改工作流文件后：
```bash
git add .github/workflows/
git commit -m "ci: update workflow configuration"
git push origin main
```

工作流文件位于：
- `.github/workflows/ping-runner.yml`
- `.github/workflows/db-ops.yml`
- `.github/workflows/server-check.yml`
- `.github/workflows/cicd.yml`

## 📞 支持

如有问题，请：
1. 查看 GitHub Actions 日志
2. 检查服务器日志
3. 查阅项目文档
4. 提交 Issue

---

**环境说明**:
- 开发环境: 本地 MacBook
- 生产环境: 服务器 A (单机部署)
- CI/CD: GitHub Actions
- 部署方式: SSH + systemd
