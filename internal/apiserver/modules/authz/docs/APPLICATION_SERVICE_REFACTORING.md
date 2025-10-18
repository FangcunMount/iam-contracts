# 应用层服务重构总结

**日期**: 2025年10月18日  
**重构内容**: 统一使用项目标准错误处理机制

---

## 📋 重构概述

将应用层服务中自定义的 `apperrors` 包替换为项目统一的错误处理机制：
- 使用 `pkg/errors` 包的 `WithCode()` 和 `WrapC()` 函数
- 使用 `internal/pkg/code` 包中注册的错误码
- 删除自定义的 `application/apperrors` 包

---

## 🔧 主要变更

### 1. 错误码定义 (`internal/pkg/code/authz.go`)

新增授权模块相关错误码：

```go
const (
	// ErrRoleNotFound - 404: Role not found.
	ErrRoleNotFound = 102001

	// ErrRoleAlreadyExists - 409: Role already exists.
	ErrRoleAlreadyExists = 102002

	// ErrResourceNotFound - 404: Resource not found.
	ErrResourceNotFound = 102003

	// ErrResourceAlreadyExists - 409: Resource already exists.
	ErrResourceAlreadyExists = 102004

	// ErrAssignmentNotFound - 404: Assignment not found.
	ErrAssignmentNotFound = 102005

	// ErrInvalidAction - 400: Invalid action for resource.
	ErrInvalidAction = 102006

	// ErrPolicyVersionNotFound - 404: Policy version not found.
	ErrPolicyVersionNotFound = 102007
)
```

**特点**：
- 错误码范围：102xxx (授权模块专用)
- 自动映射HTTP状态码 (404, 409, 400等)
- 支持国际化消息

---

### 2. 错误处理模式

#### Before (旧方式 - 自定义错误)
```go
// 定义
func NewBadRequest(message string) error {
    return &AppError{Type: ErrBadRequest, Message: message}
}

// 使用
if cmd.Name == "" {
    return nil, apperrors.NewBadRequest("角色名称不能为空")
}

// 判断
if apperrors.IsNotFound(err) {
    // handle not found
}
```

#### After (新方式 - 统一错误码)
```go
// 使用
if cmd.Name == "" {
    return nil, errors.WithCode(code.ErrInvalidArgument, "角色名称不能为空")
}

// 判断
if errors.IsCode(err, code.ErrRoleNotFound) {
    // handle not found
}

// 包装错误
if err != nil {
    return nil, errors.Wrap(err, "获取角色失败")
}
```

---

## 📦 更新的服务

### 1. RoleService (`application/role/service.go`)

**错误码映射**：
| 原错误类型 | 新错误码 | HTTP状态码 |
|-----------|---------|-----------|
| BadRequest | ErrInvalidArgument | 400 |
| NotFound | ErrRoleNotFound | 404 |
| Conflict | ErrRoleAlreadyExists | 409 |
| Forbidden | ErrPermissionDenied | 403 |

**关键方法**：
- `CreateRole`: 创建角色，检查名称冲突
- `UpdateRole`: 更新角色信息
- `DeleteRole`: 删除角色，租户隔离检查
- `GetRoleByID/GetRoleByName`: 查询角色
- `ListRoles`: 分页列表

---

### 2. AssignmentService (`application/assignment/service.go`)

**错误码映射**：
| 操作 | 错误码 | 说明 |
|------|--------|------|
| 角色不存在 | ErrRoleNotFound | 授权时角色检查 |
| 赋权记录不存在 | ErrAssignmentNotFound | 撤销时记录检查 |
| 跨租户操作 | ErrPermissionDenied | 租户隔离 |
| 参数验证 | ErrInvalidArgument | 必填字段检查 |

**关键方法**：
- `Grant`: 授权（MySQL + Casbin g规则）
- `Revoke/RevokeByID`: 撤销授权（事务回滚保护）
- `ListBySubject/ListByRole`: 查询赋权关系

**事务保护**：
```go
// 添加 Casbin 规则
if err := s.casbinPort.AddGroupingPolicy(ctx, groupingRule); err != nil {
    // 回滚：删除数据库记录
    _ = s.assignmentRepo.Delete(ctx, newAssignment.ID)
    return nil, errors.Wrap(err, "添加 Casbin 规则失败")
}
```

---

### 3. PolicyService (`application/policy/service.go`)

**错误码映射**：
| 操作 | 错误码 | 说明 |
|------|--------|------|
| 角色不存在 | ErrRoleNotFound | 策略规则关联角色 |
| 资源不存在 | ErrResourceNotFound | 策略规则关联资源 |
| Action无效 | ErrInvalidAction | 动作不在资源允许列表 |
| 版本不存在 | ErrPolicyVersionNotFound | 版本查询 |

**关键方法**：
- `AddPolicyRule`: 添加策略规则 + 版本递增 + Redis通知
- `RemovePolicyRule`: 移除策略规则 + 版本递增 + Redis通知
- `GetPoliciesByRole`: 查询角色的策略规则
- `GetCurrentVersion`: 获取当前版本号

**版本管理**：
```go
// 递增版本号
newVersion, err := s.policyVersionRepo.Increment(ctx, cmd.TenantID, cmd.ChangedBy, cmd.Reason)
if err != nil {
    log.Errorf("递增策略版本失败: %v", err)
    // 不阻塞主流程，只记录日志
}

// 发布版本变更通知
if newVersion != nil {
    if err := s.versionNotifier.Publish(ctx, cmd.TenantID, newVersion.Version); err != nil {
        log.Errorf("发布版本变更通知失败: %v", err)
    }
}
```

---

### 4. ResourceService (`application/resource/service.go`)

**错误码映射**：
| 操作 | 错误码 | HTTP状态码 |
|------|--------|-----------|
| 资源不存在 | ErrResourceNotFound | 404 |
| 资源已存在 | ErrResourceAlreadyExists | 409 |
| 参数无效 | ErrInvalidArgument | 400 |

**关键方法**：
- `CreateResource`: 创建资源目录
- `UpdateResource`: 更新资源（DisplayName, Actions, Description）
- `DeleteResource`: 删除资源
- `GetResourceByID/GetResourceByKey`: 查询资源
- `ListResources`: 分页列表
- `ValidateAction`: 验证动作合法性

---

### 5. VersionService (`application/version/service.go`)

**错误码映射**：
| 操作 | 错误码 | 说明 |
|------|--------|------|
| 版本不存在 | ErrPolicyVersionNotFound | 自动创建初始版本 |
| 参数无效 | ErrInvalidArgument | TenantID必填 |

**关键方法**：
- `GetCurrentVersion`: 获取当前版本（不存在则创建）
- `GetOrCreateVersion`: 确保版本记录存在

---

## ✨ 优势总结

### 1. 统一的错误处理
- **Before**: 5个服务使用自定义 `apperrors` 包，不一致
- **After**: 统一使用 `pkg/errors` + `internal/pkg/code`

### 2. 标准化的HTTP状态码
```go
// 自动映射
ErrRoleNotFound        → 404 Not Found
ErrRoleAlreadyExists   → 409 Conflict
ErrInvalidArgument     → 400 Bad Request
ErrPermissionDenied    → 403 Forbidden
```

### 3. 错误链追踪
```go
// 包装错误，保留调用栈
if err != nil {
    return nil, errors.Wrap(err, "获取角色失败")
}

// 输出时自动包含完整堆栈信息
fmt.Printf("%+v\n", err)
```

### 4. 错误码判断
```go
// 精确判断错误类型
if errors.IsCode(err, code.ErrRoleNotFound) {
    // 404 处理
} else if errors.IsCode(err, code.ErrPermissionDenied) {
    // 403 处理
}
```

---

## 📁 文件清单

### 修改的文件
```
internal/pkg/code/authz.go                         (新增8个错误码)
internal/apiserver/modules/authz/application/
├── role/service.go                                (重写)
├── assignment/service.go                          (重写)
├── policy/service.go                              (重写)
├── resource/service.go                            (重写)
└── version/service.go                             (重写)
```

### 删除的文件
```
internal/apiserver/modules/authz/application/
└── apperrors/errors.go                            (删除)
```

---

## ✅ 验证结果

```bash
# 编译检查
$ go build ./internal/apiserver/modules/authz/application/...
✅ 成功

# 代码质量检查
$ go vet ./internal/apiserver/modules/authz/...
✅ 无警告

# 依赖项
import (
    "github.com/fangcun-mount/iam-contracts/internal/pkg/code"
    "github.com/fangcun-mount/iam-contracts/pkg/errors"
)
```

---

## 🎯 使用示例

### 创建带错误码的错误
```go
// 参数验证错误
if cmd.Name == "" {
    return errors.WithCode(code.ErrInvalidArgument, "角色名称不能为空")
}

// 资源不存在错误
if err != nil {
    return errors.WithCode(code.ErrRoleNotFound, "角色 %d 不存在", roleID)
}
```

### 包装底层错误
```go
// 保留原始错误信息
role, err := s.roleRepo.FindByID(ctx, roleID)
if err != nil {
    return nil, errors.Wrap(err, "获取角色失败")
}
```

### 错误判断
```go
// 精确匹配错误码
err := s.roleRepo.FindByID(ctx, roleID)
if errors.IsCode(err, code.ErrRoleNotFound) {
    // 404 Not Found
}
```

### REST API 层错误处理
```go
func (h *RoleHandler) GetRole(c *gin.Context) {
    role, err := h.roleService.GetRoleByID(ctx, roleID, tenantID)
    if err != nil {
        coder := errors.ParseCoder(err)
        c.JSON(coder.HTTPStatus(), gin.H{
            "code":    coder.Code(),
            "message": coder.String(),
        })
        return
    }
    c.JSON(200, role)
}
```

---

## 🚀 后续步骤

现在所有应用层服务已完成，可以继续：
1. ✅ 创建 REST API 处理器 (PAP)
2. ✅ 创建 PEP SDK (DomainGuard)
3. ✅ 集成测试
4. ✅ API 文档

---

**总结**: 通过统一错误处理机制，应用层服务代码更加规范、可维护性更强，为后续的 REST API 开发和错误响应处理奠定了坚实基础。
