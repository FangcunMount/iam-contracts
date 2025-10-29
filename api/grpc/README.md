# gRPC API 文档

> IAM Contracts gRPC API 规范（Protocol Buffers）

## 📋 文档列表

### 1. [授权服务 (iam.authz.v1.proto)](./iam.authz.v1.proto)

**功能域**: 权限判定与策略决策

#### 服务定义

```protobuf
service AuthZ {
  rpc Allow        (AllowReq)        returns (AllowResp);
  rpc AllowOnActor (AllowOnActorReq) returns (AllowResp);
  rpc BatchAllow   (BatchAllowReq)   returns (BatchAllowResp);
  rpc Explain      (ExplainReq)      returns (ExplainResp);
}
```

#### RPC 方法说明

| 方法 | 请求 | 响应 | 说明 | 适用场景 |
|------|------|------|------|----------|
| **Allow** | `AllowReq` | `AllowResp` | 基础权限判定 | 判断用户对资源的操作权限 |
| **AllowOnActor** | `AllowOnActorReq` | `AllowResp` | 基于 Actor 的权限判定 | 判断用户对特定 Actor（如儿童）的操作权限 |
| **BatchAllow** | `BatchAllowReq` | `BatchAllowResp` | 批量权限判定 | 一次性判定多个权限（减少 RPC 调用） |
| **Explain** | `ExplainReq` | `ExplainResp` | 权限决策解释 | 调试/审计：了解权限判定依据 |

---

#### 1.1 Allow - 基础权限判定

**功能**: 判断用户是否有权限对资源执行特定操作

**请求示例**:

```protobuf
AllowReq {
  user_id: "usr_1234567890"
  resource: "answersheet"
  action: "submit"
  scope: {
    type: "questionnaire"
    id: "PHQ9"
  }
}
```

**响应示例**:

```protobuf
AllowResp {
  allow: true
  reason: "ok"
}
```

**Go 客户端示例**:

```go
import (
    authzv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authz/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

// 创建客户端
conn, err := grpc.Dial("api.example.com:9090",
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
)
defer conn.Close()

client := authzv1.NewAuthZClient(conn)

// 调用 Allow
resp, err := client.Allow(ctx, &authzv1.AllowReq{
    UserId:   "usr_1234567890",
    Resource: "answersheet",
    Action:   "submit",
    Scope: &authzv1.Scope{
        Type: "questionnaire",
        Id:   "PHQ9",
    },
})

if err != nil {
    log.Fatalf("Allow failed: %v", err)
}

if resp.Allow {
    // 允许操作
    fmt.Println("Permission granted")
} else {
    // 拒绝操作
    fmt.Printf("Permission denied: %s\n", resp.Reason)
}
```

**常见 Reason 值**:

| Reason | 说明 | 处理建议 |
|--------|------|----------|
| `ok` | 权限允许 | 执行操作 |
| `no_role` | 用户无对应角色 | 提示用户申请角色 |
| `no_guardianship` | 无监护关系 | 提示用户建立监护关系 |
| `pdp_error` | 策略引擎错误 | 记录日志，联系管理员 |
| `scope_mismatch` | 作用域不匹配 | 检查请求参数 |

---

#### 1.2 AllowOnActor - 基于 Actor 的权限判定

**功能**: 判断用户对特定 Actor（如儿童、患者）的操作权限

**适用场景**:

- 监护人操作儿童档案
- 医生访问患者数据
- 教师管理学生信息

**请求示例**:

```protobuf
AllowOnActorReq {
  user_id: "usr_1234567890"
  resource: "child_profile"
  action: "update"
  scope: {
    type: "system"
    id: "*"
  }
  actor: {
    type: "testee"
    id: "chd_9876543210"
    scope: {
      type: "org"
      id: "HOSP-001"
    }
  }
}
```

**响应示例**:

```protobuf
AllowResp {
  allow: true
  reason: "ok"  // 通过监护关系验证
}
```

**Go 客户端示例**:

```go
resp, err := client.AllowOnActor(ctx, &authzv1.AllowOnActorReq{
    UserId:   "usr_1234567890",
    Resource: "child_profile",
    Action:   "update",
    Scope: &authzv1.Scope{
        Type: "system",
        Id:   "*",
    },
    Actor: &authzv1.ActorRef{
        Type: "testee",
        Id:   "chd_9876543210",
    },
})

if resp.Allow {
    // 有权更新儿童档案（已验证监护关系）
    updateChildProfile(actorID, newData)
}
```

---

#### 1.3 BatchAllow - 批量权限判定

**功能**: 一次性判定多个权限请求（性能优化）

**适用场景**:

- 页面加载时批量检查按钮权限
- 列表渲染时批量判定操作权限
- 减少网络往返次数

**请求示例**:

```protobuf
BatchAllowReq {
  checks: [
    {
      user_id: "usr_1234567890"
      resource: "answersheet"
      action: "read"
      scope: { type: "questionnaire", id: "PHQ9" }
    },
    {
      user_id: "usr_1234567890"
      resource: "answersheet"
      action: "submit"
      scope: { type: "questionnaire", id: "PHQ9" }
    },
    {
      user_id: "usr_1234567890"
      resource: "answersheet"
      action: "delete"
      scope: { type: "questionnaire", id: "PHQ9" }
    }
  ]
}
```

**响应示例**:

```protobuf
BatchAllowResp {
  results: [
    { allow: true, reason: "ok" },         // read: 允许
    { allow: true, reason: "ok" },         // submit: 允许
    { allow: false, reason: "no_role" }    // delete: 拒绝
  ]
}
```

**Go 客户端示例**:

```go
// 批量检查权限
resp, err := client.BatchAllow(ctx, &authzv1.BatchAllowReq{
    Checks: []*authzv1.AllowReq{
        {UserId: userID, Resource: "answersheet", Action: "read", Scope: scope},
        {UserId: userID, Resource: "answersheet", Action: "submit", Scope: scope},
        {UserId: userID, Resource: "answersheet", Action: "delete", Scope: scope},
    },
})

// 解析结果
permissions := map[string]bool{
    "read":   resp.Results[0].Allow,
    "submit": resp.Results[1].Allow,
    "delete": resp.Results[2].Allow,
}

// 根据权限渲染 UI
renderUI(permissions)
```

---

#### 1.4 Explain - 权限决策解释

**功能**: 提供权限判定的详细解释（调试/审计用）

**请求示例**:

```protobuf
ExplainReq {
  check: {
    user_id: "usr_1234567890"
    resource: "answersheet"
    action: "submit"
    scope: { type: "questionnaire", id: "PHQ9" }
  }
}
```

**响应示例**:

```protobuf
ExplainResp {
  allow: true
  reason: "ok"
  matched_policies: [
    "p, role:psychologist, answersheet, submit, questionnaire:PHQ9",
    "g, usr_1234567890, role:psychologist"
  ]
}
```

**Go 客户端示例**:

```go
resp, err := client.Explain(ctx, &authzv1.ExplainReq{
    Check: &authzv1.AllowReq{
        UserId:   "usr_1234567890",
        Resource: "answersheet",
        Action:   "submit",
        Scope:    scope,
    },
})

// 输出决策依据
fmt.Printf("Allow: %v\n", resp.Allow)
fmt.Printf("Reason: %s\n", resp.Reason)
fmt.Println("Matched Policies:")
for _, policy := range resp.MatchedPolicies {
    fmt.Printf("  - %s\n", policy)
}
```

---

### 2. [身份查询 (iam.identity.v1.proto)](./iam.identity.v1.proto)

**功能域**: 用户、儿童档案查询与监护关系判定

#### 身份查询服务定义

```protobuf
service IdentityRead {
  rpc GetUser  (GetUserReq)  returns (GetUserResp);
  rpc GetChild (GetChildReq) returns (GetChildResp);
}

service GuardianshipQuery {
  rpc IsGuardian   (IsGuardianReq)   returns (IsGuardianResp);
  rpc ListChildren (ListChildrenReq) returns (ListChildrenResp);
}
```

#### 身份查询 RPC 方法说明

| 服务 | 方法 | 请求 | 响应 | 说明 |
|------|------|------|------|------|
| **IdentityRead** | `GetUser` | `GetUserReq` | `GetUserResp` | 查询用户信息 |
| | `GetChild` | `GetChildReq` | `GetChildResp` | 查询儿童档案 |
| **GuardianshipQuery** | `IsGuardian` | `IsGuardianReq` | `IsGuardianResp` | 判定监护关系 |
| | `ListChildren` | `ListChildrenReq` | `ListChildrenResp` | 列出监护儿童 |

---

#### 2.1 GetUser - 查询用户信息

**请求示例**:

```protobuf
GetUserReq {
  user_id: "usr_1234567890"
}
```

**响应示例**:

```protobuf
GetUserResp {
  user: {
    id: "usr_1234567890"
    status: "active"
    nickname: "张三"
    avatar: "https://cdn.example.com/avatars/usr_1234567890.jpg"
    created_at: { seconds: 1698566400 }
    updated_at: { seconds: 1698566400 }
  }
}
```

**Go 客户端示例**:

```go
import identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"

client := identityv1.NewIdentityReadClient(conn)

resp, err := client.GetUser(ctx, &identityv1.GetUserReq{
    UserId: "usr_1234567890",
})

if err != nil {
    log.Fatalf("GetUser failed: %v", err)
}

fmt.Printf("User: %s (Status: %s)\n", resp.User.Nickname, resp.User.Status)
```

---

#### 2.2 GetChild - 查询儿童档案

**请求示例**:

```protobuf
GetChildReq {
  child_id: "chd_9876543210"
}
```

**响应示例**:

```protobuf
GetChildResp {
  child: {
    id: "chd_9876543210"
    legal_name: "小明"
    gender: 1
    dob: "2020-05-15"
    id_type: "id_card"
    height_cm: 105
    weight_kg: "18.5"
    created_at: { seconds: 1698566400 }
    updated_at: { seconds: 1698566400 }
  }
}
```

**Go 客户端示例**:

```go
resp, err := client.GetChild(ctx, &identityv1.GetChildReq{
    ChildId: "chd_9876543210",
})

child := resp.Child
fmt.Printf("Child: %s (Gender: %d, DOB: %s)\n", 
    child.LegalName, child.Gender, child.Dob)
```

---

#### 2.3 IsGuardian - 判定监护关系

**功能**: 快速判定用户是否为儿童的监护人（高性能查询）

**请求示例**:

```protobuf
IsGuardianReq {
  user_id: "usr_1234567890"
  child_id: "chd_9876543210"
}
```

**响应示例**:

```protobuf
IsGuardianResp {
  is_guardian: true
}
```

**Go 客户端示例**:

```go
guardianClient := identityv1.NewGuardianshipQueryClient(conn)

resp, err := guardianClient.IsGuardian(ctx, &identityv1.IsGuardianReq{
    UserId:  "usr_1234567890",
    ChildId: "chd_9876543210",
})

if resp.IsGuardian {
    // 允许访问儿童数据
    accessChildData(childID)
} else {
    // 拒绝访问
    return errors.New("not a guardian")
}
```

**性能优化**:

- 内部使用缓存（Redis）
- 平均响应时间 < 10ms
- 适合高频调用场景

---

#### 2.4 ListChildren - 列出监护儿童

**功能**: 查询用户监护的所有儿童

**请求示例**:

```protobuf
ListChildrenReq {
  user_id: "usr_1234567890"
  limit: 20
  offset: 0
}
```

**响应示例**:

```protobuf
ListChildrenResp {
  total: 2
  items: [
    {
      id: "chd_9876543210"
      legal_name: "小明"
      gender: 1
      dob: "2020-05-15"
      id_type: "id_card"
      height_cm: 105
      weight_kg: "18.5"
    },
    {
      id: "chd_1111111111"
      legal_name: "小红"
      gender: 2
      dob: "2021-03-20"
      id_type: "id_card"
      height_cm: 95
      weight_kg: "15.0"
    }
  ]
}
```

**Go 客户端示例**:

```go
resp, err := guardianClient.ListChildren(ctx, &identityv1.ListChildrenReq{
    UserId: "usr_1234567890",
    Limit:  20,
    Offset: 0,
})

fmt.Printf("Total children: %d\n", resp.Total)
for _, child := range resp.Items {
    fmt.Printf("  - %s (ID: %s)\n", child.LegalName, child.Id)
}
```

---

## 🔐 认证与安全

### mTLS 配置

gRPC 使用双向 TLS 认证：

```go
// 加载客户端证书
cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
if err != nil {
    log.Fatalf("failed to load key pair: %v", err)
}

// 加载 CA 证书
caCert, err := ioutil.ReadFile("ca.crt")
if err != nil {
    log.Fatalf("failed to read ca cert: %v", err)
}

caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

// TLS 配置
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      caCertPool,
    ServerName:   "api.example.com",
}

// 创建连接
creds := credentials.NewTLS(tlsConfig)
conn, err := grpc.Dial("api.example.com:9090", 
    grpc.WithTransportCredentials(creds),
)
```

### Metadata 认证

```go
import (
    "google.golang.org/grpc/metadata"
)

// 添加认证 metadata
ctx := metadata.AppendToOutgoingContext(ctx,
    "authorization", "Bearer "+token,
    "x-request-id", requestID,
)

// 调用 RPC
resp, err := client.Allow(ctx, req)
```

---

## ⚡ 性能与最佳实践

### 1. 连接复用

```go
// ❌ 不推荐：每次调用都创建新连接
func checkPermission(userID, resource string) bool {
    conn, _ := grpc.Dial("api.example.com:9090")
    defer conn.Close()
    client := authzv1.NewAuthZClient(conn)
    resp, _ := client.Allow(ctx, req)
    return resp.Allow
}

// ✅ 推荐：复用连接
var (
    conn   *grpc.ClientConn
    client authzv1.AuthZClient
)

func init() {
    conn, _ = grpc.Dial("api.example.com:9090", opts...)
    client = authzv1.NewAuthZClient(conn)
}

func checkPermission(userID, resource string) bool {
    resp, _ := client.Allow(ctx, req)
    return resp.Allow
}
```

### 2. 批量调用

```go
// ❌ 不推荐：多次单独调用
canRead := checkPermission(userID, "answersheet", "read")
canWrite := checkPermission(userID, "answersheet", "write")
canDelete := checkPermission(userID, "answersheet", "delete")

// ✅ 推荐：使用 BatchAllow
resp, _ := client.BatchAllow(ctx, &authzv1.BatchAllowReq{
    Checks: []*authzv1.AllowReq{
        {UserId: userID, Resource: "answersheet", Action: "read"},
        {UserId: userID, Resource: "answersheet", Action: "write"},
        {UserId: userID, Resource: "answersheet", Action: "delete"},
    },
})
```

### 3. 超时控制

```go
// 设置 RPC 超时
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

resp, err := client.Allow(ctx, req)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Error("RPC timeout")
    }
}
```

### 4. 错误处理

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

resp, err := client.Allow(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.InvalidArgument:
            log.Error("Invalid request parameters")
        case codes.Unauthenticated:
            log.Error("Authentication failed")
        case codes.PermissionDenied:
            log.Error("Permission denied")
        case codes.NotFound:
            log.Error("Resource not found")
        default:
            log.Errorf("gRPC error: %v", st.Message())
        }
    }
}
```

---

## 🧪 测试工具

### BloomRPC

1. 下载 [BloomRPC](https://github.com/bloomrpc/bloomrpc)
2. 导入 proto 文件: `iam.authz.v1.proto`, `iam.identity.v1.proto`
3. 配置服务地址: `api.example.com:9090`
4. 配置 TLS 证书
5. 发送测试请求

### grpcurl

```bash
# 列出服务
grpcurl -cacert ca.crt api.example.com:9090 list

# 列出方法
grpcurl -cacert ca.crt api.example.com:9090 list iam.authz.v1.AuthZ

# 调用 RPC
grpcurl -cacert ca.crt \
  -d '{"user_id":"usr_123","resource":"answersheet","action":"submit"}' \
  api.example.com:9090 iam.authz.v1.AuthZ/Allow
```

### Postman

Postman 已支持 gRPC：

1. 新建 gRPC Request
2. 导入 proto 文件
3. 配置服务 URL 和方法
4. 设置 Metadata (authorization, x-request-id)
5. 发送请求

---

## 📚 相关资源

- **Proto 文件**:
  - [iam.authz.v1.proto](./iam.authz.v1.proto) - 授权服务 proto 定义
  - [iam.identity.v1.proto](./iam.identity.v1.proto) - 身份查询 proto 定义

- **代码生成**:

  ```bash
  # Go
  make proto-gen
  
  # Python
  make proto-gen-python
  
  # TypeScript
  make proto-gen-ts
  ```

- **SDK**:
  - [Go SDK](https://github.com/FangcunMount/iam-sdk-go)
  - [Python SDK](https://pypi.org/project/iam-grpc-client/)
  - [Node.js SDK](https://www.npmjs.com/package/@iam/grpc-client)

---

## 📞 技术支持

- **gRPC 问题**: [GitHub Issues](https://github.com/FangcunMount/iam-contracts/issues)
- **性能问题**: <performance@example.com>
- **安全问题**: <security@example.com>
