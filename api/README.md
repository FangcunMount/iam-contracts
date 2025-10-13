# IAM · Identity API（v1）

本目录提供「基础用户（Identity）」模块的公开 API 文档与规范文件，面向调用方（业务后端、运营后台、网关）。

- 规范：`identity.v1.yaml`（OpenAPI 3.1）
- 版本：v1（向后兼容小改，破坏性升级将发布 v2）

## 基础信息

- Base URL：`/api/v1`
- 认证：`Authorization: Bearer <JWT>`（JWT.sub = user_id）
- 幂等：`POST` 支持 `X-Idempotency-Key`（建议 UUID，24h 内复用视为同一次）
- 追踪：可上传 `X-Request-Id`，服务端将回显
- 时间：ISO8601（UTC 或带时区）
- 分页：`?limit=<1..100>&offset=<0..>`

## RBAC & Scope（必读）

- 资源：`identity.user` / `identity.child` / `identity.guardian`
- 动作：`read | write | register | grant | revoke`
- Scope：`{scope_type, scope_id}`，默认 `{system,"*"}`
- 判定：  
  - 非对象型：`Allow(user, resource, action, scope)`  
  - 对象型（涉及 Child）：`AllowOnActor = RBAC.Allow(...) ∧ Guardianship.HasGuardian(user, child)`

> **注意**：Scope 由路由/网关提取（例如 `/orgs/{orgId}/...` → `{org, orgId}`），调用方通常无需显式传入。

## 错误返回

```json
{ "code":"Identity.ChildExists", "message":"child already exists", "requestId":"..." }
```

### 常见错误码：
| 错误码                     | 含义                             | HTTP 状态码 |
|----------------------------|----------------------------------|-------------|
| `InvalidArgument`          | 参数错误                         | 400         |
| `Unauthenticated`          | 认证失败（缺少/无效 Token）     | 401         |
| `Forbidden`                | 访问被拒绝                       | 403         |
| `NotFound`                 | 资源未找到                       | 404         |
| `Conflict`                 | 资源冲突                         | 409         |
| `Internal`                 | 服务器内部错误                   | 500         |

## 快速开始（cURL 示例）

### (1) 注册孩子（并授当前用户为监护人）

```bash
curl -X POST "$HOST/api/v1/children:register" \
 -H "Authorization: Bearer $JWT" \
 -H "Content-Type: application/json" \
 -H "X-Idempotency-Key: $(uuidgen)" \
 -d '{
   "legalName":"张三","gender":1,"dob":"2018-09-01",
   "idType":"idcard","idNo":"4403**********1234",
   "heightCm":120,"weightKg":"22.5","relation":"parent"
 }'
```

### (2) 撤销监护

```bash
curl -X POST "$HOST/api/v1/guardians:revoke" \
 -H "Authorization: Bearer $JWT" \
 -H "Content-Type: application/json" \
 -d '{"userId":"usr_parent","childId":"chd_001","relation":"parent"}'
```

更多接口与数据模型请见 identity.v1.yaml
