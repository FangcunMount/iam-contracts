# REST API 实现总结

## 概览

成功为 IAM 授权系统的 PAP (Policy Administration Point) 实现了完整的 REST API 层，包括 DTO 定义和 Handler 实现。

## 完成的工作

### 1. DTO 层 (5个文件)

#### common.go - 通用响应结构

- **Response**: 标准成功响应 `{code, message, data}`
- **ListResponse**: 分页列表响应 `{code, message, data, total, offset, limit}`
- **ErrorResponse**: 错误响应 `{code, message, error}`
- **辅助函数**: NewResponse(), NewListResponse(), NewErrorResponse()

#### role.go - 角色管理 DTO

- **CreateRoleRequest**: 创建角色请求 (name, display_name, description)
- **UpdateRoleRequest**: 更新角色请求 (display_name, description)
- **RoleResponse**: 角色响应 (id, name, display_name, tenant_id, description)
- **ListRoleQuery**: 角色列表查询 (offset, limit)

#### assignment.go - 角色分配 DTO

- **GrantRequest**: 授予角色请求 (subject_type: user/group, subject_id, role_id)
- **RevokeRequest**: 撤销角色请求 (subject_type, subject_id, role_id)
- **AssignmentResponse**: 分配响应 (id, subject_type, subject_id, role_id, tenant_id, granted_by)
- **验证标签**: `binding:"required,oneof=user group"` 确保主体类型正确

#### policy.go - 策略管理 DTO

- **AddPolicyRequest**: 添加策略请求 (role_id, resource_id, action, changed_by, reason)
- **RemovePolicyRequest**: 移除策略请求 (role_id, resource_id, action, changed_by, reason)
- **PolicyRuleResponse**: 策略规则响应 (subject, domain, object, action)
- **PolicyVersionResponse**: 策略版本响应 (tenant_id, version, changed_by, reason)

#### resource.go - 资源管理 DTO

- **CreateResourceRequest**: 创建资源请求 (key, display_name, app_name, domain, type, actions[])
- **UpdateResourceRequest**: 更新资源请求 (display_name, actions, description)
- **ResourceResponse**: 资源响应 (id, key, display_name, app_name, domain, type, actions, description)
- **ValidateActionRequest/Response**: 验证动作请求/响应 (resource_key, action) -> (valid)

### 2. Handler 层 (5个文件)

#### base.go - 通用工具函数
```go
func getTenantID(c *gin.Context) string
    // 从 X-Tenant-ID 头提取租户ID，默认 "default"
    
func getUserID(c *gin.Context) string
    // 从 X-User-ID 头提取用户ID，默认 "system"
    
func handleError(c *gin.Context, err error)
    // 使用 errors.ParseCoder() 解析错误码，返回标准 JSON 错误响应
    
func success(c *gin.Context, data interface{})
    // 返回 200 + 数据
    
func successList(c *gin.Context, data interface{}, total int64, offset, limit int)
    // 返回 200 + 分页数据
    
func successNoContent(c *gin.Context)
    // 返回 200 无数据
```

#### role.go - RoleHandler (6个方法)

- **CreateRole**: POST /authz/roles - 创建角色
- **UpdateRole**: PUT /authz/roles/{id} - 更新角色
- **DeleteRole**: DELETE /authz/roles/{id} - 删除角色
- **GetRole**: GET /authz/roles/{id} - 获取角色详情
- **ListRoles**: GET /authz/roles - 列出角色 (支持分页)
- **toRoleResponse**: 领域对象 → DTO 转换

**关键实现**:

- 使用 `getTenantID(c)` 实现租户隔离
- 使用 `handleError(c, err)` 统一错误处理
- ID 解析: `strconv.ParseUint(c.Param("id"), 10, 64)`

#### assignment.go - AssignmentHandler (5个方法)

- **GrantRole**: POST /authz/assignments/grant - 授予角色
- **RevokeRole**: POST /authz/assignments/revoke - 撤销角色
- **RevokeRoleByID**: DELETE /authz/assignments/{id} - 根据ID撤销
- **ListAssignmentsBySubject**: GET /authz/assignments/subject - 列出主体的分配
- **ListAssignmentsByRole**: GET /authz/roles/{role_id}/assignments - 列出角色的分配
- **convertToSubjectType**: 字符串 → SubjectType 转换 (user/group)

**关键实现**:

- 主体类型转换: `convertToSubjectType(req.SubjectType)` 确保类型安全
- 使用 `getUserID(c)` 记录授权人
- 命令模式: 所有修改操作使用 Command 对象

#### policy.go - PolicyHandler (4个方法)

- **AddPolicyRule**: POST /authz/policies - 添加策略规则
- **RemovePolicyRule**: DELETE /authz/policies - 移除策略规则
- **GetPoliciesByRole**: GET /authz/roles/{role_id}/policies - 获取角色策略
- **GetCurrentVersion**: GET /authz/policies/version - 获取当前策略版本

**关键实现**:

- 版本查询: 使用 `GetCurrentVersionQuery{TenantID: tenantID}`
- 策略规则响应: Casbin (Sub, Dom, Obj, Act) → DTO (Subject, Domain, Object, Action)
- 记录变更原因: ChangedBy + Reason 用于审计

#### resource.go - ResourceHandler (8个方法)

- **CreateResource**: POST /authz/resources - 创建资源
- **UpdateResource**: PUT /authz/resources/{id} - 更新资源
- **DeleteResource**: DELETE /authz/resources/{id} - 删除资源
- **GetResource**: GET /authz/resources/{id} - 获取资源详情
- **GetResourceByKey**: GET /authz/resources/key/{key} - 根据键获取资源
- **ListResources**: GET /authz/resources - 列出资源 (支持分页)
- **ValidateAction**: POST /authz/resources/validate-action - 验证资源动作
- **toResourceResponse**: 领域对象 → DTO 转换

**关键实现**:

- 双路由支持: 按ID查询 `/resources/{id}` 和按键查询 `/resources/key/{key}`
- 动作验证: `ValidateActionQuery{ResourceKey, Action}` → `{Valid: bool}`
- 资源目录: 支持 AppName, Domain, Type 字段但当前查询不过滤

## 技术亮点

### 1. 租户隔离
```go
tenantID := getTenantID(c)
// 从 X-Tenant-ID 头提取，所有 Service 层调用都传入 tenantID
```

### 2. 统一错误处理
```go
handleError(c, err)
// 自动解析 errors.WithCode() 设置的错误码
// 映射到 HTTP 状态码: 404 (NotFound), 409 (AlreadyExists), 400 (InvalidArgument), 500 (Internal)
```

### 3. 类型安全转换
```go
// Assignment: string → SubjectType
convertToSubjectType("user") → SubjectTypeUser
convertToSubjectType("group") → SubjectTypeGroup

// Policy: uint64 → ResourceID
resource.NewResourceID(req.ResourceID)

// Role: uint64 → RoleID  
domainRole.NewRoleID(roleID)
```

### 4. Swagger 注解
所有 Handler 方法包含完整的 Swagger 注解:
```go
// @Summary 创建角色
// @Tags Role
// @Accept json
// @Produce json
// @Param request body dto.CreateRoleRequest true "创建角色请求"
// @Success 200 {object} dto.Response{data=dto.RoleResponse}
// @Router /authz/roles [post]
```

### 5. Gin 验证标签
```go
type GrantRequest struct {
    SubjectType string `json:"subject_type" binding:"required,oneof=user group"`
    SubjectID   string `json:"subject_id" binding:"required"`
    RoleID      uint64 `json:"role_id" binding:"required"`
}
// binding 标签由 Gin 自动验证，失败返回 400
```

## 编译验证

```bash
$ go build ./internal/apiserver/modules/authz/interface/restful/handler/...
# 成功编译，无错误
```

## 待完成工作

### 1. 路由注册 (高优先级)
需要在 `internal/apiserver/routers.go` 中注册所有路由:

```go
// 角色管理
authzGroup.POST("/roles", roleHandler.CreateRole)
authzGroup.PUT("/roles/:id", roleHandler.UpdateRole)
authzGroup.DELETE("/roles/:id", roleHandler.DeleteRole)
authzGroup.GET("/roles/:id", roleHandler.GetRole)
authzGroup.GET("/roles", roleHandler.ListRoles)

// 角色分配
authzGroup.POST("/assignments/grant", assignmentHandler.GrantRole)
authzGroup.POST("/assignments/revoke", assignmentHandler.RevokeRole)
authzGroup.DELETE("/assignments/:id", assignmentHandler.RevokeRoleByID)
authzGroup.GET("/assignments/subject", assignmentHandler.ListAssignmentsBySubject)
authzGroup.GET("/roles/:role_id/assignments", assignmentHandler.ListAssignmentsByRole)

// 策略管理
authzGroup.POST("/policies", policyHandler.AddPolicyRule)
authzGroup.DELETE("/policies", policyHandler.RemovePolicyRule)
authzGroup.GET("/roles/:role_id/policies", policyHandler.GetPoliciesByRole)
authzGroup.GET("/policies/version", policyHandler.GetCurrentVersion)

// 资源管理
authzGroup.POST("/resources", resourceHandler.CreateResource)
authzGroup.PUT("/resources/:id", resourceHandler.UpdateResource)
authzGroup.DELETE("/resources/:id", resourceHandler.DeleteResource)
authzGroup.GET("/resources/:id", resourceHandler.GetResource)
authzGroup.GET("/resources/key/:key", resourceHandler.GetResourceByKey)
authzGroup.GET("/resources", resourceHandler.ListResources)
authzGroup.POST("/resources/validate-action", resourceHandler.ValidateAction)
```

### 2. Handler 实例化
在 `container/assembler` 中创建 Handler 实例:

```go
roleHandler := handler.NewRoleHandler(roleService)
assignmentHandler := handler.NewAssignmentHandler(assignmentService)
policyHandler := handler.NewPolicyHandler(policyService)
resourceHandler := handler.NewResourceHandler(resourceService)
```

### 3. PEP SDK
创建策略执行点 SDK，提供业务服务使用的权限检查 API

### 4. 集成测试
测试完整的 REST API 流程，包括租户隔离、错误码映射、Casbin 同步等

## 文件清单

```
internal/apiserver/modules/authz/interface/restful/
├── dto/
│   ├── common.go       (60 lines) - 通用响应结构
│   ├── role.go         (30 lines) - 角色 DTO
│   ├── assignment.go   (25 lines) - 分配 DTO
│   ├── policy.go       (40 lines) - 策略 DTO
│   └── resource.go     (50 lines) - 资源 DTO
└── handler/
    ├── base.go         (70 lines) - 通用工具函数
    ├── role.go         (180 lines) - RoleHandler
    ├── assignment.go   (240 lines) - AssignmentHandler
    ├── policy.go       (170 lines) - PolicyHandler
    └── resource.go     (260 lines) - ResourceHandler
```

**总计**: 10 个文件, ~1,125 行代码

## 架构符合度

✅ **端口适配器模式**: Handler 作为 REST 入站适配器  
✅ **领域驱动设计**: DTO ↔ Domain 对象分离  
✅ **CQRS**: Command (Create/Update/Delete) vs Query (Get/List)  
✅ **依赖注入**: Handler 依赖 Service 接口  
✅ **错误处理**: 统一使用 pkg/errors + internal/pkg/code  
✅ **租户隔离**: 所有请求通过 X-Tenant-ID 头隔离  

## 下一步

继续实现 todolist 第 3 项: **注册路由到 router.go**
