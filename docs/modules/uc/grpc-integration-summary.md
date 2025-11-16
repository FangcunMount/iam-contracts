# UC 模块 gRPC 服务集成总结

## 概述

本文档记录了 UC (User Center) 模块的 gRPC 服务实现和集成过程。

## 实现的服务

根据 `api/grpc/iam/identity/v1/identity.proto` 文件，我们实现了以下 4 个 gRPC 服务：

### 1. IdentityRead - 身份读取服务
用于查询用户和儿童的身份信息。

**RPC 方法：**
- `GetUser` - 获取单个用户信息
- `BatchGetUsers` - 批量获取用户信息
- `SearchUsers` - 搜索用户
- `GetChild` - 获取单个儿童信息
- `BatchGetChildren` - 批量获取儿童信息

### 2. GuardianshipQuery - 监护关系查询服务
用于查询监护关系。

**RPC 方法：**
- `IsGuardian` - 检查是否为监护人
- `ListChildren` - 列出监护人的所有儿童
- `ListGuardians` - 列出儿童的所有监护人

### 3. GuardianshipCommand - 监护关系命令服务
用于管理监护关系。

**RPC 方法：**
- `AddGuardian` - 添加监护人
- `UpdateGuardianRelation` - 更新监护关系
- `RevokeGuardian` - 撤销监护人
- `BatchRevokeGuardians` - 批量撤销监护人
- `ImportGuardians` - 导入监护人

### 4. IdentityLifecycle - 身份生命周期服务
用于管理用户的创建、更新、状态变更等。

**RPC 方法：**
- `CreateUser` - 创建用户
- `UpdateUser` - 更新用户信息
- `DeactivateUser` - 停用用户
- `BlockUser` - 封禁用户
- `LinkExternalIdentity` - 关联外部身份

## 代码结构

### 服务实现层
位置：`internal/apiserver/interface/uc/grpc/`

```
internal/apiserver/interface/uc/grpc/
├── service.go                    # UC gRPC 服务聚合器
└── identity/
    ├── service.go                # Identity 服务聚合器
    ├── service_impl.go           # RPC 方法实现
    └── mapper.go                 # 数据转换函数
```

#### service.go (UC 聚合器)
- 聚合所有 UC 相关的 gRPC 服务
- 提供统一的注册方法

#### identity/service.go
- 创建 4 个服务器实例：
  - `identityReadServer` - 身份读取
  - `guardianshipQueryServer` - 监护关系查询
  - `guardianshipCommandServer` - 监护关系命令
  - `identityLifecycleServer` - 身份生命周期
- 依赖注入：领域仓储、应用服务

#### identity/service_impl.go
- 实现所有 RPC 方法
- 错误处理和状态码映射
- 分页参数处理

#### identity/mapper.go
- `userResultToProto()` - 用户结果转 Proto
- `childResultToProto()` - 儿童结果转 Proto  
- `guardianshipResultToProto()` - 监护关系转 Proto
- `toGRPCError()` - 错误码映射

### 依赖注入

#### Container 集成
- 位置：`internal/apiserver/container/assembler/user.go`
- gRPC 服务已集成到 `UserModule` 中
- `UserModule.GRPCService` 字段存储 UC gRPC 服务

#### 初始化流程
在 `UserModule.Initialize()` 方法中：

```go
// 1. 创建仓储层
userRepo := userInfra.NewRepository(db)
childRepo := childInfra.NewRepository(db)
guardRepo := guardianshipInfra.NewRepository(db)

// 2. 创建应用服务层
userQuerySrv := appuser.NewUserQueryApplicationService(uow)
childQuerySrv := appchild.NewChildQueryApplicationService(uow)
guardQuerySrv := appguard.NewGuardianshipQueryApplicationService(uow)
// ... 其他服务

// 3. 创建 identity gRPC 服务
identitySvc := identityGrpc.NewService(
    userRepo,
    childRepo,
    guardRepo,
    userQuerySrv,
    childQuerySrv,
    guardQuerySrv,
    userAppSrv,
    userProfileAppSrv,
    userStatusSrv,
    guardAppSrv,
)

// 4. 聚合到 UC gRPC 服务
m.GRPCService = ucGrpc.NewService(identitySvc)
```

### 服务注册

#### server.go 集成
位置：`internal/apiserver/server.go`

在 `registerGRPCServices()` 方法中注册：

```go
// 注册用户模块的 gRPC 服务（包含 Identity 相关服务）
if s.container.UserModule != nil && s.container.UserModule.GRPCService != nil {
    s.container.UserModule.GRPCService.Register(s.grpcServer.Server)
    log.Info("📡 Registered User gRPC services (IdentityRead, GuardianshipQuery, GuardianshipCommand, IdentityLifecycle)")
}
```

## 架构特点

### 六边形架构
- **接口层 (Interface)**: gRPC 服务实现
- **应用层 (Application)**: 业务逻辑编排
- **领域层 (Domain)**: 领域模型和仓储接口
- **基础设施层 (Infra)**: 数据库访问实现

### 依赖关系
```
gRPC Service (interface)
    ↓ 依赖
Application Service (application)
    ↓ 依赖
Domain Repository (domain)
    ↓ 实现
MySQL Repository (infra/mysql)
```

### 错误处理
- 使用 `toGRPCError()` 将应用层错误码映射到 gRPC 状态码
- 支持的映射：
  - `code.ErrUserNotFound` → `codes.NotFound`
  - `code.ErrInvalidParameter` → `codes.InvalidArgument`
  - `code.ErrDatabase` → `codes.Internal`
  - 其他 → `codes.Unknown`

## 测试

### 验证服务注册
启动服务后，查看日志：
```
📡 Registered User gRPC services (IdentityRead, GuardianshipQuery, GuardianshipCommand, IdentityLifecycle)
✅ All gRPC services registered successfully
```

### grpcurl 测试示例

#### 1. 获取用户信息
```bash
grpcurl -plaintext -d '{"user_id": 1}' \
  localhost:8081 iam.identity.v1.IdentityRead/GetUser
```

#### 2. 批量获取用户
```bash
grpcurl -plaintext -d '{"user_ids": [1, 2, 3]}' \
  localhost:8081 iam.identity.v1.IdentityRead/BatchGetUsers
```

#### 3. 搜索用户
```bash
grpcurl -plaintext -d '{"query": "张三", "page_num": 1, "page_size": 10}' \
  localhost:8081 iam.identity.v1.IdentityRead/SearchUsers
```

#### 4. 检查监护关系
```bash
grpcurl -plaintext -d '{"guardian_user_id": 1, "child_id": 100}' \
  localhost:8081 iam.identity.v1.GuardianshipQuery/IsGuardian
```

#### 5. 添加监护人
```bash
grpcurl -plaintext -d '{
  "guardian_user_id": 1,
  "child_id": 100,
  "relation_type": "PARENT"
}' localhost:8081 iam.identity.v1.GuardianshipCommand/AddGuardian
```

#### 6. 创建用户
```bash
grpcurl -plaintext -d '{
  "username": "newuser",
  "real_name": "新用户",
  "phone": "13800138000"
}' localhost:8081 iam.identity.v1.IdentityLifecycle/CreateUser
```

### 使用 grpcui 可视化测试
```bash
grpcui -plaintext localhost:8081
```

## 后续优化建议

1. **性能优化**
   - 批量查询时使用 DataLoader 减少 N+1 查询
   - 添加缓存层（Redis）
   - 实现流式 RPC 处理大量数据

2. **监控和追踪**
   - 添加 gRPC 拦截器记录请求日志
   - 集成 OpenTelemetry 追踪
   - 添加 Prometheus 指标

3. **安全增强**
   - 实现 gRPC 认证拦截器
   - 添加权限验证
   - 实现 TLS 加密

4. **测试覆盖**
   - 添加单元测试
   - 添加集成测试
   - 实现 E2E 测试

## 相关文档

- [Identity Proto 定义](../../../api/grpc/iam/identity/v1/identity.proto)
- [系统架构概览](../../overview/system-overview.md)
- [API 测试指南](../../quality/api-testing-guide.md)

## 变更历史

- 2024-XX-XX: 完成 Identity 模块 gRPC 服务实现
- 2024-XX-XX: 集成到 UserModule，移除独立的 UCModule
- 2024-XX-XX: 修复仓储接口，统一使用 meta.ID
