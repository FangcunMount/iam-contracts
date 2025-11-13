# HTTP 状态码统一设计

## 概述

自本次变更起,所有 RESTful API 的 HTTP 响应统一返回 **HTTP 200 OK**,无论是成功还是业务错误。业务状态通过响应体中的 `code` 字段来表示。

## 设计原则

### 之前的设计 (已废弃)
- HTTP 状态码用于表示业务错误状态
- 401: 认证失败
- 404: 资源未找到
- 409: 资源冲突
- 500: 服务器错误

### 当前的设计
- **所有响应统一返回 HTTP 200**
- 通过响应体中的 `code` 字段表示业务状态
- HTTP 状态码仅用于表示传输层/协议层的问题

## 响应格式

### 成功响应
```json
{
  "data": {
    // 实际的响应数据
  }
}
```

HTTP 状态码: **200 OK**

### 错误响应
```json
{
  "code": 102201,
  "message": "Account already exists",
  "reference": ""
}
```

HTTP 状态码: **200 OK**

## 错误码说明

错误码分类:
- `1xxx00` - 基础错误 (Base)
- `10xx00` - 认证错误 (Authentication)
- `11xx00` - 授权错误 (Authorization)
- `12xx00` - 用户错误 (User/Identity)
- `13xx00` - IDP 错误 (Identity Provider)

常见错误码示例:
- `102201` - 账号已存在
- `102202` - 外部 ID 已存在
- `102203` - 账号未找到
- `102101` - 认证失败
- `102102` - 凭据无效
- `102103` - Token 无效

详细的错误码列表请参考 `internal/pkg/code/` 目录下的文件。

## 优势

1. **客户端处理简化**: 客户端不需要处理各种 HTTP 状态码,统一检查响应体中的 `code` 字段即可
2. **更清晰的职责分离**: HTTP 状态码表示传输层状态,业务错误通过业务错误码表示
3. **更好的 API 设计**: 符合现代 REST API 最佳实践
4. **统一的错误处理**: 所有错误信息都在响应体中,便于日志记录和问题排查

## 客户端示例

### JavaScript/TypeScript
```typescript
async function callAPI(url: string) {
  const response = await fetch(url);
  
  // HTTP 层成功,检查业务状态
  if (response.ok) {
    const data = await response.json();
    
    // 检查业务错误码
    if (data.code && data.code !== 0) {
      throw new Error(`Business error: ${data.message} (code: ${data.code})`);
    }
    
    return data;
  }
  
  // HTTP 层失败 (网络错误、服务器宕机等)
  throw new Error(`HTTP error: ${response.status}`);
}
```

### Go
```go
type APIResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

func callAPI(url string) (*APIResponse, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("HTTP error: %w", err)
    }
    defer resp.Body.Close()
    
    var apiResp APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("decode error: %w", err)
    }
    
    // 检查业务错误码
    if apiResp.Code != 0 {
        return nil, fmt.Errorf("business error: %s (code: %d)", 
            apiResp.Message, apiResp.Code)
    }
    
    return &apiResp, nil
}
```

## 实现细节

### 核心函数
`pkg/core/core.go` 中的 `WriteResponse` 函数负责统一的响应写入:

```go
func WriteResponse(c *gin.Context, err error, data interface{}) {
    if err != nil {
        coder := errors.ParseCoder(err)
        // 统一返回 HTTP 200，业务错误通过响应体中的 code 字段表示
        c.JSON(http.StatusOK, ErrResponse{
            Code:      coder.Code(),
            Message:   coder.String(),
            Reference: coder.Reference(),
        })
        return
    }
    c.JSON(http.StatusOK, data)
}
```

### 错误处理模块
所有模块的错误处理都已统一使用 `WriteResponse`:
- `pkg/core/core.go` - 通用响应处理
- `internal/apiserver/interface/authz/restful/handler/base.go` - 授权模块
- 其他 handler 文件

## 注意事项

1. **gRPC 不受影响**: 此变更仅影响 RESTful HTTP API,gRPC 服务仍然使用 gRPC 的状态码系统
2. **向后兼容性**: 客户端需要更新以适应新的响应格式
3. **监控和日志**: 日志记录和监控系统应该关注响应体中的 `code` 字段,而不是 HTTP 状态码

## 变更历史

- **2024-11-11**: 统一所有 RESTful API 响应返回 HTTP 200,业务错误通过响应体中的 code 字段表示
