# 统一异常处理

## 🎯 设计理念

统一异常处理是框架的重要组成部分，提供标准化的错误码、错误消息和错误处理机制。通过统一的错误处理，确保API响应的一致性，提高系统的可维护性和用户体验。

## 🏗️ 架构设计

```text
┌─────────────────────────────────────────────────────────────┐
│                  Error Handling System                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Error Types                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Business  │  │   System    │  │   Validation│    │ │
│  │  │    Error    │  │    Error    │  │    Error    │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Error Codes                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Success   │  │   Client    │  │    Server   │    │ │
│  │  │    Codes    │  │    Codes    │  │    Codes    │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Error Handlers                           │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Global    │  │   Custom    │  │   Logging   │    │ │
│  │  │   Handler   │  │   Handler   │  │   Handler   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 📦 核心组件

### 错误码定义

```go
// 成功码
const (
    Success = 200
)

// 客户端错误码 (400-499)
const (
    ErrInvalidRequest        = 400
    ErrUnauthorized          = 401
    ErrForbidden            = 403
    ErrNotFound             = 404
    ErrMethodNotAllowed     = 405
    ErrConflict             = 409
    ErrValidationFailed     = 422
    ErrTooManyRequests      = 429
)

// 服务器错误码 (500-599)
const (
    ErrInternalServerError  = 500
    ErrNotImplemented       = 501
    ErrServiceUnavailable   = 503
    ErrDatabaseError        = 510
    ErrCacheError           = 511
    ErrExternalServiceError = 512
)

// 业务错误码 (1000-9999)
const (
    ErrUserNotFound         = 1001
    ErrUserAlreadyExists    = 1002
    ErrInvalidCredentials   = 1003
    ErrTokenExpired         = 1004
    ErrTokenInvalid         = 1005
    ErrPermissionDenied     = 1006
    ErrResourceNotFound     = 1007
    ErrResourceConflict     = 1008
    ErrModuleInitializationFailed = 1009
)
```

### 错误结构

```go
// Error 错误结构
type Error struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
    Stack   string      `json:"stack,omitempty"`
}

// ErrorResponse API错误响应
type ErrorResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Error   *Error      `json:"error,omitempty"`
}
```

### 错误创建函数

```go
// NewError 创建新错误
func NewError(code int, message string) *Error {
    return &Error{
        Code:    code,
        Message: message,
    }
}

// NewErrorWithDetails 创建带详情的错误
func NewErrorWithDetails(code int, message string, details interface{}) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Details: details,
    }
}

// NewErrorWithStack 创建带堆栈的错误
func NewErrorWithStack(code int, message string, stack string) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Stack:   stack,
    }
}
```

## 🔧 错误处理机制

### 全局错误处理器

```go
// GlobalErrorHandler 全局错误处理器
func GlobalErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // 检查是否有错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            
            // 记录错误日志
            logger := log.FromContext(c.Request.Context())
            logger.Errorw("Request error",
                "error", err.Error(),
                "method", c.Request.Method,
                "path", c.Request.URL.Path,
                "status", c.Writer.Status(),
            )

            // 根据错误类型返回相应的响应
            response := handleError(err)
            c.JSON(response.Code, response)
        }
    }
}

// handleError 处理错误并返回标准响应
func handleError(err error) *ErrorResponse {
    switch e := err.(type) {
    case *Error:
        return &ErrorResponse{
            Code:    e.Code,
            Message: e.Message,
            Error:   e,
        }
    case *ValidationError:
        return &ErrorResponse{
            Code:    ErrValidationFailed,
            Message: "Validation failed",
            Error: &Error{
                Code:    ErrValidationFailed,
                Message: "Validation failed",
                Details: e.Details,
            },
        }
    case *BusinessError:
        return &ErrorResponse{
            Code:    e.Code,
            Message: e.Message,
            Error: &Error{
                Code:    e.Code,
                Message: e.Message,
                Details: e.Details,
            },
        }
    default:
        // 未知错误，返回500
        return &ErrorResponse{
            Code:    ErrInternalServerError,
            Message: "Internal server error",
            Error: &Error{
                Code:    ErrInternalServerError,
                Message: "Internal server error",
            },
        }
    }
}
```

### 自定义错误类型

```go
// ValidationError 验证错误
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

func (v *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", v.Field, v.Message)
}

// BusinessError 业务错误
type BusinessError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

func (b *BusinessError) Error() string {
    return fmt.Sprintf("business error %d: %s", b.Code, b.Message)
}

// SystemError 系统错误
type SystemError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Stack   string      `json:"stack,omitempty"`
}

func (s *SystemError) Error() string {
    return fmt.Sprintf("system error %d: %s", s.Code, s.Message)
}
```

## 🔄 错误处理流程

### 错误处理中间件

```go
// ErrorHandlingMiddleware 错误处理中间件
func ErrorHandlingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 设置恢复函数
        defer func() {
            if r := recover(); r != nil {
                // 记录panic日志
                logger := log.FromContext(c.Request.Context())
                logger.Errorw("Panic recovered",
                    "panic", r,
                    "method", c.Request.Method,
                    "path", c.Request.URL.Path,
                )

                // 返回500错误
                c.JSON(http.StatusInternalServerError, &ErrorResponse{
                    Code:    ErrInternalServerError,
                    Message: "Internal server error",
                    Error: &Error{
                        Code:    ErrInternalServerError,
                        Message: "Internal server error",
                    },
                })
            }
        }()

        c.Next()
    }
}
```

### 错误响应格式

```go
// 成功响应
{
    "code": 200,
    "message": "Success",
    "data": {
        "user_id": 123,
        "username": "admin"
    }
}

// 错误响应
{
    "code": 400,
    "message": "Bad Request",
    "error": {
        "code": 400,
        "message": "Invalid request parameters",
        "details": {
            "field": "email",
            "value": "invalid-email",
            "message": "Invalid email format"
        }
    }
}

// 验证错误响应
{
    "code": 422,
    "message": "Validation failed",
    "error": {
        "code": 422,
        "message": "Validation failed",
        "details": [
            {
                "field": "username",
                "value": "",
                "message": "Username is required"
            },
            {
                "field": "email",
                "value": "invalid-email",
                "message": "Invalid email format"
            }
        ]
    }
}
```

## 🎨 错误处理模式

### 1. 错误包装

```go
// 使用errors包包装错误
import "github.com/pkg/errors"

func processUser(userID int64) error {
    user, err := getUser(userID)
    if err != nil {
        return errors.Wrapf(err, "failed to get user %d", userID)
    }
    
    if err := validateUser(user); err != nil {
        return errors.Wrap(err, "user validation failed")
    }
    
    return nil
}
```

### 2. 错误转换

```go
// 将底层错误转换为业务错误
func getUser(userID int64) (*User, error) {
    user, err := userRepo.FindByID(userID)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, &BusinessError{
                Code:    ErrUserNotFound,
                Message: fmt.Sprintf("User %d not found", userID),
            }
        }
        return nil, &SystemError{
            Code:    ErrDatabaseError,
            Message: "Database error",
        }
    }
    return user, nil
}
```

### 3. 错误聚合

```go
// 聚合多个错误
type AggregateError struct {
    Errors []error
}

func (a *AggregateError) Error() string {
    if len(a.Errors) == 0 {
        return "no errors"
    }
    
    messages := make([]string, len(a.Errors))
    for i, err := range a.Errors {
        messages[i] = err.Error()
    }
    
    return fmt.Sprintf("multiple errors: %s", strings.Join(messages, "; "))
}

func (a *AggregateError) Add(err error) {
    if err != nil {
        a.Errors = append(a.Errors, err)
    }
}

func (a *AggregateError) HasErrors() bool {
    return len(a.Errors) > 0
}
```

## 📈 错误处理最佳实践

### 1. 错误码设计

```go
// 错误码命名规范
const (
    // 模块前缀 + 错误类型 + 具体错误
    ErrUserNotFound         = 1001  // 用户模块 - 未找到
    ErrUserAlreadyExists    = 1002  // 用户模块 - 已存在
    ErrUserInvalidPassword  = 1003  // 用户模块 - 密码无效
    
    ErrAuthTokenExpired     = 2001  // 认证模块 - Token过期
    ErrAuthTokenInvalid     = 2002  // 认证模块 - Token无效
    ErrAuthPermissionDenied = 2003  // 认证模块 - 权限不足
)
```

### 2. 错误消息设计

```go
// 错误消息应该：
// 1. 清晰明确
// 2. 对用户友好
// 3. 包含必要的上下文信息
// 4. 避免暴露敏感信息

// 好的错误消息
"User with email 'user@example.com' already exists"
"Password must be at least 8 characters long"
"Invalid token: token has expired"

// 不好的错误消息
"Error occurred"
"Something went wrong"
"Database error: connection refused"
```

### 3. 错误日志记录

```go
// 记录详细的错误信息
func logError(ctx context.Context, err error, req *http.Request) {
    logger := log.FromContext(ctx)
    
    logger.Errorw("Request error",
        "error", err.Error(),
        "method", req.Method,
        "path", req.URL.Path,
        "user_agent", req.UserAgent(),
        "remote_addr", req.RemoteAddr,
        "request_id", getRequestID(ctx),
    )
    
    // 记录堆栈信息（仅在开发环境）
    if config.IsDevelopment() {
        logger.Errorw("Error stack trace",
            "stack", string(debug.Stack()),
        )
    }
}
```

### 4. 错误恢复

```go
// 优雅的错误恢复
func gracefulErrorRecovery() {
    defer func() {
        if r := recover(); r != nil {
            log.Errorw("Application panic recovered",
                "panic", r,
                "stack", string(debug.Stack()),
            )
            
            // 发送告警
            sendAlert("Application panic", r)
            
            // 优雅关闭
            gracefulShutdown()
        }
    }()
    
    // 应用程序主逻辑
    runApplication()
}
```

## 🧪 测试策略

### 单元测试

```go
func TestErrorHandling(t *testing.T) {
    // 测试业务错误
    err := &BusinessError{
        Code:    ErrUserNotFound,
        Message: "User not found",
    }
    
    response := handleError(err)
    assert.Equal(t, ErrUserNotFound, response.Code)
    assert.Equal(t, "User not found", response.Message)
    
    // 测试验证错误
    validationErr := &ValidationError{
        Field:   "email",
        Value:   "invalid-email",
        Message: "Invalid email format",
    }
    
    response = handleError(validationErr)
    assert.Equal(t, ErrValidationFailed, response.Code)
    assert.Equal(t, "Validation failed", response.Message)
}
```

### 集成测试

```go
func TestErrorMiddleware(t *testing.T) {
    router := gin.New()
    router.Use(ErrorHandlingMiddleware())
    router.Use(GlobalErrorHandler())
    
    router.GET("/test-error", func(c *gin.Context) {
        c.Error(&BusinessError{
            Code:    ErrUserNotFound,
            Message: "User not found",
        })
    })
    
    req := httptest.NewRequest("GET", "/test-error", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response ErrorResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, ErrUserNotFound, response.Code)
}
```

## 🎯 监控和告警

### 错误监控

```go
// 错误计数器
var (
    errorCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_errors_total",
            Help: "Total number of HTTP errors",
        },
        []string{"code", "method", "path"},
    )
)

// 记录错误指标
func recordError(code int, method, path string) {
    errorCounter.WithLabelValues(
        strconv.Itoa(code),
        method,
        path,
    ).Inc()
}
```

### 错误告警

```go
// 错误率告警
func checkErrorRate() {
    errorRate := getErrorRate()
    if errorRate > 0.1 { // 错误率超过10%
        sendAlert("High error rate detected", map[string]interface{}{
            "error_rate": errorRate,
            "threshold":  0.1,
        })
    }
}

// 特定错误告警
func checkSpecificErrors() {
    criticalErrors := getCriticalErrors()
    for _, err := range criticalErrors {
        if err.Count > 10 { // 特定错误超过10次
            sendAlert("Critical error detected", map[string]interface{}{
                "error_code": err.Code,
                "error_message": err.Message,
                "count": err.Count,
            })
        }
    }
}
```

## 📊 错误分析

### 错误分类统计

```sql
-- 按错误类型统计
SELECT 
    error_code,
    error_message,
    COUNT(*) as count,
    DATE(created_at) as date
FROM error_logs 
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY error_code, error_message, DATE(created_at)
ORDER BY count DESC;

-- 按时间分布统计
SELECT 
    HOUR(created_at) as hour,
    COUNT(*) as error_count
FROM error_logs 
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY HOUR(created_at)
ORDER BY hour;
```

### 错误趋势分析

```go
// 错误趋势分析
type ErrorTrend struct {
    Date       string  `json:"date"`
    ErrorCount int     `json:"error_count"`
    ErrorRate  float64 `json:"error_rate"`
}

func analyzeErrorTrend(days int) []ErrorTrend {
    trends := make([]ErrorTrend, days)
    
    for i := 0; i < days; i++ {
        date := time.Now().AddDate(0, 0, -i)
        errorCount := getErrorCountByDate(date)
        totalRequests := getTotalRequestsByDate(date)
        errorRate := float64(errorCount) / float64(totalRequests)
        
        trends[i] = ErrorTrend{
            Date:       date.Format("2006-01-02"),
            ErrorCount: errorCount,
            ErrorRate:  errorRate,
        }
    }
    
    return trends
}
```
