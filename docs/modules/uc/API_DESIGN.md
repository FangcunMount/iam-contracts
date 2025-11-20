# 用户中心 - API 设计

> [返回用户中心文档](./README.md)

本文档详细介绍用户中心的 RESTful API 和 gRPC API 设计。

---

## 6. API 设计

### 6.1 RESTful API

#### 6.1.1 用户管理 API

> **注意**: 用户注册通过认证中心（Authn）模块完成，请参考 `/api/v1/auth/register` 端点。

```http
# 获取当前用户资料
GET /api/v1/me
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "usr_1234567890",
  "name": "张三",
  "phone": "13800138000",
  "email": "zhangsan@example.com",
  "id_card": "110***********1234",
  "status": 1
}

# 更新当前用户资料
PATCH /api/v1/me
Authorization: Bearer {token}
Content-Type: application/json

{
  "nickname": "张三丰",
  "contacts": [
    {"type": "phone", "value": "13900139000"},
    {"type": "email", "value": "zhangsan@newdomain.com"}
  ]
}

Response: 200 OK
{
  "id": "usr_1234567890",
  "name": "张三丰",
  "phone": "13900139000",
  "email": "zhangsan@newdomain.com",
  ...
}
```

#### 6.1.2 儿童档案 API

```http
# 注册儿童（自动建立监护关系）
POST /api/v1/children/register
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "小明",
  "gender": 1,
  "dob": "2020-05-15",
  "id_card": "110***********5678",
  "height_cm": 105,
  "weight_kg": 18.5
}

Response: 201 Created
{
  "id": "chd_9876543210",
  "name": "小明",
  "gender": 1,
  "dob": "2020-05-15",
  "height_cm": 105,
  "weight_kg": 18.5
}

# 查询儿童档案
GET /api/v1/children/{id}
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "chd_9876543210",
  "name": "小明",
  "gender": 1,
  "dob": "2020-05-15",
  "id_card": "110***********5678",
  "height_cm": 105,
  "weight_kg": 18.5
}

# 更新儿童档案
PATCH /api/v1/children/{id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "nickname": "小明明",
  "profile": {
    "gender": 1,
    "dob": "2020-05-15"
  },
  "height_weight": {
    "height_cm": 110,
    "weight_kg": 20
  }
}

Response: 200 OK
{...}

# 获取当前用户的儿童列表
GET /api/v1/me/children?offset=0&limit=20
Authorization: Bearer {token}

Response: 200 OK
{
  "total": 2,
  "limit": 20,
  "offset": 0,
  "items": [
    {"id": "chd_001", "name": "小明", ...},
    {"id": "chd_002", "name": "小红", ...}
  ]
}

# 搜索相似儿童（查重）
GET /api/v1/children/search?name=小明&dob=2020-05-15
Authorization: Bearer {token}

Response: 200 OK
{
  "total": 1,
  "items": [
    {"id": "chd_9876543210", "name": "小明", ...}
  ]
}
```

#### 6.1.3 监护关系 API

```http
# 授予监护权
POST /api/v1/guardians/grant
Authorization: Bearer {token}
Content-Type: application/json

{
  "child_id": "chd_9876543210",
  "relation": "parent"
}

Response: 201 Created
{
  "id": "gua_111222333",
  "user_id": "usr_1234567890",
  "child_id": "chd_9876543210",
  "relation": "parent",
  "granted_at": "2025-10-17T10:30:00Z"
}

# 注意：撤销监护权功能尚未实现

```

### 6.2 gRPC API

```protobuf
// api/grpc/identity.proto
syntax = "proto3";

package identity;

service IdentityRead {
  // 查询用户
  rpc GetUser(GetUserRequest) returns (User);
  
  // 查询儿童
  rpc GetChild(GetChildRequest) returns (Child);
}

service GuardianshipQuery {
  // 判断是否有监护关系
  rpc IsGuardian(IsGuardianRequest) returns (IsGuardianResponse);
  
  // 列出用户的所有儿童
  rpc ListChildren(ListChildrenRequest) returns (ListChildrenResponse);
}

message GetUserRequest {
  string user_id = 1;
}

message User {
  string id = 1;
  string name = 2;
  string phone = 3;
  string email = 4;
  int32 status = 5;
}

message IsGuardianRequest {
  string user_id = 1;
  string child_id = 2;
}

message IsGuardianResponse {
  bool is_guardian = 1;
  string relation = 2;
}
```

---
