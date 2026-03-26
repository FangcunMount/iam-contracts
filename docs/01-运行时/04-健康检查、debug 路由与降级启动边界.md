# 健康检查、debug 路由与降级启动边界

本文回答：`iam-contracts` 今天有哪些健康检查和 debug 暴露面，哪些端点不依赖完整业务模块，以及 MySQL / Redis / EventBus / 容器初始化失败时进程到底会不会继续启动。

## 30 秒结论

- 当前运行时有两套健康检查面：HTTP 基础路由上的 `/health`、`/ping`，以及 gRPC 侧额外暴露的 `grpc.health.v1.Health` + 独立 HTTP `/healthz` / `/readyz` / `/livez`。
- `debug/routes` 和 `debug/modules` 是中央 router 直接注册的基础调试面；它们不依赖业务域模块成功初始化。
- `PrepareRun()` 对 MySQL、Redis、EventBus、容器初始化失败大多只记 `warn`，然后继续注册 HTTP 路由和 gRPC 服务。
- 这意味着进程可以以“部分能力可用”的状态启动；“进程活着”不等于“业务模块都已完整可用”。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| HTTP 基础健康检查 | `/health`、`/ping` | [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |
| gRPC 健康检查 | `grpc.health.v1.Health` | [../../internal/pkg/grpc/server.go](../../internal/pkg/grpc/server.go) |
| 独立 HTTP 探针 | `/healthz`、`/readyz`、`/livez` | [../../internal/pkg/grpc/server.go](../../internal/pkg/grpc/server.go) |
| debug 路由 | `/debug/routes`、`/debug/modules` | [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |
| 降级启动入口 | `PrepareRun()` 里多处 `Warnf(...); continue` | [../../internal/apiserver/server.go](../../internal/apiserver/server.go) |
| gRPC 降级注册 | 容器或 gRPC server 为空时跳过注册 | [../../internal/apiserver/server.go](../../internal/apiserver/server.go) |

## 1. 当前有哪些健康检查面

### 1.1 HTTP 基础路由

中央 router 在业务模块之前就注册了这些基础端点：

- `/health`
- `/ping`
- `/debug/routes`
- `/debug/modules`
- `/openapi/*`
- `/swagger/*`
- `/api/v1/public/info`

这意味着即便后面的模块路由没有完整注册，基础健康面和调试面仍然可能存在。

### 1.2 gRPC 自带健康检查

如果 `EnableHealthCheck` 打开，`internal/pkg/grpc/server.go` 会：

1. 注册 `grpc.health.v1.Health`
2. 在独立 `healthz-port` 上起一个 HTTP server
3. 暴露 `/healthz`、`/readyz`、`/livez`

其中：

- `/healthz` 直接查 gRPC health 状态
- `/readyz` 只有在整体状态为 `SERVING` 时才返回 `READY`
- `/livez` 只表示进程活着，不代表业务模块已经完整初始化

## 2. debug 路由今天能看什么

### 2.1 `/debug/routes`

这个端点遍历 `gin.Engine.Routes()`，返回当前进程里已经注册的所有 HTTP 路由。它适合回答：

- 某个 REST 路由今天到底有没有挂进进程
- 基础路由和模块路由当前各有哪些

### 2.2 `/debug/modules`

这个端点只回答“容器和模块有没有初始化出来”，不会证明业务逻辑一定可用。当前它主要返回：

- `container_initialized`
- `modules.authn`
- `modules.authz`
- `modules.user`
- `modules.idp`

因此它更像运行时状态快照，而不是业务就绪证明。

## 3. 什么叫“降级启动”

当前 `PrepareRun()` 的口径不是“任一依赖失败就终止进程”，而是更接近：

1. 先尽量初始化数据库、Redis、EventBus、容器
2. 某一步失败时记录 `warn`
3. 继续创建 router、注册 HTTP 路由、尝试注册 gRPC 服务
4. 最终仍然启动 HTTP 和 gRPC server

### 3.1 当前会被降级处理的初始化失败

| 初始化项 | 当前行为 |
| ---- | ---- |
| `dbManager.Initialize()` 失败 | 记录 warning，继续 |
| 取 MySQL 连接失败 | 记 warning，`mysqlDB = nil`，继续 |
| 取 Redis 失败 | 记 warning，`cacheClient = nil`，继续 |
| 创建 EventBus 失败 | 记 warning，`eventBus = nil`，继续 |
| `container.Initialize()` 失败 | 记 warning，继续 |

### 3.2 对运行面的直接影响

| 影响面 | 当前行为 |
| ---- | ---- |
| HTTP 基础路由 | 会继续注册 |
| 模块路由 | 取决于容器和各模块是否已初始化 |
| gRPC 服务注册 | `grpcServer` 或 `container` 为空时直接跳过 |
| 进程启动 | HTTP / gRPC 仍会尝试启动 |

## 4. 最容易讲错的边界

### 已实现

- 有多种健康检查入口
- 有基础 debug 路由
- 运行时支持在依赖不完整时继续启动

### 待补证据

- 各模块在“部分初始化”状态下，哪些 handler 真正还能稳定工作，还需要继续逐模块核对

### 规划改造

- 如果未来希望把“进程启动成功”和“业务完全就绪”严格分离，应补更明确的 readiness 语义，而不是只依赖当前这组基础探针

## 5. 继续往下读

1. [01-服务入口、HTTP 与模块装配.md](./01-服务入口、HTTP 与模块装配.md)
2. [02-gRPC与mTLS.md](./02-gRPC与mTLS.md)
3. [03-HTTP认证中间件与身份上下文.md](./03-HTTP认证中间件与身份上下文.md)
4. [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)
