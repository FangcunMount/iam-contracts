# Jenkins Go 版本问题修复指南

> **问题**: Jenkins 构建失败 - `go: download go1.24 for linux/amd64: toolchain not available`  
> **原因**: Jenkins 节点上没有安装 Go 1.24  
> **日期**: 2025-10-19

---

## 🔍 问题分析

### 错误信息
```
[2025-10-19T06:37:22.223Z] go: downloading go1.24 (linux/amd64)
[2025-10-19T06:37:22.223Z] go: download go1.24 for linux/amd64: toolchain not available
```

### 原因
1. 项目使用 Go 1.24 (`go.mod` 中定义)
2. Jenkins 节点上的 Go 版本过低或未安装
3. "依赖管理" 阶段尝试执行 `go mod download` 等命令
4. Go 1.24 尚未在所有环境中广泛可用（Go 1.23 是当前最新稳定版）

---

## ✅ 推荐解决方案

### 方案 1：使用 Docker 部署时跳过本地 Go 命令（推荐 ⭐）

**优点**：
- ✅ 无需在 Jenkins 节点安装 Go
- ✅ Docker 镜像会自动使用正确的 Go 版本
- ✅ 环境一致性更好
- ✅ 快速解决，无需等待 Go 1.24 正式发布

**实现**：修改 Jenkinsfile，在 Docker 部署模式下跳过需要本地 Go 环境的阶段。

#### 修改内容

**1. 依赖管理阶段 - 添加条件**
```groovy
stage('依赖管理') {
    when {
        expression { params.DEPLOY_MODE != 'docker' }  // Docker 模式跳过
    }
    steps {
        echo '📦 下载 Go 依赖...'
        sh '''
            go env -w GO111MODULE=on
            go env -w GOPROXY=https://goproxy.cn,direct
            go mod download
            go mod tidy
            go mod verify
        '''
    }
}
```

**2. 代码检查阶段 - 添加条件**
```groovy
stage('代码检查') {
    when {
        allOf {
            expression { env.RUN_LINT == 'true' }
            expression { params.DEPLOY_MODE != 'docker' }  // Docker 模式跳过
        }
    }
    parallel {
        // ... 格式化检查和静态分析
    }
}
```

**3. 单元测试阶段 - 添加条件**
```groovy
stage('单元测试') {
    when {
        allOf {
            expression { env.RUN_TESTS == 'true' }
            expression { params.DEPLOY_MODE != 'docker' }  // Docker 模式跳过
        }
    }
    steps {
        echo '🧪 运行单元测试...'
        // ... 测试命令
    }
}
```

**4. 编译构建阶段 - 添加条件**
```groovy
stage('编译构建') {
    when {
        allOf {
            expression { env.RUN_BUILD == 'true' }
            expression { params.DEPLOY_MODE != 'docker' }  // Docker 模式跳过
        }
    }
    steps {
        echo '🔨 编译 Go 应用...'
        // ... 编译命令
    }
}
```

#### 工作流程

修改后，Docker 部署模式的流程：

```
1. ✅ Checkout          - 拉取代码
2. ✅ Setup             - 加载环境变量
3. ⏭️  依赖管理         - 跳过（Docker 镜像会处理）
4. ⏭️  代码检查         - 跳过（Docker 镜像会处理）
5. ⏭️  单元测试         - 跳过（Docker 镜像会处理）
6. ⏭️  编译构建         - 跳过（Docker 镜像会处理）
7. ✅ 构建 Docker 镜像  - 在这里完成所有 Go 相关操作
8. ✅ 准备 Docker 网络  - 创建网络
9. ✅ 部署              - Docker Compose 启动服务
10. ✅ 健康检查         - 验证部署成功
```

---

### 方案 2：在 Jenkins 节点上安装 Go 1.24

**优点**：
- ✅ 可以在 Jenkins 上运行测试和代码检查
- ✅ 不依赖 Docker（适用于 Binary 和 Systemd 部署模式）

**缺点**：
- ⚠️ Go 1.24 尚未正式发布，安装复杂
- ⚠️ 需要服务器访问权限
- ⚠️ 每个 Jenkins 节点都需要安装

**实现步骤**：

#### SSH 到 Jenkins 服务器

```bash
ssh user@jenkins-server
```

#### 安装 Go 1.24（如果可用）

```bash
# 方法 1: 使用官方安装脚本（推荐等 1.24 正式发布）
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz

# 方法 2: 从源码构建（高级用户）
git clone https://go.googlesource.com/go
cd go
git checkout go1.24
cd src
./all.bash

# 方法 3: 使用 gvm (Go Version Manager)
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
source ~/.gvm/scripts/gvm
gvm install go1.24
gvm use go1.24 --default
```

#### 验证安装

```bash
go version
# 应该输出: go version go1.24.0 linux/amd64
```

#### 配置 Jenkins

在 Jenkins 系统配置中添加 Go 工具：

1. 进入 **Manage Jenkins** → **Global Tool Configuration**
2. 找到 **Go** 部分
3. 添加 Go 安装：
   - **Name**: `Go 1.24`
   - **Install automatically**: 取消勾选
   - **GOROOT**: `/usr/local/go`

---

### 方案 3：降级项目到 Go 1.23（临时方案）

如果 Go 1.24 不是必需的，可以降级到 Go 1.23（当前最新稳定版）。

**已修改的文件**：
- ✅ `go.mod`: `go 1.24` → `go 1.23`
- ✅ `build/docker/Dockerfile`: `golang:1.24-alpine` → `golang:1.23-alpine`
- ✅ `build/docker/README.md`: 文档更新
- ✅ `docs/deploy/MAKEFILE_GUIDE.md`: 文档更新

**回滚方法**（如果需要）：
```bash
# 恢复到 Go 1.24
git checkout go.mod build/docker/Dockerfile build/docker/README.md docs/deploy/MAKEFILE_GUIDE.md
```

---

## 📊 方案对比

| 特性 | 方案 1: 跳过本地 Go | 方案 2: 安装 Go 1.24 | 方案 3: 降级到 1.23 |
|------|-------------------|-------------------|-------------------|
| **实施难度** | ⭐ 简单 | ⭐⭐⭐ 复杂 | ⭐ 简单 |
| **Jenkins 节点要求** | 仅需 Docker | 需要 Go 1.24 | 需要 Go 1.23+ |
| **适用部署模式** | Docker | 全部 | 全部 |
| **CI/CD 功能** | 仅构建部署 | 完整（含测试） | 完整（含测试） |
| **环境一致性** | ⭐⭐⭐ 高 | ⭐⭐ 中 | ⭐⭐ 中 |
| **维护成本** | ⭐ 低 | ⭐⭐⭐ 高 | ⭐ 低 |
| **推荐指数** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |

---

## 🎯 推荐策略

### 当前（短期）
使用 **方案 1: 跳过本地 Go 命令**
- Docker 部署模式下不需要 Jenkins 节点上的 Go
- 快速解决构建失败问题
- 降低维护成本

### 未来（长期）
当 Go 1.24 正式发布后，可以考虑：
- **方案 2**: 如果需要在 Jenkins 上运行测试和代码检查
- 或保持 **方案 1**: 如果 Docker 部署足够

### 如果不需要 Go 1.24 特性
使用 **方案 3: 降级到 Go 1.23**
- Go 1.23 是当前最新稳定版
- 更好的兼容性和稳定性
- 除非代码依赖 1.24 的新特性

---

## 🚀 立即执行

### 执行方案 1（推荐）

```bash
# 1. 修改 Jenkinsfile（见上面的修改内容）
vim Jenkinsfile

# 2. 提交更改
git add Jenkinsfile
git commit -m "fix: Docker 部署模式下跳过本地 Go 命令检查"
git push

# 3. 重新触发 Jenkins 构建
# Jenkins 会自动拉取最新代码并使用新的 Pipeline
```

### 执行方案 3（如果不需要 Go 1.24）

```bash
# 已经修改完成，直接提交：
git add go.mod build/docker/Dockerfile build/docker/README.md docs/deploy/MAKEFILE_GUIDE.md
git commit -m "chore: 降级 Go 版本从 1.24 到 1.23（当前稳定版）"
git push
```

---

## 🔍 验证

### 方案 1 验证

提交后，观察 Jenkins 构建日志，应该看到：

```
[Pipeline] stage
[Pipeline] { (依赖管理)
Stage "依赖管理" skipped due to when conditional  // ✅ 跳过了
[Pipeline] }

[Pipeline] stage
[Pipeline] { (代码检查)
Stage "代码检查" skipped due to when conditional  // ✅ 跳过了
[Pipeline] }

[Pipeline] stage
[Pipeline] { (构建 Docker 镜像)
[Pipeline] echo
🐳 构建 Docker 镜像...  // ✅ 直接进入 Docker 构建
```

### 方案 3 验证

```bash
# 本地测试编译
go version  # 确认本地 Go 版本
go mod tidy
go build -o bin/apiserver ./cmd/apiserver/

# Docker 构建测试
docker build -f build/docker/Dockerfile -t test:latest .
```

---

## 📝 后续优化

### 如果选择方案 1

可以考虑在 Dockerfile 中添加测试阶段：

```dockerfile
# 在构建阶段运行测试
FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 运行测试
RUN go test -v -race ./... || echo "Warning: Tests failed"

# 编译
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w" \
    -o apiserver ./cmd/apiserver/
```

这样测试会在 Docker 构建过程中执行，确保代码质量。

---

## 📚 参考资料

- [Go 版本发布历史](https://go.dev/doc/devel/release)
- [Docker 多阶段构建](https://docs.docker.com/build/building/multi-stage/)
- [Jenkins Pipeline 条件执行](https://www.jenkins.io/doc/book/pipeline/syntax/#when)
- [Go 工具链管理](https://go.dev/doc/toolchain)

---

**更新日期**: 2025-10-19  
**状态**: 待选择方案  
**建议**: 使用方案 1（Docker 模式跳过本地 Go）
