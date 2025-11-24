# Docker Hub 备份配置指南

本文档介绍如何配置 CI/CD 流程，使 Docker 镜像同时推送到 GitHub Container Registry (GHCR) 和 Docker Hub 进行备份。

## 📋 概述

**当前配置**：

- **主仓库**：`ghcr.io/fangcunmount/iam-contracts` (GitHub Container Registry)
- **备份仓库**：`<你的用户名>/iam-contracts` (Docker Hub)

**推送策略**：

- 每次推送到 `main` 分支时自动触发
- 同时推送两个标签：
  - `latest`：最新版本
  - `<git-sha>`：特定提交版本（如 `a1b2c3d`）

---

## 🔑 步骤 1：创建 Docker Hub Access Token

### 1.1 登录 Docker Hub

访问 [Docker Hub](https://hub.docker.com/) 并登录你的账户。

### 1.2 创建 Access Token

1. 进入 **Account Settings** → **Security**
2. 找到 **Access Tokens** 部分
3. 点击 **New Access Token** 按钮
4. 填写信息：
   - **Access Token Description**: `GitHub Actions IAM Contracts`
   - **Access permissions**: `Read & Write`
5. 点击 **Generate**
6. **立即复制 Token**（⚠️ Token 只显示一次，关闭后无法再查看）

### 1.3 Token 示例

```text
Token 格式：dckr_pat_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
保存位置：安全的密码管理器中
```

---

## 🔐 步骤 2：在 GitHub 配置 Secrets

### 2.1 进入仓库设置

1. 打开 GitHub 仓库：`https://github.com/FangcunMount/iam-contracts`
2. 进入 **Settings** → **Secrets and variables** → **Actions**

### 2.2 添加 Secrets

点击 **New repository secret**，添加以下两个 Secrets：

#### Secret 1: DOCKERHUB_USERNAME

```text
Name: DOCKERHUB_USERNAME
Secret: <你的 Docker Hub 用户名>
```

**示例**：

- 如果你的 Docker Hub 用户名是 `yangshujie`
- 镜像地址将是 `yangshujie/iam-contracts`

#### Secret 2: DOCKERHUB_TOKEN

```text
Name: DOCKERHUB_TOKEN
Secret: <刚才复制的 Access Token>
```

**示例格式**：

```text
Token 以 dckr_pat_ 开头，后跟随机字符串
```

### 2.3 验证配置

配置完成后，你应该在 Secrets 列表中看到：

```text
✓ DOCKERHUB_USERNAME
✓ DOCKERHUB_TOKEN
✓ SVRA_HOST
✓ SVRA_USERNAME
✓ SVRA_SSH_KEY
✓ MYSQL_USERNAME
✓ MYSQL_PASSWORD
✓ MYSQL_DBNAME
... (其他 Secrets)
```

---

## 🚀 步骤 3：测试推送

### 3.1 触发 CI/CD

推送代码到 `main` 分支：

```bash
git add .
git commit -m "test: 测试 Docker Hub 推送"
git push origin main
```

### 3.2 查看 Actions 日志

1. 进入 **Actions** 标签页
2. 找到最新的 **CI/CD Pipeline** 工作流
3. 点击 **docker** job
4. 查看 **Tag and Push to Docker Hub** 步骤

### 3.3 预期输出

成功推送后应该看到：

```text
✅ 镜像已推送到 Docker Hub:
   - yangshujie/iam-contracts:latest
   - yangshujie/iam-contracts:a1b2c3d456e789f0123456789abcdef012345678
```

### 3.4 在 Docker Hub 验证

1. 访问 `https://hub.docker.com/r/<你的用户名>/iam-contracts`
2. 应该看到两个标签：
   - `latest`
   - `<git-sha>`

---

## 🛠️ 使用备份镜像

### 拉取镜像

```bash
# 拉取 latest 版本
docker pull <你的用户名>/iam-contracts:latest

# 拉取特定版本
docker pull <你的用户名>/iam-contracts:a1b2c3d456e789f0123456789abcdef012345678
```

### 在服务器上使用

如果 GHCR 不可用，可以切换到 Docker Hub：

```bash
# 修改部署脚本或 docker-compose.prod.yml
# 从：ghcr.io/fangcunmount/iam-contracts:latest
# 改为：<你的用户名>/iam-contracts:latest

docker pull <你的用户名>/iam-contracts:latest
docker run -d \
  --name iam-apiserver \
  -p 9080:8080 \
  -p 9444:9444 \
  --env-file .env \
  <你的用户名>/iam-contracts:latest
```

---

## 🔍 故障排查

### 问题 1：推送失败 - 认证错误

**错误信息**：

```text
Error: unauthorized: authentication required
```

**解决方法**：

1. 检查 `DOCKERHUB_USERNAME` 是否正确
2. 检查 `DOCKERHUB_TOKEN` 是否有效
3. 重新生成 Token 并更新 Secret

### 问题 2：镜像名称不合法

**错误信息**：

```text
Error: invalid reference format
```

**解决方法**：

- Docker Hub 用户名必须全小写
- 镜像名称不能包含大写字母、空格或特殊字符

### 问题 3：权限不足

**错误信息**：

```text
Error: denied: requested access to the resource is denied
```

**解决方法**：

1. 确认 Access Token 权限为 **Read & Write**
2. 确认 Docker Hub 仓库存在或允许自动创建
3. 检查 Docker Hub 账户状态

### 问题 4：查看详细日志

在 GitHub Actions 中：

1. 进入失败的 workflow
2. 点击 **docker** job
3. 展开 **Login to Docker Hub** 和 **Tag and Push to Docker Hub** 步骤
4. 查看详细错误信息

---

## 📊 监控和维护

### 定期检查

- **每月检查**：Docker Hub Access Token 是否即将过期
- **每周检查**：镜像推送是否正常
- **存储管理**：Docker Hub 免费账户有存储限制，定期清理旧镜像

### 镜像清理策略

Docker Hub 免费账户限制：

- **镜像数量**：无限制
- **存储空间**：有限制（根据账户类型）
- **拉取次数**：6 个月内 200 次（匿名）/ 无限制（认证）

**建议**：

- 保留最近 10 个 git-sha 标签
- `latest` 标签始终保留
- 定期清理超过 3 个月的旧版本

### 手动清理镜像

```bash
# 删除特定标签
docker rmi <你的用户名>/iam-contracts:<旧版本-sha>
docker push <你的用户名>/iam-contracts:<旧版本-sha> --delete
```

或在 Docker Hub 网页端操作：

1. 进入仓库 **Tags** 页面
2. 选择要删除的标签
3. 点击 **Delete**

---

## 🔒 安全最佳实践

1. **永远不要在代码中硬编码 Token**
2. **使用 Access Token 而不是密码**
3. **定期轮换 Token**（建议每 6 个月）
4. **最小权限原则**：只授予 CI/CD 必需的权限
5. **监控异常活动**：定期检查 Docker Hub 活动日志

---

## 📚 相关资源

- [Docker Hub 官方文档](https://docs.docker.com/docker-hub/)
- [GitHub Actions docker/login-action](https://github.com/docker/login-action)
- [GitHub Secrets 文档](https://docs.github.com/en/actions/security-guides/encrypted-secrets)

---

## 📞 支持

如有问题，请：

1. 查看 [故障排查](#-故障排查) 部分
2. 在 GitHub Issues 中提问
3. 联系项目维护者

---

**配置完成！** 🎉

现在你的 Docker 镜像会自动备份到 Docker Hub，提供额外的可靠性和可用性。
