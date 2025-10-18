# 错误码重构总结报告

## 重构目标

对 `internal/pkg/code` 下的错误码进行重新组织，遵循以下原则：

1. **按模块分离**: 每个模块的错误码在自己的文件中定义和注册
2. **移除冗余**: 删除未使用的错误码
3. **清晰职责**: 避免交叉注册（如 authn 错误码在 base.go 中注册）

## 重构前的问题

### 1. 交叉注册问题

- **authn.go** 中定义的错误码在 **base.go** 中注册
- **authz.go** 中定义的错误码也在 **base.go** 中注册
- 导致代码不易维护，职责不清晰

### 2. 未使用的错误码

- **ErrTokenGeneration** (100208) - 完全未使用，已删除

### 3. 注册位置混乱
```
authn.go (定义)     →     base.go (注册)     ❌ 不合理
authz.go (定义)     →     base.go (注册)     ❌ 不合理
identity.go (定义)  →     无注册             ❌ Bug (已在上个版本修复)
```

## 重构方案

### 文件职责重新划分

#### 1. base.go - 基础通用错误码
**职责**: 只注册真正通用的、跨模块的基础错误码

**保留的错误码**:
```go
// HTTP 基础错误
ErrSuccess, ErrUnknown, ErrBind, ErrValidation, ErrPageNotFound
ErrInvalidArgument, ErrInvalidMessage

// 数据库错误
ErrDatabase

// 编码/解码错误  
ErrEncodingFailed, ErrDecodingFailed
ErrInvalidJSON, ErrEncodingJSON, ErrDecodingJSON
ErrInvalidYaml, ErrEncodingYaml, ErrDecodingYaml

// 模块错误
ErrModuleInitializationFailed, ErrModuleNotFound

// 内部服务器错误
ErrInternalServerError

// 通用认证授权错误
ErrUnauthenticated, ErrUnauthorized, ErrInvalidCredentials
```

**移除的错误码** (转移到其他文件):

- ❌ ErrEncrypt → 转移到 authn.go
- ❌ ErrSignatureInvalid → 转移到 authn.go
- ❌ ErrExpired → 转移到 authn.go
- ❌ ErrInvalidAuthHeader → 转移到 authn.go
- ❌ ErrMissingHeader → 转移到 authn.go
- ❌ ErrPasswordIncorrect → 转移到 authn.go
- ❌ ErrPermissionDenied → 转移到 authz.go

#### 2. authn.go - 认证模块错误码
**职责**: 认证相关的所有错误码，自己定义，自己注册

**重构后**:
```go
const (
    ErrTokenInvalid = 100005
    ErrEncrypt = 100201
    ErrSignatureInvalid = 100202
    ErrExpired = 100203
    ErrInvalidAuthHeader = 100204
    ErrMissingHeader = 100205
    ErrPasswordIncorrect = 100206
)

func init() {
    registerAuthn()
}

func registerAuthn() {
    errors.MustRegister(&authnCoder{...})
    // 注册所有 7 个认证错误码
}
```

**删除的错误码**:

- ❌ ErrTokenGeneration (100208) - 未使用

#### 3. authz.go - 授权模块错误码
**职责**: 授权相关的错误码，自己定义，自己注册

**重构后**:
```go
const (
    ErrPermissionDenied = 100207
)

func init() {
    registerAuthz()
}

func registerAuthz() {
    errors.MustRegister(&authzCoder{
        code: ErrPermissionDenied, 
        status: http.StatusForbidden, 
        msg: "Permission denied"
    })
}
```

**新增**: 完整的注册函数和 Coder 实现

#### 4. identity.go - 用户身份模块错误码
**职责**: 用户、儿童、监护关系相关错误码

**保持不变** (上个版本已修复):
```go
const (
    // User errors
    ErrUserNotFound = 110001
    ErrUserAlreadyExists = 110002
    ErrUserBasicInfoInvalid = 110003
    ErrUserStatusInvalid = 110004
    ErrUserInvalid = 110005
    ErrUserBlocked = 110006
    ErrUserInactive = 110007
    
    // Identity errors
    ErrIdentityUserBlocked = 110101
    ErrIdentityChildExists = 110102
    ErrIdentityChildNotFound = 110103
    ErrIdentityGuardianshipExists = 110104
    ErrIdentityGuardianshipNotFound = 110105
)

func init() {
    registerIdentity()
}
```

## 重构效果对比

### 重构前: base.go
```go
func init() {
    // 注册基础错误码
    registerBase(ErrSuccess, ...)
    registerBase(ErrUnknown, ...)
    
    // ❌ 混杂了 authn 模块的错误码
    registerBase(ErrEncrypt, ...)
    registerBase(ErrExpired, ...)
    registerBase(ErrPasswordIncorrect, ...)
    
    // ❌ 混杂了 authz 模块的错误码
    registerBase(ErrPermissionDenied, ...)
    
    // 基础错误码
    registerBase(ErrDatabase, ...)
    ...
}
```

### 重构后: base.go
```go
func init() {
    // ✅ 只注册真正通用的基础错误码
    registerBase(ErrSuccess, 200, "OK")
    registerBase(ErrUnknown, 500, "Internal server error")
    registerBase(ErrBind, 400, "Error occurred while binding...")
    registerBase(ErrValidation, 400, "Validation failed")
    registerBase(ErrPageNotFound, 404, "Page not found")
    registerBase(ErrDatabase, 500, "Database error")
    registerBase(ErrInternalServerError, 500, "Internal server error")
    registerBase(ErrUnauthenticated, 401, "Authentication failed")
    registerBase(ErrUnauthorized, 403, "Authorization failed")
    ...
}
```

### 重构后: authn.go
```go
func init() {
    registerAuthn()
}

func registerAuthn() {
    // ✅ 所有认证错误码在自己的文件中注册
    errors.MustRegister(&authnCoder{code: ErrTokenInvalid, ...})
    errors.MustRegister(&authnCoder{code: ErrEncrypt, ...})
    errors.MustRegister(&authnCoder{code: ErrSignatureInvalid, ...})
    errors.MustRegister(&authnCoder{code: ErrExpired, ...})
    errors.MustRegister(&authnCoder{code: ErrInvalidAuthHeader, ...})
    errors.MustRegister(&authnCoder{code: ErrMissingHeader, ...})
    errors.MustRegister(&authnCoder{code: ErrPasswordIncorrect, ...})
}
```

### 重构后: authz.go
```go
func init() {
    registerAuthz()
}

func registerAuthz() {
    // ✅ 授权错误码在自己的文件中注册
    errors.MustRegister(&authzCoder{
        code: ErrPermissionDenied, 
        status: http.StatusForbidden, 
        msg: "Permission denied"
    })
}
```

## 错误码分布统计

### 按文件统计

| 文件 | 错误码数量 | 职责 |
|------|-----------|------|
| base.go | 23 | 通用基础错误码 |
| authn.go | 7 | 认证相关错误码 |
| authz.go | 1 | 授权相关错误码 |
| identity.go | 12 | 用户身份相关错误码 |
| jwks.go | 19 | JWKS 相关错误码 |
| **总计** | **62** | **全部已注册** |

### 按模块统计

| 模块 | 错误码数量 | 文件 |
|------|-----------|------|
| 基础通用 | 23 | base.go |
| 认证 (authn) | 7 | authn.go |
| 授权 (authz) | 1 | authz.go |
| 用户身份 (identity) | 12 | identity.go |
| JWKS | 19 | jwks.go |

## 测试验证

### 1. 错误码注册测试
```bash
$ go test -v ./internal/pkg/code/
=== RUN   TestAuthnErrorCodesRegistration
--- PASS: TestAuthnErrorCodesRegistration (0.00s)
=== RUN   TestAuthnErrorCodesUsage
--- PASS: TestAuthnErrorCodesUsage (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrTokenInvalid (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrEncrypt (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrSignatureInvalid (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrExpired (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrInvalidAuthHeader (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrMissingHeader (0.00s)
    --- PASS: TestAuthnErrorCodesUsage/ErrPasswordIncorrect (0.00s)
=== RUN   TestIdentityErrorCodesRegistration
--- PASS: TestIdentityErrorCodesRegistration (0.00s)
PASS
ok      github.com/fangcun-mount/iam-contracts/internal/pkg/code  0.556s
```

### 2. 业务模块测试
```bash
$ go test ./internal/apiserver/modules/authn/...
ok  github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn  1.318s
ok  github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication  2.135s
ok  github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/service/authenticator  2.124s
...
✅ 所有测试通过
```

## 重构收益

### 1. 代码组织更清晰

- ✅ 每个模块的错误码在自己的文件中
- ✅ 定义和注册在同一位置
- ✅ 职责单一，易于维护

### 2. 避免循环依赖

- ✅ base.go 不再依赖其他模块的具体错误码
- ✅ 每个模块独立注册，解耦合

### 3. 易于扩展

- ✅ 添加新模块错误码时，只需创建新文件
- ✅ 不会影响其他模块

### 4. 代码一致性
```go
// 统一的模式
// 1. 定义错误码常量
const (
    ErrXxx = 1xxxxx
)

// 2. 初始化注册
func init() {
    registerXxx()
}

// 3. 注册函数
func registerXxx() {
    errors.MustRegister(&xxxCoder{...})
}

// 4. Coder 实现
type xxxCoder struct { ... }
func (c *xxxCoder) Code() int { ... }
func (c *xxxCoder) HTTPStatus() int { ... }
func (c *xxxCoder) String() string { ... }
```

## 变更文件清单

### 修改的文件

1. ✅ `internal/pkg/code/base.go`
   - 移除 authn 相关错误码的注册
   - 移除 authz 相关错误码的注册
   - 保留纯基础通用错误码

2. ✅ `internal/pkg/code/authn.go`
   - 删除未使用的 ErrTokenGeneration
   - 添加所有认证错误码的注册逻辑
   - 实现 authnCoder

3. ✅ `internal/pkg/code/authz.go`
   - 添加完整的注册逻辑
   - 实现 authzCoder

4. ✅ `internal/pkg/code/authn_registration_test.go`
   - 更新测试用例，覆盖所有 7 个认证错误码
   - 验证注册位置正确

### 未修改的文件

- ✅ `internal/pkg/code/identity.go` - 上个版本已修复
- ✅ `internal/pkg/code/identity_registration_test.go` - 测试已完善
- ✅ `internal/pkg/code/jwks.go` - 已正确实现
- ✅ `internal/pkg/code/base_test.go` - 无需修改

## 后续建议

### 1. 文档化错误码规范
建议在 `docs/` 下创建错误码设计规范文档：

- 错误码命名规则
- 错误码范围分配
- 注册模式最佳实践

### 2. 错误码冲突检测
可以添加编译时检查，确保：

- 错误码不重复
- HTTP 状态码与错误码语义一致
- 所有定义的错误码都已注册

### 3. 错误码使用统计
定期统计哪些错误码从未被使用，考虑清理

## 总结

本次重构完成了以下目标：

- ✅ **解决交叉注册问题**: 每个模块在自己的文件中注册
- ✅ **移除冗余代码**: 删除未使用的 ErrTokenGeneration
- ✅ **统一代码风格**: 所有错误码文件遵循相同模式
- ✅ **完全向后兼容**: 所有现有测试通过
- ✅ **提升代码质量**: 职责清晰，易于维护

### 重构前后对比

**重构前**:

- ❌ 4 个文件混合定义和注册
- ❌ base.go 承担过多职责
- ❌ 存在未使用的错误码

**重构后**:

- ✅ 5 个文件各司其职
- ✅ base.go 只负责基础错误码
- ✅ 无冗余代码
- ✅ 62 个错误码全部正确注册
- ✅ 测试覆盖完整

### 技术债务清理

- ✅ 删除了 1 个未使用的错误码
- ✅ 修正了 7 个错误码的注册位置
- ✅ 完善了 1 个模块的注册逻辑 (authz)
- ✅ 所有错误码文件遵循统一模式
