# Air 热重载开发环境配置

## 简介

本项目使用 [Air](https://github.com/cosmtrek/air) 作为 Go 应用的热重载工具，在开发时自动检测文件变化并重新编译、重启服务。

## 安装 Air

如果还未安装 Air，请执行：

```bash
# 使用 go install
go install github.com/cosmtrek/air@latest

# 或使用 curl (macOS/Linux)
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

## 配置文件说明

### `.air-apiserver.toml`

API Server 的 Air 配置文件，主要配置项：

- **工作目录**: 项目根目录
- **编译命令**: `go build -o ./tmp/apiserver ./cmd/apiserver/apiserver.go`
- **运行参数**: `-c configs/apiserver.yaml` (使用开发环境配置)
- **监听目录**: `cmd`, `internal`, `pkg`, `configs`
- **监听扩展**: `.go`, `.yaml`, `.yml`, `.toml`, `.json`
- **排除目录**: `tmp`, `vendor`, `bin`, `logs`, `docs`, `scripts`, `api`
- **排除文件**: `*_test.go`, `*_gen.go`
- **重建延迟**: 1000ms (防止频繁重新构建)

## 使用方法

### 启动开发环境

```bash
# 启动所有服务（热重载模式）
make dev

# 或仅启动 API Server
make dev-apiserver
```

### 停止开发环境

```bash
# 停止所有服务
make dev-stop

# 或直接 Ctrl+C
```

## 工作原理

1. **文件监听**: Air 监听指定目录下的文件变化
2. **自动编译**: 检测到变化后，自动执行 `go build` 编译
3. **进程管理**: 
   - 停止旧的进程
   - 启动新编译的二进制文件
   - 传递配置参数
4. **日志输出**: 在控制台显示编译和运行日志

## 目录结构

```
iam-contracts/
├── .air-apiserver.toml    # Air 配置文件
├── tmp/                   # 临时文件目录（已加入 .gitignore）
│   ├── apiserver          # 编译后的二进制文件
│   ├── air.log            # Air 日志
│   └── pids/              # 进程 PID 文件
│       └── air-apiserver.pid
├── cmd/apiserver/         # API Server 入口
├── configs/               # 配置文件
│   └── apiserver.yaml     # 开发环境配置
└── internal/              # 内部代码
```

## 注意事项

### 1. 配置文件优先级

Air 运行时会读取 `configs/apiserver.yaml` 配置，确保：
- 数据库连接信息正确
- Redis 连接信息正确
- 端口没有被占用

### 2. 性能优化

- **延迟重建**: 配置了 1000ms 延迟，避免频繁保存导致多次重建
- **排除测试文件**: 不监听 `*_test.go`，测试文件变化不会触发重建
- **排除文档**: 不监听 `docs/`, `scripts/` 等目录

### 3. 故障排查

#### 端口被占用
```bash
# 查看端口占用
lsof -i :8080

# 杀死占用进程
kill -9 <PID>
```

#### Air 进程残留
```bash
# 查看 Air 进程
ps aux | grep air

# 清理临时文件
rm -rf tmp/*
```

#### 编译失败
查看 `tmp/air.log` 获取详细编译错误信息。

## 日志配置

Air 日志分为几个部分：
- **主日志** (洋红色): Air 主进程日志
- **监听器** (青色): 文件监听日志
- **构建** (黄色): 编译过程日志
- **运行器** (绿色): 应用启动日志
- **应用** (白色): 应用程序输出

## 最佳实践

### 开发流程

1. **启动服务**: `make dev`
2. **修改代码**: 编辑 Go 源文件
3. **自动重载**: Air 自动检测变化，重新编译并重启
4. **查看日志**: 控制台实时显示应用日志
5. **停止服务**: `Ctrl+C` 或 `make dev-stop`

### 配置调整

修改 `configs/apiserver.yaml` 后也会触发重载，例如：
- 修改端口
- 修改数据库连接
- 修改日志级别

### 多服务开发

如需添加其他服务（如 authz-server），可以：
1. 复制 `.air-apiserver.toml` 为 `.air-authzserver.toml`
2. 修改编译命令和运行参数
3. 在 Makefile 中添加对应的 `dev-authzserver` 目标

## 参考资源

- [Air GitHub](https://github.com/cosmtrek/air)
- [Air 配置文档](https://github.com/cosmtrek/air/blob/master/README-zh_cn.md)
- [项目 Makefile](../Makefile)
