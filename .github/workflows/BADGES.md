# GitHub Actions Status Badges

将以下徽章添加到项目的 README.md 中，展示 CI/CD 状态：

## 基本徽章

### CI/CD 状态
```markdown
![CI/CD](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg)
```
![CI/CD](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg)

### 服务器健康检查
```markdown
![Server Health](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml/badge.svg)
```
![Server Health](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml/badge.svg)

### Runner 状态
```markdown
![Ping Runner](https://github.com/FangcunMount/iam-contracts/actions/workflows/ping-runner.yml/badge.svg)
```
![Ping Runner](https://github.com/FangcunMount/iam-contracts/actions/workflows/ping-runner.yml/badge.svg)

---

## 特定分支徽章

### Main 分支
```markdown
![CI/CD - Main](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg?branch=main)
```

### Develop 分支
```markdown
![CI/CD - Develop](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg?branch=develop)
```

---

## 完整示例（添加到 README.md）

```markdown
# IAM Contracts

[![CI/CD](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg)](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml)
[![Server Health](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml/badge.svg)](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

身份与访问管理系统 (Identity and Access Management)

## 特性

- 🚀 完整的 CI/CD 流程
- 🔒 安全的身份认证与授权
- 📊 实时健康监控
- 🐳 Docker 容器化部署
- 🔄 自动化数据库操作

## 快速开始

查看 [快速启动指南](.github/workflows/QUICKSTART.md)

## CI/CD 状态

| 工作流 | 状态 | 描述 |
|--------|------|------|
| CI/CD Pipeline | ![CI/CD](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg) | 持续集成与部署 |
| Server Health Check | ![Health](https://github.com/FangcunMount/iam-contracts/actions/workflows/server-check.yml/badge.svg) | 服务器健康检查 |
| Database Operations | ![DB Ops](https://github.com/FangcunMount/iam-contracts/actions/workflows/db-ops.yml/badge.svg) | 数据库操作 |
| Ping Runner | ![Runner](https://github.com/FangcunMount/iam-contracts/actions/workflows/ping-runner.yml/badge.svg) | Runner 连通性 |

## 文档

- [架构概览](docs/architecture-overview.md)
- [部署指南](docs/DEPLOYMENT_CHECKLIST.md)
- [CI/CD 文档](.github/workflows/README.md)
- [快速启动](.github/workflows/QUICKSTART.md)
```

---

## 自定义徽章样式

### 使用 shields.io 自定义

```markdown
<!-- 自定义样式 -->
![CI/CD](https://img.shields.io/github/actions/workflow/status/FangcunMount/iam-contracts/cicd.yml?style=flat-square&label=CI%2FCD)

<!-- 不同样式选项 -->
?style=flat          # 默认平面样式
?style=flat-square   # 平面方形样式
?style=plastic       # 塑料样式
?style=for-the-badge # 大徽章样式
?style=social        # 社交样式
```

### 添加颜色

```markdown
![Status](https://img.shields.io/badge/status-active-success?style=flat-square)
![Environment](https://img.shields.io/badge/environment-production-blue?style=flat-square)
```

---

## 组合徽章示例

```markdown
## 项目状态

![Build](https://github.com/FangcunMount/iam-contracts/actions/workflows/cicd.yml/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)
```

---

## 更新 README.md

将徽章添加到项目根目录的 README.md 文件顶部，让访问者一眼就能看到项目状态。

**注意：** 将上述示例中的 `FangcunMount` 和 `iam-contracts` 替换为你的实际 GitHub 用户名/组织名和仓库名。
