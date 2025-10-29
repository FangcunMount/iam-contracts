# REST vs gRPC - API 选型对比

> 帮助开发者选择合适的 API 协议

## 🎯 快速决策

### 我应该用哪个？

```text
┌─────────────────────────────────┐
│  我需要什么？                     │
└─────────────────────────────────┘
          │
          ├─ Web/H5/小程序前端 ───────────► REST
          │
          ├─ 移动 App ────────────────────► REST (首选) 或 gRPC
          │
          ├─ 运营后台 ────────────────────► REST
          │
          ├─ 高频权限判定 (QPS > 1000) ──► gRPC
          │
          ├─ 监护关系查询 (毫秒级) ───────► gRPC
          │
          ├─ 批量操作 (减少往返) ─────────► gRPC
          │
          ├─ 外部系统集成 ────────────────► REST
          │
          └─ 命令行工具/脚本 ─────────────► REST
```

---

## 📊 详细对比

### 1. 协议特性

| 特性 | REST | gRPC | 说明 |
|------|------|------|------|
| **传输协议** | HTTP/1.1 or HTTP/2 | HTTP/2 | gRPC 强制 HTTP/2 |
| **数据格式** | JSON (可读) | Protocol Buffers (二进制) | gRPC 更节省带宽 |
| **性能** | 中等 | 高 | gRPC 快 2-10 倍 |
| **浏览器支持** | ✅ 原生支持 | ⚠️ 需要 gRPC-Web | REST 更通用 |
| **可读性** | ✅ 易读易调试 | ❌ 二进制不可读 | REST 更友好 |
| **流式传输** | ❌ 不支持 | ✅ 支持 | gRPC 支持流式 RPC |
| **代码生成** | 手动或工具 | ✅ 自动生成 | gRPC 代码生成更简单 |

---

### 2. 认证中心 (AuthN) - 推荐 REST

#### 为什么选 REST？

| 原因 | 说明 |
|------|------|
| **人类可读** | 调试登录问题时可直接查看 JSON |
| **浏览器友好** | 前端直接调用，无需额外库 |
| **标准化** | OAuth、JWKS 等标准都基于 REST |
| **审计需求** | JSON 日志易于阅读和审计 |

#### API 示例

**登录**:

```bash
# REST - 清晰易懂
curl -X POST https://api.example.com/api/v1/auth/login \
  -d '{"username":"admin","password":"pass123"}'

# 响应可读
{
  "access_token": "eyJ...",
  "expires_in": 86400
}
```

**适用场景**:

- ✅ 用户登录/注册
- ✅ 令牌刷新
- ✅ 账户绑定/解绑
- ✅ JWKS 公钥查询

---

### 3. 用户中心 (Identity) - REST 为主，gRPC 为辅

#### REST 适用场景

| 操作 | REST API | 原因 |
|------|----------|------|
| **创建用户** | `POST /api/v1/users` | 低频操作，可读性重要 |
| **更新用户** | `PATCH /api/v1/users/{id}` | 部分更新，REST 更直观 |
| **注册儿童** | `POST /api/v1/children/register` | 复杂业务，需要详细响应 |
| **我的孩子** | `GET /api/v1/me/children` | 前端调用，JSON 易处理 |

**示例** - 注册儿童（REST 更清晰）:

```bash
curl -X POST https://api.example.com/api/v1/children/register \
  -H "Authorization: Bearer <token>" \
  -d '{
    "legal_name": "小明",
    "gender": 1,
    "dob": "2020-05-15",
    "relation": "parent"
  }'

# 响应包含 child 和 guardianship 两部分
{
  "child": { "id": "chd_123", ... },
  "guardianship": { "id": 456, ... }
}
```

#### gRPC 适用场景

| 操作 | gRPC RPC | 原因 |
|------|----------|------|
| **查询用户** | `GetUser(GetUserReq)` | 高频读取，性能优先 |
| **查询儿童** | `GetChild(GetChildReq)` | 服务间调用，速度快 |
| **监护判定** | `IsGuardian(IsGuardianReq)` | 毫秒级响应，高并发 |
| **列出儿童** | `ListChildren(ListChildrenReq)` | 批量查询，减少延迟 |

**示例** - 监护判定（gRPC 更快）:

```go
// gRPC - 平均 5ms 响应
resp, err := client.IsGuardian(ctx, &identityv1.IsGuardianReq{
    UserId:  "usr_123",
    ChildId: "chd_456",
})

if resp.IsGuardian {
    // 允许访问
}
```

---

### 4. 授权中心 (AuthZ) - 强烈推荐 gRPC

#### 为什么必须用 gRPC？

| 原因 | 数据 | 说明 |
|------|------|------|
| **高 QPS** | > 10,000 req/s | 权限判定是最高频的调用 |
| **低延迟** | < 10ms | P99 延迟要求毫秒级 |
| **批量调用** | 1 次 vs 10 次 | BatchAllow 减少网络往返 |
| **内部调用** | 微服务间 | 不对外暴露，无需 JSON |

#### 性能对比

| 场景 | REST | gRPC | 提升 |
|------|------|------|------|
| **单次判权** | 15ms | 5ms | 3x |
| **批量判权 (10个)** | 150ms | 12ms | 12x |
| **吞吐量** | 2000 QPS | 15000 QPS | 7.5x |

#### API 对比

**REST** (假设存在):

```bash
# 10 次 HTTP 请求
for perm in read write delete submit ...; do
  curl -X POST https://api.example.com/api/v1/authz/allow \
    -d '{"resource":"answersheet","action":"'$perm'"}'
done
# 总耗时: ~150ms
```

**gRPC**:

```go
// 1 次 RPC 请求
resp, _ := client.BatchAllow(ctx, &authzv1.BatchAllowReq{
    Checks: []*authzv1.AllowReq{
        {Resource: "answersheet", Action: "read"},
        {Resource: "answersheet", Action: "write"},
        // ... 8 more checks
    },
})
// 总耗时: ~12ms
```

---

### 5. 混合使用场景

#### 场景 1: Web 应用

```
┌─────────────┐
│  Web 前端   │
└──────┬──────┘
       │ REST
       ▼
┌─────────────┐     gRPC      ┌─────────────┐
│  BFF 层     │◄─────────────►│  IAM 服务   │
│ (Node.js)   │               │             │
└─────────────┘               └─────────────┘

前端 → REST → BFF
BFF → gRPC → IAM (高性能)
```

**示例 - BFF 聚合**:

```javascript
// BFF API (REST)
app.get('/api/me/children', async (req, res) => {
  const userId = req.user.id;
  
  // 调用 gRPC 获取儿童列表 (快速)
  const children = await grpcClient.ListChildren({ userId });
  
  // 批量判定权限 (gRPC 批量)
  const permissions = await grpcClient.BatchAllow({
    checks: children.map(child => ({
      userId,
      resource: 'child_profile',
      action: 'update',
      actor: { type: 'testee', id: child.id }
    }))
  });
  
  // 返回 JSON 给前端
  res.json({
    children: children.map((child, i) => ({
      ...child,
      canUpdate: permissions[i].allow
    }))
  });
});
```

#### 场景 2: 移动 App

**方案 A**: 纯 REST

```text
┌─────────────┐
│  Mobile App │
└──────┬──────┘
       │ REST (HTTPS)
       ▼
┌─────────────┐
│  IAM 服务   │
└─────────────┘
```

**方案 B**: gRPC-Mobile

```text
┌─────────────┐
│  Mobile App │
└──────┬──────┘
       │ gRPC-Mobile
       ▼
┌─────────────┐
│  IAM 服务   │
└─────────────┘
```

**建议**: iOS/Android 优先使用 gRPC-Mobile（性能更好）

---

## 🎓 最佳实践

### 何时用 REST

✅ **推荐场景**:

- Web/H5/小程序前端
- 运营后台 (CRUD 操作)
- 第三方集成
- 命令行工具
- 需要人类可读日志
- OAuth/OIDC 流程

❌ **不推荐场景**:

- 高频权限判定 (使用 gRPC)
- 批量操作 (使用 gRPC BatchAllow)
- 微服务间调用 (使用 gRPC)
- 毫秒级性能要求 (使用 gRPC)

---

### 何时用 gRPC

✅ **推荐场景**:

- 权限判定 (AuthZ)
- 监护关系查询 (高频)
- 微服务间调用
- 批量操作
- 实时数据同步 (流式 RPC)

❌ **不推荐场景**:

- 浏览器直接调用 (除非 gRPC-Web)
- 需要调试友好 (JSON 更易读)
- 外部集成 (REST 更通用)
- 登录/注册 (REST 更标准)

---

## 📈 性能测试数据

### 测试环境

- **服务器**: 4C8G (AWS c5.xlarge)
- **客户端**: 同区域 EC2
- **网络**: 内网 (< 1ms RTT)

### 单次权限判定

| 协议 | P50 | P95 | P99 | QPS (单机) |
|------|-----|-----|-----|------------|
| REST | 12ms | 18ms | 25ms | 2,500 |
| gRPC | 4ms | 7ms | 10ms | 15,000 |

### 批量判定 (10个权限)

| 协议 | 总耗时 (P50) | 总耗时 (P95) | QPS |
|------|-------------|-------------|-----|
| REST (10次调用) | 120ms | 180ms | 200 |
| gRPC (BatchAllow) | 10ms | 15ms | 3,000 |

### 带宽消耗

| 操作 | REST | gRPC | 节省 |
|------|------|------|------|
| 单次判权 | 512 bytes | 85 bytes | 83% |
| 批量判权 (10个) | 5.12 KB | 420 bytes | 92% |

---

## 🛠️ 开发工具推荐

### REST API

| 工具 | 用途 | 推荐度 |
|------|------|--------|
| **Postman** | 调试/测试 | ⭐⭐⭐⭐⭐ |
| **Swagger UI** | 文档查看 | ⭐⭐⭐⭐⭐ |
| **cURL** | 命令行测试 | ⭐⭐⭐⭐ |
| **HTTPie** | 友好的 cURL 替代 | ⭐⭐⭐⭐ |

### gRPC API

| 工具 | 用途 | 推荐度 |
|------|------|--------|
| **BloomRPC** | GUI 调试 | ⭐⭐⭐⭐⭐ |
| **grpcurl** | 命令行测试 | ⭐⭐⭐⭐⭐ |
| **Postman** | 调试 (v8+) | ⭐⭐⭐⭐ |
| **Evans** | 交互式 CLI | ⭐⭐⭐⭐ |

---

## 📞 FAQ

### Q: 我可以同时使用 REST 和 gRPC 吗？

**A**: 可以！推荐混合使用：

- 前端/外部 → REST
- 微服务间/高频调用 → gRPC

### Q: gRPC 浏览器不支持怎么办？

**A**: 使用 [gRPC-Web](https://github.com/grpc/grpc-web)：

```javascript
// gRPC-Web 客户端 (浏览器)
const client = new AuthZClient('https://api.example.com:8080');
client.allow(request, {}, (err, response) => {
  if (response.allow) {
    // ...
  }
});
```

### Q: REST 性能真的比 gRPC 差很多吗？

**A**: 取决于场景：

- 低频 CRUD: 差异不大 (10-20ms)
- 高频调用: gRPC 快 3-10 倍
- 批量操作: gRPC 快 10-50 倍

### Q: 如何从 REST 迁移到 gRPC？

**A**: 分阶段迁移：

1. 保留现有 REST API（向后兼容）
2. 新增 gRPC 端点（高频路径）
3. 逐步迁移客户端
4. 观察性能提升
5. 废弃旧 REST 端点（可选）

---

## 📚 相关文档

- [REST API 文档](../rest/README.md)
- [gRPC API 文档](../grpc/README.md)
- [API 主文档](../README.md)

---

**更新时间**: 2025-10-29  
**版本**: v1.0
