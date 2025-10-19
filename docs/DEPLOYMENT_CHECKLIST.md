# Jenkins 部署检查清单

> **更新日期**: 2025-10-19  
> **Commit**: 66bd119  
> **状态**: ✅ 准备就绪

---

## ✅ 已完成的修复

### 1. ✅ Go 版本兼容性问题
**问题**: Jenkins 节点没有 Go 1.24  
**解决方案**:
- ✅ 降级到 Go 1.23（当前稳定版）
- ✅ Docker 部署模式跳过本地 Go 环境检查
- ✅ 所有 Go 操作在 Docker 镜像内完成

**修改文件**:
- `go.mod`: `go 1.24` → `go 1.23`
- `build/docker/Dockerfile`: `golang:1.24-alpine` → `golang:1.23-alpine`
- `Jenkinsfile`: 添加 Docker 模式条件判断

### 2. ✅ Jenkinsfile Git 仓库问题
**问题**: Git 命令在 environment 块执行失败  
**解决**: 移动到 Checkout 阶段执行

### 3. ✅ Jenkinsfile Post 块安全性
**问题**: 变量未定义导致 post 块失败  
**解决**: 添加 try-catch 和默认值

### 4. ✅ 环境变量加载
**问题**: Jenkins 凭据未配置  
**解决**: 从 Jenkins 凭据加载（已在 Setup 阶段成功）

### 5. ✅ 数据库配置
**问题**: 数据库名称、密码、初始化脚本错误  
**解决**: 
- 所有配置统一为 `iam_contracts`
- SQL 脚本完整重写匹配代码
- Docker Compose 配置正确

### 6. ✅ Nginx 配置
**问题**: 缺少 Nginx 服务  
**解决**: Docker Compose 添加 Nginx 服务

### 7. ✅ JWT 配置
**问题**: 缺少 JWT_SECRET  
**解决**: 在环境变量中配置

---

## 🚀 Jenkins 部署流程预期

### Docker 部署模式流程

```
┌─────────────────────────────────────────────────────────────────┐
│ Stage 1: Checkout                                               │
│ ✅ 拉取代码                                                      │
│ ✅ 获取 Git 信息 (commit, branch, time)                         │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 2: Setup                                                  │
│ ✅ 从 Jenkins 凭据加载环境变量                                   │
│ ✅ 设置构建参数                                                  │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 3: 依赖管理                                               │
│ ⏭️  SKIPPED (Docker 模式)                                       │
│ 原因: Docker 镜像会自动处理 Go 依赖                              │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 4: 代码检查                                               │
│ ⏭️  SKIPPED (Docker 模式)                                       │
│ 原因: 无需 Jenkins 节点上的 Go 环境                             │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 5: 单元测试                                               │
│ ⏭️  SKIPPED (Docker 模式)                                       │
│ 原因: 无需 Jenkins 节点上的 Go 环境                             │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 6: 编译构建                                               │
│ ⏭️  SKIPPED (Docker 模式)                                       │
│ 原因: Docker 镜像会编译二进制文件                                │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 7: 构建 Docker 镜像                                        │
│ ✅ 使用 golang:1.23-alpine 构建                                  │
│ ✅ 多阶段构建（构建阶段 + 运行阶段）                             │
│ ✅ 生成镜像: iam-contracts/iam-contracts:prod                    │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 8: 准备 Docker 网络                                        │
│ ✅ 创建网络: iam-network                                         │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 9: 推送镜像                                               │
│ ⏭️  SKIPPED (PUSH_IMAGE=false)                                  │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 10: 数据库初始化                                          │
│ ⏭️  SKIPPED (INITIALIZE_DATABASE=false)                         │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 11: 数据库迁移                                            │
│ ⏭️  SKIPPED (RUN_MIGRATION=false)                               │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 12: 加载种子数据                                          │
│ ⏭️  SKIPPED (LOAD_SEED_DATA=false)                              │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 13: 部署                                                  │
│ ✅ Docker Compose 启动服务                                       │
│   - MySQL 8.0                                                   │
│   - Redis 7                                                     │
│   - IAM API Server                                              │
│   - Nginx 反向代理                                              │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Stage 14: 健康检查                                              │
│ ✅ 检查 MySQL 就绪                                               │
│ ✅ 检查 Redis 就绪                                               │
│ ✅ 检查 API Server 健康 (/healthz)                              │
│ ✅ 检查 Nginx 可访问                                            │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Post Actions                                                    │
│ ✅ 清理临时文件                                                  │
│ ✅ 显示部署结果                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔍 关键检查点

### Checkout 阶段
```groovy
预期输出:
✅ Cloning the remote Git repository
✅ Cloning repository https://github.com/FangcunMount/iam-contracts
✅ Checking out Revision 66bd119... (refs/remotes/origin/main)
✅ Commit message: "fix: Docker 部署模式跳过本地 Go 环境检查 + 降级到 Go 1.23"
```

### Setup 阶段
```groovy
预期输出:
✅ 从凭据 'iam-contracts-prod-env' 加载环境变量
✅ 部署模式: docker
✅ 镜像标签: prod
✅ 运行构建: true
✅ Docker构建: true
✅ 执行部署: true
✅ 健康检查: true
```

### 依赖管理阶段（应该跳过）
```groovy
预期输出:
⏭️  Stage "依赖管理" skipped due to when conditional
```

### 代码检查阶段（应该跳过）
```groovy
预期输出:
⏭️  Stage "代码检查" skipped due to when conditional
```

### 单元测试阶段（应该跳过）
```groovy
预期输出:
⏭️  Stage "单元测试" skipped due to when conditional
```

### 构建 Docker 镜像阶段
```groovy
预期输出:
✅ 🐳 构建 Docker 镜像...
✅ Step 1/X : FROM golang:1.23-alpine AS builder  ← 注意这里是 1.23
✅ Step X/X : CMD ["/app/apiserver"]
✅ Successfully built <image-id>
✅ Successfully tagged iam-contracts/iam-contracts:prod
```

### 部署阶段
```groovy
预期输出:
✅ 🚀 部署应用...
✅ Creating network "iam-network" with the default driver
✅ Creating iam-mysql ... done
✅ Creating iam-redis ... done
✅ Creating iam-apiserver ... done
✅ Creating iam-nginx ... done
```

### 健康检查阶段
```groovy
预期输出:
✅ MySQL is ready
✅ Redis is ready
✅ API Server is healthy
✅ Nginx is accessible
```

---

## ⚠️ 可能的问题和解决方案

### 问题 1: Docker 镜像构建失败 - Go 模块下载超时

**症状**:
```
error: failed to fetch module
```

**解决**:
```bash
# 在 Dockerfile 中已配置 GOPROXY
ENV GOPROXY=https://goproxy.cn,direct
```

### 问题 2: MySQL 连接失败

**症状**:
```
Error: dial tcp: connect: connection refused
```

**检查**:
```bash
# 1. 检查 MySQL 容器是否运行
docker ps | grep mysql

# 2. 检查环境变量是否正确
docker exec iam-apiserver env | grep MYSQL

# 3. 检查 MySQL 日志
docker logs iam-mysql
```

**解决**: 环境变量已在 Jenkins 凭据中正确配置

### 问题 3: Nginx 502 Bad Gateway

**症状**:
```
502 Bad Gateway
```

**原因**: API Server 未就绪

**检查**:
```bash
# 检查 API Server 状态
docker logs iam-apiserver

# 检查健康端点
curl http://localhost:8080/healthz
```

**解决**: 健康检查阶段会验证

---

## 📝 部署后验证

### 1. 检查所有容器运行状态

```bash
docker ps

# 预期输出:
# iam-nginx       - Up, 80/tcp, 443/tcp
# iam-apiserver   - Up, 8080/tcp
# iam-mysql       - Up, 3306/tcp
# iam-redis       - Up, 6379/tcp
```

### 2. 检查网络连接

```bash
docker network inspect iam-network

# 应该包含所有 4 个容器
```

### 3. 测试 API 端点

```bash
# 通过 Nginx（生产方式）
curl -k https://iam.yangshujie.com/healthz

# 直接访问 API Server（调试用）
curl http://localhost:8080/healthz

# 预期响应:
# {"status":"ok"}
```

### 4. 检查数据库连接

```bash
# 登录 MySQL
docker exec -it iam-mysql mysql -uiam -p2gy0dCwG iam_contracts

# 查看表
mysql> SHOW TABLES;

# 预期输出:
# +-------------------------+
# | Tables_in_iam_contracts |
# +-------------------------+
# | children                |
# | guardianships           |
# | resources               |
# | role_resources          |
# | roles                   |
# | user_roles              |
# | users                   |
# +-------------------------+
```

### 5. 检查 Redis 连接

```bash
# 连接 Redis
docker exec -it iam-redis redis-cli -a 68OTeDXq

# 测试命令
127.0.0.1:6379> PING
# 预期响应: PONG
```

---

## 🎯 下次部署准备

### 如果需要数据库初始化（首次部署）

Jenkins 构建参数修改：
- ✅ 勾选 `INITIALIZE_DATABASE`
- ✅ 勾选 `LOAD_SEED_DATA`（如果需要测试数据）

### 如果需要数据库迁移（版本升级）

Jenkins 构建参数修改：
- ✅ 勾选 `RUN_MIGRATION`

### 如果需要推送镜像到仓库

Jenkins 构建参数修改：
- ✅ 勾选 `PUSH_IMAGE`
- 配置 `IMAGE_REGISTRY` 参数

---

## 📊 性能预期

### 构建时间
- **首次构建**: ~5-8 分钟（下载依赖和基础镜像）
- **增量构建**: ~2-4 分钟（Docker 缓存）

### 部署时间
- **停止旧服务**: ~10 秒
- **启动新服务**: ~30 秒
- **健康检查**: ~20 秒
- **总计**: ~1 分钟

---

## ✅ 部署成功标志

看到以下输出即表示部署成功：

```
================================================
✅ 部署成功
================================================
项目: iam-contracts
分支: main
构建: #4
提交: 66bd119
================================================
部署地址:
  - API:   http://localhost:8080
  - Nginx: https://iam.yangshujie.com
健康检查: ✅ 通过
================================================
```

---

## 🔗 相关文档

- [Go 版本问题修复指南](./JENKINS_GO_VERSION_FIX.md)
- [Jenkins 部署错误修复记录](./JENKINSFILE_ERROR_FIX.md)
- [Makefile 使用指南](./deploy/MAKEFILE_GUIDE.md)
- [Docker 部署说明](../build/docker/README.md)

---

**准备状态**: ✅ 可以部署  
**最后更新**: 2025-10-19  
**下一步**: 在 Jenkins 中触发构建
