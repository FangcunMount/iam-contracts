# GitHub Actions CI/CD 文件清单

本次创建的所有 GitHub Actions 相关文件清单：

## 📁 工作流文件（`.github/workflows/`）

### 1. 核心工作流

| 文件名 | 用途 | 触发方式 |
|--------|------|---------|
| `cicd.yml` | 主 CI/CD 流程 | Push, PR, 标签, 手动 |
| `server-check.yml` | 服务器健康检查 | 定时（每小时）, 手动 |
| `db-ops.yml` | 数据库操作 | 手动 |
| `ping-runner.yml` | Runner 连通性测试 | 定时（每天）, 手动 |

### 2. 配置文件

| 文件名 | 用途 |
|--------|------|
| `secrets.example` | Secrets 配置模板 |
| `README.md` | 工作流详细文档 |
| `QUICKSTART.md` | 快速启动指南 |
| `BADGES.md` | 状态徽章配置 |

---

## 📝 工作流详细说明

### `cicd.yml` - CI/CD Pipeline

**功能模块：**

#### CI 阶段
1. **Lint（代码检查）**
   - golangci-lint 代码检查
   - gofmt 格式检查
   
2. **Test（单元测试）**
   - 运行所有单元测试
   - 生成代码覆盖率报告
   - 上传到 Codecov
   - 使用 MySQL 和 Redis 服务容器

3. **Build（构建）**
   - 构建多平台二进制文件（Linux amd64/arm64）
   - 嵌入版本信息
   - 上传构建产物

4. **Docker Build（镜像构建）**
   - 构建多架构 Docker 镜像
   - 推送到 Docker Hub
   - 自动标签管理

#### CD 阶段
5. **Deploy Dev（开发环境）**
   - 部署到开发服务器
   - 二进制部署方式
   - 自动重启服务

6. **Deploy Staging（预发布环境）**
   - Docker Compose 部署
   - 冒烟测试验证

7. **Deploy Production（生产环境）**
   - 创建备份
   - 蓝绿部署
   - 健康检查
   - 失败自动回滚
   - 创建 GitHub Release

#### 通知阶段
8. **Notify（通知）**
   - 成功通知
   - 失败时自动创建 Issue

**触发条件：**
- Push 到 `main`、`develop`、`release/**` 分支
- Pull Request 到 `main`、`develop`
- 创建标签 `v*.*.*`
- 手动触发

---

### `server-check.yml` - Server Health Check

**功能：**
- ✅ API 端点健康检查
- ✅ API 版本验证
- ✅ API 响应时间监控
- ✅ 数据库连接检查
- ✅ SSL 证书有效期检查
- ✅ 服务器资源监控
- ✅ 认证流程测试

**检查类型：**
- `full` - 完整检查（所有项目）
- `quick` - 快速检查（基本连通性）
- `api-only` - API 专项检查
- `database-only` - 数据库专项检查

**触发条件：**
- 定时任务：每小时一次
- 手动触发

**特性：**
- 生产环境失败自动创建 Issue
- SSL 证书到期预警（30 天内）

---

### `db-ops.yml` - Database Operations

**支持的操作：**

1. **Health Check（健康检查）**
   - 数据库连接测试
   - 表列表查询
   - 数据库大小统计

2. **Backup（备份）**
   - 完整数据库备份
   - 自动压缩
   - 清理旧备份（保留 30 天）

3. **Migrate（迁移）**
   - 执行数据库迁移脚本
   - 更新架构

4. **Seed（数据填充）**
   - 填充测试数据
   - 仅限非生产环境

**触发条件：**
- 手动触发（选择操作类型和环境）

---

### `ping-runner.yml` - Ping Runner

**功能：**
- 显示 Runner 系统信息
- 测试网络连通性
- 检查 Docker 可用性
- 检查 Go 环境
- 生成健康报告

**触发条件：**
- 定时任务：每天凌晨 1:00 UTC（北京时间 9:00）
- 手动触发

---

## 🔐 所需 GitHub Secrets

### 必需 Secrets

```bash
# Docker Hub
DOCKER_USERNAME          # Docker Hub 用户名
DOCKER_PASSWORD          # Docker Hub Token

# API URLs
DEV_API_URL             # 开发环境 API 地址
STAGING_API_URL         # 预发布环境 API 地址
PROD_API_URL            # 生产环境 API 地址

# 数据库配置（每个环境）
DB_HOST_{env}           # 数据库主机
DB_PORT_{env}           # 数据库端口
DB_NAME_{env}           # 数据库名称
DB_USER_{env}           # 数据库用户
DB_PASSWORD_{env}       # 数据库密码

# SSH 配置（可选，用于健康检查）
SERVER_HOST_{env}       # 服务器主机
SERVER_USER_{env}       # SSH 用户
SSH_PRIVATE_KEY_{env}   # SSH 私钥
```

**注意：** `{env}` 为 `dev`、`staging` 或 `prod`

### 可选 Secrets

```bash
CODECOV_TOKEN           # Codecov 上传 Token
SLACK_WEBHOOK_URL       # Slack 通知 Webhook
```

---

## 📚 文档文件说明

### `README.md`
完整的工作流文档，包含：
- 工作流详细说明
- 配置指南
- 使用示例
- 故障排查
- 最佳实践

### `QUICKSTART.md`
5 分钟快速启动指南：
- 快速配置步骤
- 测试验证流程
- 常见问题解决
- 下一步建议

### `BADGES.md`
GitHub 徽章配置：
- CI/CD 状态徽章
- 不同样式示例
- README 集成示例

### `secrets.example`
Secrets 配置模板：
- 完整的 Secrets 列表
- 配置示例
- 安全注意事项

---

## 🚀 快速开始流程

### 1. 配置 Secrets
```bash
1. 进入 GitHub 仓库 Settings
2. 导航到 Secrets and variables -> Actions
3. 根据 secrets.example 添加必需的 Secrets
```

### 2. 测试 CI
```bash
git checkout -b test/ci-setup
echo "test" > test.txt
git add test.txt
git commit -m "test: CI setup"
git push origin test/ci-setup
# 创建 PR 观察 CI 运行
```

### 3. 测试部署
```bash
# 开发环境
git push origin develop

# 预发布环境
git checkout -b release/v0.1.0
git push origin release/v0.1.0

# 生产环境
git tag v0.1.0
git push origin v0.1.0
```

---

## 🎯 工作流矩阵

| 工作流 | 开发 | 预发布 | 生产 | 频率 |
|--------|------|--------|------|------|
| `cicd.yml` | ✅ | ✅ | ✅ | Push/PR/标签 |
| `server-check.yml` | ✅ | ✅ | ✅ | 每小时 |
| `db-ops.yml` | ✅ | ✅ | ✅ | 手动 |
| `ping-runner.yml` | N/A | N/A | N/A | 每天 |

---

## 📊 工作流依赖关系

```
cicd.yml
├── lint
├── test
├── build
│   ├── binary (amd64)
│   └── binary (arm64)
├── docker-build
└── deploy
    ├── deploy-dev (develop 分支)
    ├── deploy-staging (release/* 分支)
    └── deploy-prod (v*.*.* 标签)

server-check.yml
├── health-check
├── api-check
├── database-check
└── ssl-check

db-ops.yml
├── health-check
├── backup
├── migrate
└── seed

ping-runner.yml
└── system-check
```

---

## ✅ 验证清单

配置完成后，请确认：

- [ ] 所有必需的 Secrets 已配置
- [ ] ping-runner 工作流运行成功
- [ ] CI 工作流（lint, test, build）通过
- [ ] 开发环境部署成功
- [ ] server-check 工作流运行正常
- [ ] 文档已添加状态徽章

---

## 🔄 维护建议

### 定期检查（每月）
- [ ] 更新 GitHub Actions 版本
- [ ] 检查并轮换 Secrets
- [ ] 审查工作流性能
- [ ] 清理旧的构建产物
- [ ] 更新文档

### 安全审计（每季度）
- [ ] 审查 Secrets 权限
- [ ] 检查 self-hosted runner 安全
- [ ] 审计第三方 Actions
- [ ] 更新依赖版本

---

## 📞 获取帮助

如遇问题：
1. 查看 `.github/workflows/README.md`
2. 参考 `.github/workflows/QUICKSTART.md`
3. 检查工作流日志
4. 在仓库中创建 Issue

---

**创建日期：** 2025-10-21  
**版本：** 1.0.0  
**维护者：** DevOps Team
