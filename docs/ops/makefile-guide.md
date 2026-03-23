# Makefile 使用指南

本项目使用 Makefile 来简化常见的开发、测试和部署任务。

## 快速开始

```bash
# 显示所有可用命令
make help

# 构建项目
make build

# 运行服务
make run

# 停止服务
make stop

# 查看服务状态
make status
```

## 命令分类

### 📦 构建命令

| 命令 | 说明 |
| ------ | ------ |
| `make build` | 构建所有服务 |
| `make build-apiserver` | 构建 API 服务器 |
| `make clean` | 清理构建文件和临时文件 |

### 🚀 服务管理

| 命令 | 说明 |
| ------ | ------ |
| `make run` | 启动所有服务 |
| `make run-apiserver` | 启动 API 服务器 |
| `make stop` | 停止所有服务 |
| `make stop-apiserver` | 停止 API 服务器 |
| `make restart` | 重启所有服务 |
| `make restart-apiserver` | 重启 API 服务器 |
| `make status` | 查看所有服务状态 |
| `make status-apiserver` | 查看 API 服务器状态 |
| `make logs` | 查看所有服务日志 |
| `make logs-apiserver` | 查看 API 服务器日志（实时） |
| `make health` | 健康检查所有服务 |

### 🛠️ 开发工具

| 命令 | 说明 |
| ------ | ------ |
| `make dev` | 启动开发环境（热更新） |
| `make dev-apiserver` | 独立启动 API 服务器开发环境 |
| `make dev-stop` | 停止开发环境 |
| `make dev-status` | 查看开发环境状态 |

**开发环境特点**：

- 使用 [Air](https://github.com/air-verse/air) 实现热更新
- 代码变更后自动重新编译和重启
- 适合本地开发和调试

### 🧪 测试命令

| 命令 | 说明 |
| ------ | ------ |
| `make test` | 运行所有测试 |
| `make test-unit` | 运行单元测试 |
| `make test-integration` | 运行集成测试 |
| `make test-coverage` | 生成测试覆盖率报告 |
| `make test-race` | 运行竞态检测测试 |
| `make test-bench` | 运行基准测试 |

**测试覆盖率**：

```bash
make test-coverage
# 覆盖率报告生成在: coverage/coverage.html
```

### ✨ 代码质量

| 命令 | 说明 |
| ------ | ------ |
| `make lint` | 运行代码检查 |
| `make fmt` | 格式化代码 |
| `make fmt-check` | 检查代码格式（CI 使用） |

**Lint 工具**：

- 优先使用 `golangci-lint`（如果已安装）
- 否则退回到 `go vet`

### 📦 依赖管理

| 命令 | 说明 |
| ------ | ------ |
| `make deps` | 下载所有依赖 |
| `make deps-download` | 下载依赖 |
| `make deps-tidy` | 整理依赖 |
| `make deps-verify` | 验证依赖 |

### 🔧 Protocol Buffers

| 命令 | 说明 |
| ------ | ------ |
| `make proto-gen` | 生成 protobuf 代码 |

**前提条件**：

- 需要 `scripts/proto/generate.sh` 脚本存在

### 🔍 调试和诊断

| 命令 | 说明 |
| ------ | ------ |
| `make version` | 显示版本信息 |
| `make debug` | 显示调试信息 |
| `make ps` | 显示相关进程 |
| `make ports` | 检查端口占用 |

### 🚀 CI/CD

| 命令 | 说明 |
| ------ | ------ |
| `make ci` | 运行 CI 流程（验证、格式检查、Lint、测试） |
| `make release` | 发布版本（清理、构建） |

### ⚡ 快捷命令

| 命令 | 等同于 | 说明 |
| ------ | -------- | ------ |
| `make up` | `make run` | 启动服务 |
| `make down` | `make stop` | 停止服务 |
| `make re` | `make restart` | 重启服务 |
| `make st` | `make status` | 查看状态 |
| `make log` | `make logs` | 查看日志 |

## 工作流示例

### 日常开发流程

```bash
# 1. 拉取最新代码
git pull

# 2. 更新依赖
make deps

# 3. 启动开发环境（热更新）
make dev

# 4. 开发过程中...
# 代码会自动重新编译和重启

# 5. 格式化代码
make fmt

# 6. 运行测试
make test

# 7. 提交代码前检查
make ci
```

### 生产部署流程

```bash
# 1. 清理旧的构建
make clean

# 2. 拉取最新代码
git pull

# 3. 验证依赖
make deps-verify

# 4. 运行测试
make test

# 5. 构建服务
make build

# 6. 停止旧服务
make stop

# 7. 启动新服务
make run

# 8. 检查服务状态
make status

# 9. 查看日志
make logs
```

### 快速重启服务

```bash
# 方式 1：使用 restart 命令
make restart

# 方式 2：使用快捷命令
make re

# 方式 3：手动停止和启动
make stop && make run
```

### 问题排查

```bash
# 1. 查看服务状态
make status

# 2. 查看日志
make logs

# 3. 检查进程
make ps

# 4. 检查端口
make ports

# 5. 查看调试信息
make debug
```

## 环境变量

Makefile 使用以下变量，可以通过环境变量覆盖：

| 变量 | 默认值 | 说明 |
| ------ | -------- | ------ |
| `APISERVER_BIN` | `bin/apiserver` | API 服务器二进制文件路径 |
| `APISERVER_CONFIG` | `configs/apiserver-simple.yaml` | API 服务器配置文件 |
| `APISERVER_PORT` | `8080` | API 服务器端口 |
| `PID_DIR` | `tmp/pids` | PID 文件目录 |
| `LOG_DIR` | `logs` | 日志文件目录 |
| `COVERAGE_DIR` | `coverage` | 覆盖率报告目录 |

**示例**：

```bash
# 使用不同的配置文件
make run APISERVER_CONFIG=configs/apiserver.prod.yaml

# 使用不同的端口
make run APISERVER_PORT=9090
```

## 工具安装

### 安装开发工具

```bash
make install-tools
```

这将安装：

- [Air](https://github.com/air-verse/air) - 热更新工具
- [mockgen](https://github.com/uber-go/mock) - Mock 生成工具

### 手动安装 golangci-lint

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.2
```

## 目录结构

Makefile 会自动创建以下目录：

```text
.
├── bin/              # 编译后的二进制文件
│   └── apiserver
├── logs/             # 日志文件
│   └── apiserver.log
├── tmp/              # 临时文件
│   └── pids/         # PID 文件
│       └── apiserver.pid
└── coverage/         # 测试覆盖率报告
    ├── coverage.out
    └── coverage.html
```

## 常见问题

### Q: 服务启动失败怎么办？

```bash
# 1. 查看日志
make logs

# 2. 检查端口是否被占用
make ports

# 3. 查看进程
make ps

# 4. 手动清理
make stop
make clean
make build
make run
```

### Q: 如何查看实时日志？

```bash
# API 服务器日志
make logs-apiserver

# 或直接使用 tail
tail -f logs/apiserver.log
```

### Q: 如何强制重新构建？

```bash
# 清理后重新构建
make clean
make build
```

### Q: 开发环境热更新不生效？

```bash
# 1. 确保 Air 已安装
make install-tools

# 2. 检查 .air-apiserver.toml 配置文件

# 3. 重启开发环境
make dev-stop
make dev
```

### Q: 如何在 CI 中使用？

```bash
# GitHub Actions 示例
steps:
  - name: Checkout
    uses: actions/checkout@v4
  
  - name: Setup Go
    uses: actions/setup-go@v5
    with:
      go-version: '1.24'
  
  - name: Run CI
    run: make ci
```

## 提示和技巧

### 查看命令详情而不执行

```bash
# 使用 -n 参数
make build -n
```

### 并行执行多个命令

```bash
# 同时格式化和测试
make fmt & make test & wait
```

### 自定义构建标志

```bash
# 添加编译标签
make build GO_LDFLAGS="-ldflags '-X main.Version=v1.0.0'"
```

## 版本信息

查看项目版本信息：

```bash
make version
```

输出示例：

```text
版本信息:
  版本:     v1.0.0-5-g2ab78ae-dirty
  构建时间: 2025-10-18_13:44:21
  Git 提交: 2ab78ae
  Git 分支: main
  Go 版本:  go version go1.23.0 darwin/arm64
```

## 参考资料

- [GNU Make 文档](https://www.gnu.org/software/make/manual/)
- [Go 命令文档](https://golang.org/cmd/go/)
- [Air 文档](https://github.com/air-verse/air)
- [golangci-lint 文档](https://golangci-lint.run/)

---

**最后更新**: 2025-10-18  
**维护团队**: IAM Team
