# 用户中心 - API 设计

> [返回用户中心文档](./README.md)

本文档详细介绍用户中心的 RESTful API 和 gRPC API 设计。

---

## 6. API 设计

### 6.1 RESTful API

#### 6.1.1 用户管理 API

```http
# 创建用户
POST /api/v1/users
Content-Type: application/json

{
  "nickname": "张三",
  "contacts": [
    {"type": "phone", "value": "13800138000"},
    {"type": "email", "value": "zhangsan@example.com"}
  ]
}

Response: 201 Created
{
  "id": "usr_1234567890",
  "name": "张三",
  "phone": "13800138000",
  "email": "zhangsan@example.com",
  "status": 1
}

# 查询用户
GET /api/v1/users/{userId}

Response: 200 OK
{
  "id": "usr_1234567890",
  "name": "张三",
  "phone": "13800138000",
  "email": "zhangsan@example.com",
  "id_card": "110***********1234",
  "status": 1
}

# 更新用户资料
PATCH /api/v1/users/{userId}
Content-Type: application/json

{
  "nickname": "张三丰",
  "contacts": [
    {"type": "phone", "value": "13900139000"}
  ]
}

Response: 200 OK
{
  "id": "usr_1234567890",
  "name": "张三丰",
  "phone": "13900139000",
  ...
}

# 获取当前用户资料
GET /api/v1/profile
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "usr_1234567890",
  "name": "张三",
  ...
}
```

#### 6.1.2 儿童档案 API

```http
# 注册儿童（带监护关系）
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
  "gender": "male",
  "dob": "2020-05-15",
  "height_cm": 105,
  "weight_kg": 18.5
}

# 创建儿童档案（不建立监护关系）
POST /api/v1/children
Content-Type: application/json

{
  "name": "小红",
  "gender": 2,
  "dob": "2021-03-20"
}

Response: 201 Created
{...}

# 查询儿童档案
GET /api/v1/children/{childId}

Response: 200 OK
{
  "id": "chd_9876543210",
  "name": "小明",
  "gender": "male",
  "dob": "2020-05-15",
  "id_card": "110***********5678",
  "height_cm": 105,
  "weight_kg": 18.5
}

# 更新儿童档案
PATCH /api/v1/children/{childId}
Content-Type: application/json

{
  "gender": 1,
  "dob": "2020-05-15",
  "height_cm": 110,
  "weight_kg": 20
}

Response: 200 OK
{...}

# 获取我的儿童列表
GET /api/v1/children/me?offset=0&limit=20
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
POST /api/v1/guardianships
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

# 撤销监护权
DELETE /api/v1/guardianships/{guardianshipId}
Authorization: Bearer {token}

Response: 204 No Content

# 查询监护关系
GET /api/v1/guardianships?user_id={userId}&child_id={childId}

Response: 200 OK
{
  "total": 1,
  "items": [
    {
      "id": "gua_111222333",
      "user": {"id": "usr_1234567890", "name": "张三"},
      "child": {"id": "chd_9876543210", "name": "小明"},
      "relation": "parent",
      "granted_at": "2025-10-17T10:30:00Z"
    }
  ]
}
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
