# Sudoers 配置指南

本文档说明如何配置服务器以支持 GitHub Actions 自动部署时无需密码执行 sudo 命令。

## 问题描述

GitHub Actions 通过 SSH 执行部署脚本时，遇到以下错误：

```text
sudo: a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper
sudo: a password is required
```

**原因**：SSH 非交互式会话无法输入密码，导致所有 `sudo` 命令失败。

---

## 解决方案

配置服务器允许部署用户无密码执行 sudo 命令。

### 方案 1：允许所有 sudo 命令（简单但安全性较低）

#### 1. SSH 登录服务器

```bash
ssh <你的用户名>@<服务器地址>
```

#### 2. 编辑 sudoers 文件

```bash
sudo visudo
```

⚠️ **重要**：必须使用 `visudo` 而不是直接编辑 `/etc/sudoers`，以防止语法错误导致系统锁定。

#### 3. 添加配置

在文件末尾添加：

```text
# 允许部署用户无密码执行所有 sudo 命令
<你的用户名> ALL=(ALL) NOPASSWD: ALL
```

**示例**（如果用户名是 `yangshujie`）：

```text
yangshujie ALL=(ALL) NOPASSWD: ALL
```

#### 4. 保存并退出

- **nano 编辑器**：按 `Ctrl+X`，然后 `Y`，然后 `Enter`
- **vim 编辑器**：按 `ESC`，输入 `:wq`，然后 `Enter`

#### 5. 验证配置

```bash
# 测试 sudo 命令（不应要求密码）
sudo docker ps
sudo mkdir -p /tmp/test
sudo rm -rf /tmp/test
```

---

### 方案 2：只允许特定命令（推荐，安全性更高）

#### 1-2. 同方案 1

#### 3. 添加精细化配置

在文件末尾添加：

```text
# 允许部署用户无密码执行部署相关的 sudo 命令
<你的用户名> ALL=(ALL) NOPASSWD: /usr/bin/docker, \
                                     /usr/bin/mkdir, \
                                     /usr/bin/tar, \
                                     /usr/bin/cp, \
                                     /usr/bin/chown, \
                                     /usr/bin/chmod, \
                                     /usr/bin/tee, \
                                     /usr/bin/systemctl, \
                                     /usr/bin/rm, \
                                     /usr/bin/sed, \
                                     /usr/bin/ls
```

**完整示例**：

```text
# GitHub Actions 部署用户配置
yangshujie ALL=(ALL) NOPASSWD: /usr/bin/docker, \
                                /usr/bin/mkdir, \
                                /usr/bin/tar, \
                                /usr/bin/cp, \
                                /usr/bin/chown, \
                                /usr/bin/chmod, \
                                /usr/bin/tee, \
                                /usr/bin/systemctl, \
                                /usr/bin/rm, \
                                /usr/bin/sed, \
                                /usr/bin/ls
```

#### 4-5. 同方案 1

---

## 配置说明

### sudoers 语法

```text
用户名 主机=(运行用户) NOPASSWD: 命令列表
```

**字段解释**：

- `用户名`：允许无密码 sudo 的用户
- `ALL`：在所有主机上生效
- `(ALL)`：可以以任何用户身份运行命令
- `NOPASSWD:`：不需要密码
- `命令列表`：允许的命令（绝对路径）

### 查找命令路径

如果不确定命令的绝对路径：

```bash
which docker    # 输出: /usr/bin/docker
which tar       # 输出: /usr/bin/tar
which mkdir     # 输出: /usr/bin/mkdir
```

---

## 安全建议

### ✅ 推荐做法

1. **使用方案 2**（只允许必需命令）
2. **限制 SSH 密钥访问**：确保只有 GitHub Actions 使用的 SSH 密钥可以登录
3. **定期审计**：检查 sudoers 配置和用户权限
4. **使用专用部署用户**：创建专门用于部署的用户，而不是使用管理员账户

### ❌ 不推荐做法

1. 在 `/etc/sudoers` 中直接编辑（使用 `visudo`）
2. 允许所有用户 NOPASSWD（只配置部署用户）
3. 在生产环境中使用 root 用户进行部署

---

## 创建专用部署用户（可选，最佳实践）

如果你想创建一个专门用于部署的用户：

```bash
# 1. 创建用户
sudo useradd -m -s /bin/bash deploy

# 2. 添加到 docker 组（可选）
sudo usermod -aG docker deploy

# 3. 配置 SSH 密钥
sudo mkdir -p /home/deploy/.ssh
sudo vim /home/deploy/.ssh/authorized_keys
# 粘贴 GitHub Actions 使用的 SSH 公钥

# 4. 设置权限
sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh

# 5. 配置 sudoers
sudo visudo
# 添加: deploy ALL=(ALL) NOPASSWD: /usr/bin/docker, ...
```

然后在 GitHub Secrets 中：

- `SVRA_USERNAME` 改为 `deploy`
- `SVRA_SSH_KEY` 使用专用密钥

---

## 故障排查

### 问题 1：配置后仍然要求密码

**原因**：sudoers 文件语法错误或配置被覆盖

**解决**：

```bash
# 检查 sudoers 语法
sudo visudo -c

# 查看当前用户的 sudo 权限
sudo -l

# 检查 /etc/sudoers.d/ 目录是否有冲突配置
ls -la /etc/sudoers.d/
```

### 问题 2：visudo 提示语法错误

**原因**：配置格式不正确

**解决**：

- 确保每行末尾没有多余空格
- 多行配置使用 `\` 续行
- 命令路径必须是绝对路径
- 检查拼写错误

### 问题 3：特定命令仍需要密码

**原因**：命令路径不匹配或使用了参数

**解决**：

```bash
# 检查实际执行的命令路径
which docker  # 确认路径

# 如果使用 sudo docker run ...
# sudoers 中应该是: /usr/bin/docker 而不是 /usr/bin/docker run
```

### 问题 4：配置后 sudo 完全失效

**原因**：sudoers 文件损坏

**解决**（需要物理访问或 root 权限）：

```bash
# 单用户模式进入系统
# 或使用 root 用户登录
pkexec visudo
# 或
su - root
visudo
```

---

## 验证部署

配置完成后，重新运行 GitHub Actions workflow：

```bash
# 手动触发 workflow
# GitHub 仓库 → Actions → CI/CD Pipeline → Run workflow
```

查看日志，确认不再出现 sudo 密码提示错误。

---

## 相关资源

- [Sudoers 官方文档](https://www.sudo.ws/docs/man/sudoers.man/)
- [Ubuntu Sudoers 配置指南](https://help.ubuntu.com/community/Sudoers)
- [GitHub Actions SSH 部署最佳实践](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments)

---

## 总结

✅ **完成配置后**：

1. 部署用户可以无密码执行 sudo 命令
2. GitHub Actions CI/CD 流程可以正常部署
3. 安全性通过 SSH 密钥和命令白名单保障

⚠️ **安全提醒**：

- 只配置必需的用户和命令
- 定期审查 sudoers 配置
- 保护好 SSH 私钥（存储在 GitHub Secrets 中）
- 启用服务器防火墙和 fail2ban

配置完成！🎉
