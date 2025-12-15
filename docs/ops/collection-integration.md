# Collection 服务集成说明

## 概述

在执行 `seed_family` 数据生成任务时，系统会自动调用 Collection 服务的 API，为每个创建的儿童（Child）生成对应的受试者（Testee）记录。

## 配置说明

在 `configs/seeddata.yaml` 中添加 Collection 服务配置：

```yaml
# ==================== 11. Collection 服务配置 ====================
collection_url: "http://localhost:18083/api/v1"  # Collection 服务 API 地址
collection_auth:
  admin_token: "your-admin-token-here"  # 管理员 Token
  use_login: false  # 是否使用用户登录获取 token（暂不支持）
```

### 配置项说明

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `collection_url` | string | 否 | Collection 服务的 API 基础 URL。如果为空，将跳过受试者创建 |
| `collection_auth.admin_token` | string | 否 | 管理员 Token，用于调用 Collection API。如果为空，将跳过受试者创建 |
| `collection_auth.use_login` | bool | 否 | 是否使用用户登录获取 token（暂不支持，需先创建认证账号） |

## 工作流程

1. **创建家庭数据**
   - 生成父母用户信息
   - 生成儿童信息

2. **创建 IAM 数据**
   - 在 IAM 系统中创建父母用户（User）
   - 在 IAM 系统中创建儿童档案（Child）
   - 建立监护关系（Guardianship）

3. **创建受试者数据**
   - 调用 Collection 服务 API
   - 为每个儿童创建对应的受试者（Testee）记录
   - 关联 IAM Child ID

## API 调用详情

### 请求格式

```http
POST {collection_url}/testees
Content-Type: application/json
Authorization: Bearer {admin_token}

{
  "iam_child_id": "123456789",
  "name": "张小明",
  "gender": 1,
  "birthday": "2015-06-15",
  "source": "imported"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `iam_child_id` | string | IAM 系统中的儿童 ID |
| `name` | string | 儿童姓名 |
| `gender` | int | 性别（1=男，2=女） |
| `birthday` | string | 出生日期（格式：YYYY-MM-DD） |
| `source` | string | 数据来源，固定为 "imported" |

### 响应处理

- **成功**：HTTP 200/201，受试者创建成功
- **失败**：记录警告日志，但不阻断整体流程

## 使用示例

### 1. 配置 Collection 服务

编辑 `configs/seeddata.yaml`：

```yaml
collection_url: "http://localhost:18083/api/v1"
collection_auth:
  admin_token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 2. 执行 seed 命令

```bash
go run ./cmd/tools/seeddata \
  --dsn "user:pass@tcp(localhost:3306)/iam_contracts?parseTime=true" \
  --config configs/seeddata.yaml \
  --steps family \
  --family-count 1000 \
  --worker-count 20
```

### 3. 查看执行结果

执行过程中，如果受试者创建失败，会输出警告信息：

```
Warning: task 123: create testee for child 0 failed: collection API returned status 401: ...
```

但不会中断整体流程，会继续创建其他家庭数据。

## 获取 Admin Token

### 方法 1：从 Collection 服务获取

如果 Collection 服务提供了管理员登录接口：

```bash
curl -X POST http://localhost:18083/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'
```

### 方法 2：从 IAM 系统获取

如果使用 IAM 系统的 token：

```bash
# 使用 IAM SDK 或 API 获取 token
# 需要有足够的权限调用 Collection 服务
```

## 故障排查

### 问题：受试者创建失败，返回 401

**原因**：Admin Token 无效或过期

**解决**：
1. 检查 `collection_auth.admin_token` 配置是否正确
2. 重新获取有效的 Admin Token
3. 如果 token 为空，确认是否需要创建受试者

### 问题：受试者创建失败，返回 400

**原因**：请求参数不符合 Collection API 要求

**解决**：
1. 检查 Collection API 文档，确认请求格式
2. 查看错误响应中的具体错误信息
3. 根据需要调整 `createTestee` 函数中的请求参数

### 问题：Collection 服务连接超时

**原因**：Collection 服务未启动或网络不通

**解决**：
1. 确认 Collection 服务已启动并运行在配置的地址
2. 检查防火墙和网络配置
3. 临时解决：将 `collection_url` 设置为空，跳过受试者创建

## 注意事项

1. **Token 安全性**
   - 不要在版本控制中提交包含真实 token 的配置文件
   - 使用环境变量或密钥管理系统存储敏感信息

2. **错误处理**
   - 受试者创建失败不会影响 IAM 数据创建
   - 可以稍后通过其他方式补充创建受试者数据

3. **性能考虑**
   - 每个儿童都会发起一次 HTTP 请求
   - 大批量数据生成时注意 Collection 服务的性能和限流

4. **数据一致性**
   - IAM Child ID 是唯一标识，确保 Collection 服务正确处理
   - 如果受试者已存在，Collection API 应返回适当的错误码

## 未来改进

1. **用户登录认证**
   - 支持使用父母账号登录获取 token
   - 需要先在 seed 过程中创建认证账号

2. **批量创建 API**
   - 支持批量创建受试者，减少 HTTP 请求次数
   - 提高大规模数据生成的效率

3. **重试机制**
   - 对于网络临时故障，自动重试
   - 记录失败的儿童 ID，便于后续补充

4. **数据同步**
   - 提供工具检查 IAM Child 和 Collection Testee 的数据一致性
   - 支持增量同步
