# 🧪 API 测试指南

## 📋 目录

- [基础测试](#基础测试)
- [用户模块 (UC)](#用户模块-uc)
- [认证模块 (Authn)](#认证模块-authn)
- [授权模块 (Authz)](#授权模块-authz)
- [身份提供商 (IDP)](#身份提供商-idp)
- [使用工具测试](#使用工具测试)

---

## ✅ 前置条件

确保服务正在运行：

```bash
# 启动服务
make dev

# 验证服务状态
curl http://localhost:8080/healthz
```

---

## 🔍 基础测试

### 1. 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/healthz

# 详细健康检查
curl http://localhost:8080/health
```

### 2. 查看所有路由

```bash
# 获取所有注册的路由
curl http://localhost:8080/debug/routes | jq '.'

# 查看路由总数
curl http://localhost:8080/debug/routes | jq '.total'
```

### 3. 查看模块状态

```bash
# 查看所有模块初始化状态
curl http://localhost:8080/debug/modules | jq '.'
```

### 4. API 版本信息

```bash
# 获取系统信息
curl http://localhost:8080/api/v1/public/info | jq '.'

# 查看版本
curl http://localhost:8080/version | jq '.'
```

### 5. Swagger API 文档

在浏览器中打开：
```
http://localhost:8080/swagger/index.html
```

---

## 👤 用户模块 (UC) 测试

### 创建用户

```bash
# 创建普通用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "phone": "13800138001",
    "email": "zhangsan@example.com",
    "id_card": "110101199001011234"
  }' | jq '.'
```

### 查询用户

```bash
# 获取用户详情（需要替换实际的 userId）
curl http://localhost:8080/api/v1/users/{userId} | jq '.'

# 获取当前用户资料（需要 JWT Token）
curl http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '.'
```

### 更新用户

```bash
# 更新用户信息
curl -X PATCH http://localhost:8080/api/v1/users/{userId} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三（更新）",
    "email": "zhangsan_new@example.com"
  }' | jq '.'
```

### 儿童档案管理

```bash
# 创建儿童档案
curl -X POST http://localhost:8080/api/v1/children \
  -H "Content-Type: application/json" \
  -d '{
    "name": "小明",
    "gender": 1,
    "birthday": "2015-05-20",
    "id_card": "110101201505201234"
  }' | jq '.'

# 注册儿童（简化版）
curl -X POST http://localhost:8080/api/v1/children/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "小红",
    "gender": 2,
    "birthday": "2016-08-15"
  }' | jq '.'

# 查询儿童详情
curl http://localhost:8080/api/v1/children/{childId} | jq '.'

# 搜索儿童
curl "http://localhost:8080/api/v1/children/search?name=小明" | jq '.'

# 获取我监护的儿童列表（需要登录）
curl http://localhost:8080/api/v1/me/children \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '.'
```

### 监护关系管理

```bash
# 授予监护权
curl -X POST http://localhost:8080/api/v1/guardians/grant \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "USER_ID",
    "child_id": "CHILD_ID",
    "relation": "parent"
  }' | jq '.'

# 撤销监护权
curl -X POST http://localhost:8080/api/v1/guardians/revoke \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "USER_ID",
    "child_id": "CHILD_ID"
  }' | jq '.'

# 查询监护人列表
curl http://localhost:8080/api/v1/guardians | jq '.'
```

---

## 🔐 认证模块 (Authn) 测试

### 账号管理

```bash
# 创建操作账号（本地账号密码）
curl -X POST http://localhost:8080/api/v1/accounts/operation \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!",
    "user_id": "USER_ID"
  }' | jq '.'

# 查询操作账号
curl http://localhost:8080/api/v1/accounts/operation/admin | jq '.'

# 修改密码
curl -X POST http://localhost:8080/api/v1/accounts/operation/admin/change \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "Admin123!",
    "new_password": "NewPassword123!"
  }' | jq '.'
```

### 登录和认证

```bash
# 本地账号密码登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!",
    "login_type": "password"
  }' | jq '.'

# 保存返回的 access_token
export ACCESS_TOKEN="返回的token"

# 验证 Token
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "token": "'$ACCESS_TOKEN'"
  }' | jq '.'

# 刷新 Token
curl -X POST http://localhost:8080/api/v1/auth/refresh_token \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }' | jq '.'

# 登出
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
```

### JWKS 密钥管理

```bash
# 获取公钥（用于验签）
curl http://localhost:8080/.well-known/jwks.json | jq '.'

# 管理员查看所有密钥
curl http://localhost:8080/api/v1/admin/jwks/keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 查看可发布的公钥
curl http://localhost:8080/api/v1/admin/jwks/keys/publishable | jq '.'

# 生成新密钥
curl -X POST http://localhost:8080/api/v1/admin/jwks/keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "algorithm": "RS256",
    "key_size": 2048
  }' | jq '.'
```

---

## 🛡️ 授权模块 (Authz) 测试

### 资源管理

```bash
# 创建资源
curl -X POST http://localhost:8080/api/v1/authz/resources \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "key": "assessment:exam:001",
    "name": "心理测评考试001",
    "type": "assessment",
    "description": "青少年心理健康测评"
  }' | jq '.'

# 查询所有资源
curl http://localhost:8080/api/v1/authz/resources \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 通过 key 查询资源
curl "http://localhost:8080/api/v1/authz/resources/key/assessment:exam:001" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 查询单个资源
curl http://localhost:8080/api/v1/authz/resources/{resource_id} \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 验证操作是否有效
curl -X POST http://localhost:8080/api/v1/authz/resources/validate-action \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "resource_type": "assessment",
    "action": "read"
  }' | jq '.'
```

### 角色管理

```bash
# 创建角色
curl -X POST http://localhost:8080/api/v1/authz/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "name": "评估师",
    "code": "assessor",
    "description": "可以查看和评估测评结果",
    "is_system": false
  }' | jq '.'

# 查询所有角色
curl http://localhost:8080/api/v1/authz/roles \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 查询单个角色
curl http://localhost:8080/api/v1/authz/roles/{role_id} \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 更新角色
curl -X PUT http://localhost:8080/api/v1/authz/roles/{role_id} \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "name": "高级评估师",
    "description": "更新后的描述"
  }' | jq '.'
```

### 权限分配

```bash
# 授予角色
curl -X POST http://localhost:8080/api/v1/authz/assignments/grant \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "subject_id": "USER_ID",
    "subject_type": "user",
    "role_id": "ROLE_ID"
  }' | jq '.'

# 撤销角色
curl -X POST http://localhost:8080/api/v1/authz/assignments/revoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "assignment_id": "ASSIGNMENT_ID"
  }' | jq '.'

# 查询用户的角色分配
curl "http://localhost:8080/api/v1/authz/assignments/subject?subject_id=USER_ID&subject_type=user" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 查询角色的分配
curl http://localhost:8080/api/v1/authz/roles/{role_id}/assignments \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
```

### 策略管理

```bash
# 创建策略
curl -X POST http://localhost:8080/api/v1/authz/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "subject": "USER_ID",
    "resource": "assessment:exam:001",
    "action": "read"
  }' | jq '.'

# 查询策略版本
curl http://localhost:8080/api/v1/authz/policies/version \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 查看角色的策略
curl http://localhost:8080/api/v1/authz/roles/{role_id}/policies \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
```

---

## 🔑 身份提供商 (IDP) 测试

### 微信应用管理

```bash
# 注册微信小程序
curl -X POST http://localhost:8080/api/v1/idp/wechat-apps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "app_id": "wx1234567890abcdef",
    "app_name": "心理健康测评小程序",
    "app_secret": "your_app_secret_here"
  }' | jq '.'

# 查询微信应用
curl http://localhost:8080/api/v1/idp/wechat-apps/wx1234567890abcdef \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

# 刷新 Access Token
curl -X POST http://localhost:8080/api/v1/idp/wechat-apps/refresh-access-token \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "app_id": "wx1234567890abcdef"
  }' | jq '.'
```

### IDP 健康检查

```bash
curl http://localhost:8080/api/v1/idp/health | jq '.'
```

---

## 🧰 使用工具测试

### 1. 使用 cURL（推荐用于脚本）

创建测试脚本 `test-api.sh`：

```bash
#!/bin/bash

# 设置基础 URL
BASE_URL="http://localhost:8080"

# 1. 健康检查
echo "=== 健康检查 ==="
curl -s $BASE_URL/healthz | jq '.'

# 2. 创建用户
echo -e "\n=== 创建用户 ==="
USER_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试用户",
    "phone": "13900139001",
    "email": "test@example.com",
    "id_card": "110101199001011111"
  }')
echo $USER_RESPONSE | jq '.'
USER_ID=$(echo $USER_RESPONSE | jq -r '.data.id')

# 3. 创建账号
echo -e "\n=== 创建操作账号 ==="
curl -s -X POST $BASE_URL/api/v1/accounts/operation \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"testuser\",
    \"password\": \"Test123!\",
    \"user_id\": \"$USER_ID\"
  }" | jq '.'

# 4. 登录
echo -e "\n=== 登录 ==="
LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "Test123!",
    "login_type": "password"
  }')
echo $LOGIN_RESPONSE | jq '.'
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.access_token')

# 5. 使用 Token 访问受保护的接口
echo -e "\n=== 获取用户资料 ==="
curl -s $BASE_URL/api/v1/users/profile \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

echo -e "\n=== 测试完成 ==="
```

运行脚本：
```bash
chmod +x test-api.sh
./test-api.sh
```

### 2. 使用 Postman

1. 导入 Swagger 定义：
   - 打开 Postman
   - Import → Link → `http://localhost:8080/swagger/doc.json`

2. 设置环境变量：
   - `base_url`: `http://localhost:8080`
   - `access_token`: 登录后获取的 token

3. 在请求的 Headers 中添加：
   ```
   Authorization: Bearer {{access_token}}
   ```

### 3. 使用 HTTPie（更友好的 CLI 工具）

```bash
# 安装 HTTPie
brew install httpie  # macOS
# 或 pip install httpie

# 使用示例
# GET 请求
http GET localhost:8080/healthz

# POST 请求
http POST localhost:8080/api/v1/users \
  name="张三" \
  phone="13800138000" \
  email="test@example.com" \
  id_card="110101199001011234"

# 带认证的请求
http GET localhost:8080/api/v1/users/profile \
  "Authorization:Bearer $ACCESS_TOKEN"
```

### 4. 使用 VS Code REST Client 扩展

创建 `test.http` 文件：

```http
### 变量定义
@baseUrl = http://localhost:8080
@token = your_token_here

### 健康检查
GET {{baseUrl}}/healthz

### 创建用户
POST {{baseUrl}}/api/v1/users
Content-Type: application/json

{
  "name": "测试用户",
  "phone": "13900139000",
  "email": "test@example.com",
  "id_card": "110101199001011111"
}

### 登录
POST {{baseUrl}}/api/v1/auth/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "Test123!",
  "login_type": "password"
}

### 获取用户资料（需要先登录获取 token）
GET {{baseUrl}}/api/v1/users/profile
Authorization: Bearer {{token}}
```

点击 "Send Request" 即可测试。

---

## 📊 测试数据准备

### 加载种子数据

```bash
# 使用 seeddata 工具加载测试数据
go run ./cmd/tools/seeddata \
  --dsn "root:dev_root_123@tcp(localhost:3306)/iam_contracts?parseTime=true&loc=Local" \
  --redis "localhost:6379" \
  --redis-password "dev_cache_123" \
  --keys-dir "./tmp/keys" \
  --casbin-model "./configs/casbin_model.conf"
```

种子数据包含：
- 系统角色（管理员、普通用户等）
- 测试用户账号
- 权限资源定义
- 基础配置数据

---

## 🔍 调试技巧

### 查看详细日志

开发环境日志会输出到控制台，包含所有 SQL 查询和 Redis 操作。

### 使用调试端点

```bash
# 查看所有路由
curl http://localhost:8080/debug/routes | jq '.routes[] | select(.path | contains("user"))'

# 查看模块状态
curl http://localhost:8080/debug/modules | jq '.'
```

### 数据库直接查询

```bash
# 连接数据库
docker exec -it mysql mysql -uroot -pdev_root_123 iam_contracts

# 查看用户
SELECT * FROM iam_users LIMIT 5;

# 查看账号
SELECT * FROM iam_auth_accounts LIMIT 5;

# 查看角色
SELECT * FROM iam_authz_roles;
```

---

## ✅ 常见测试场景

### 场景 1: 完整的用户注册和登录流程

```bash
# 1. 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"李四","phone":"13900139002","email":"lisi@example.com","id_card":"110101199002021234"}' | jq '.'

# 2. 为用户创建登录账号
curl -X POST http://localhost:8080/api/v1/accounts/operation \
  -H "Content-Type: application/json" \
  -d '{"username":"lisi","password":"Lisi123!","user_id":"USER_ID"}' | jq '.'

# 3. 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"lisi","password":"Lisi123!","login_type":"password"}' | jq '.'

# 4. 使用 Token 访问受保护资源
curl http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '.'
```

### 场景 2: 监护人添加儿童并建立监护关系

```bash
# 1. 创建儿童档案
CHILD_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/children \
  -H "Content-Type: application/json" \
  -d '{"name":"小王","gender":1,"birthday":"2015-06-10"}')
CHILD_ID=$(echo $CHILD_RESPONSE | jq -r '.data.id')

# 2. 建立监护关系
curl -X POST http://localhost:8080/api/v1/guardians/grant \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"USER_ID\",\"child_id\":\"$CHILD_ID\",\"relation\":\"parent\"}" | jq '.'

# 3. 查看我的儿童
curl http://localhost:8080/api/v1/me/children \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '.'
```

### 场景 3: 权限管理完整流程

```bash
# 1. 创建资源
RESOURCE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/authz/resources \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"key":"test:resource:001","name":"测试资源","type":"test"}')
RESOURCE_ID=$(echo $RESOURCE_RESPONSE | jq -r '.data.id')

# 2. 创建角色
ROLE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/authz/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"测试角色","code":"test_role","description":"测试用角色"}')
ROLE_ID=$(echo $ROLE_RESPONSE | jq -r '.data.id')

# 3. 授予用户角色
curl -X POST http://localhost:8080/api/v1/authz/assignments/grant \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{\"subject_id\":\"$USER_ID\",\"subject_type\":\"user\",\"role_id\":\"$ROLE_ID\"}" | jq '.'

# 4. 查看用户权限
curl "http://localhost:8080/api/v1/authz/assignments/subject?subject_id=$USER_ID&subject_type=user" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'
```

---

## 📚 相关资源

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **调试路由**: http://localhost:8080/debug/routes
- **模块状态**: http://localhost:8080/debug/modules
- **JWKS 公钥**: http://localhost:8080/.well-known/jwks.json

---

**祝测试顺利！** 🎉
