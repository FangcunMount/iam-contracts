# 认证模块设计

## 🎯 设计理念

认证模块是框架的安全核心，提供多种认证策略和授权机制。基于JWT（JSON Web Token）和Basic认证，支持灵活的认证配置和扩展。模块采用策略模式设计，便于添加新的认证方式。

## 🏗️ 架构设计

```text
┌─────────────────────────────────────────────────────────────┐
│                  Authentication Module                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Auth Strategies                          │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │    JWT      │  │    Basic    │  │   Custom    │    │ │
│  │  │  Strategy   │  │  Strategy   │  │  Strategy   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Auth Middleware                          │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   JWT       │  │   Basic     │  │   Role      │    │ │
│  │  │ Middleware  │  │ Middleware  │  │ Middleware  │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Auth Handlers                            │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │   Login     │  │   Logout    │  │   Refresh   │    │ │
│  │  │   Handler   │  │   Handler   │  │   Handler   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 📦 核心组件

### 认证配置

```go
// JWT配置
type JWTConfig struct {
    Key        string        `json:"key" mapstructure:"key"`
    Timeout    time.Duration `json:"timeout" mapstructure:"timeout"`
    MaxRefresh time.Duration `json:"max-refresh" mapstructure:"max-refresh"`
    Realm      string        `json:"realm" mapstructure:"realm"`
}

// 认证选项
type AuthOptions struct {
    JWT *JWTConfig `json:"jwt" mapstructure:"jwt"`
}
```

### 认证策略接口

```go
// AuthStrategy 认证策略接口
type AuthStrategy interface {
    Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error)
    Validate(ctx context.Context, token string) (*AuthResult, error)
    GenerateToken(ctx context.Context, user *User) (string, error)
    RefreshToken(ctx context.Context, token string) (string, error)
}
```

## 🔧 认证策略

### JWT认证策略

```go
// JWTStrategy JWT认证策略
type JWTStrategy struct {
    config *JWTConfig
    auth   *Auth
}

// NewJWTStrategy 创建JWT认证策略
func NewJWTStrategy(config *JWTConfig, auth *Auth) *JWTStrategy {
    return &JWTStrategy{
        config: config,
        auth:   auth,
    }
}

// Authenticate 认证用户
func (j *JWTStrategy) Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error) {
    creds, ok := credentials.(*LoginInfo)
    if !ok {
        return nil, fmt.Errorf("invalid credentials type")
    }

    // 验证用户名密码
    if creds.Username == "admin" && creds.Password == "admin123" {
        user := &User{
            ID:       1,
            Username: creds.Username,
            Email:    "admin@example.com",
            Status:   "active",
        }

        // 生成JWT token
        token, err := j.GenerateToken(ctx, user)
        if err != nil {
            return nil, err
        }

        return &AuthResult{
            User:  user,
            Token: token,
        }, nil
    }

    return nil, fmt.Errorf("invalid credentials")
}

// GenerateToken 生成JWT token
func (j *JWTStrategy) GenerateToken(ctx context.Context, user *User) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,
        "email":    user.Email,
        "status":   user.Status,
        "exp":      time.Now().Add(j.config.Timeout).Unix(),
        "iat":      time.Now().Unix(),
        "iss":      "web-framework",
        "aud":      "web-framework",
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.config.Key))
}

// Validate 验证JWT token
func (j *JWTStrategy) Validate(ctx context.Context, tokenString string) (*AuthResult, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(j.config.Key), nil
    })

    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        user := &User{
            ID:       int64(claims["user_id"].(float64)),
            Username: claims["username"].(string),
            Email:    claims["email"].(string),
            Status:   claims["status"].(string),
        }

        return &AuthResult{
            User:  user,
            Token: tokenString,
        }, nil
    }

    return nil, fmt.Errorf("invalid token claims")
}
```

### Basic认证策略

```go
// BasicStrategy Basic认证策略
type BasicStrategy struct {
    authenticator func(username, password string) bool
}

// NewBasicStrategy 创建Basic认证策略
func NewBasicStrategy(authenticator func(username, password string) bool) *BasicStrategy {
    return &BasicStrategy{
        authenticator: authenticator,
    }
}

// Authenticate 认证用户
func (b *BasicStrategy) Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error) {
    creds, ok := credentials.(*BasicCredentials)
    if !ok {
        return nil, fmt.Errorf("invalid credentials type")
    }

    if b.authenticator(creds.Username, creds.Password) {
        user := &User{
            ID:       1,
            Username: creds.Username,
            Email:    "admin@example.com",
            Status:   "active",
        }

        return &AuthResult{
            User: user,
        }, nil
    }

    return nil, fmt.Errorf("invalid credentials")
}
```

## 🔧 中间件实现

### JWT中间件

```go
// JWTMiddleware JWT认证中间件
func JWTMiddleware(jwtStrategy *JWTStrategy) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从请求头获取token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Authorization header required",
            })
            c.Abort()
            return
        }

        // 解析Bearer token
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Bearer token required",
            })
            c.Abort()
            return
        }

        // 验证token
        result, err := jwtStrategy.Validate(c.Request.Context(), tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Invalid token",
                "error":   err.Error(),
            })
            c.Abort()
            return
        }

        // 将用户信息存入context
        c.Set("user", result.User)
        c.Set("user_id", result.User.ID)
        c.Set("username", result.User.Username)

        c.Next()
    }
}
```

### Basic认证中间件

```go
// BasicAuthMiddleware Basic认证中间件
func BasicAuthMiddleware(basicStrategy *BasicStrategy) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从请求头获取Basic认证信息
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.Header("WWW-Authenticate", `Basic realm="web-framework"`)
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Basic authentication required",
            })
            c.Abort()
            return
        }

        // 解析Basic认证
        if !strings.HasPrefix(authHeader, "Basic ") {
            c.Header("WWW-Authenticate", `Basic realm="web-framework"`)
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Invalid Basic authentication format",
            })
            c.Abort()
            return
        }

        // 解码Base64
        encoded := strings.TrimPrefix(authHeader, "Basic ")
        decoded, err := base64.StdEncoding.DecodeString(encoded)
        if err != nil {
            c.Header("WWW-Authenticate", `Basic realm="web-framework"`)
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Invalid Basic authentication encoding",
            })
            c.Abort()
            return
        }

        // 解析用户名密码
        parts := strings.SplitN(string(decoded), ":", 2)
        if len(parts) != 2 {
            c.Header("WWW-Authenticate", `Basic realm="web-framework"`)
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Invalid Basic authentication format",
            })
            c.Abort()
            return
        }

        username, password := parts[0], parts[1]

        // 验证凭据
        credentials := &BasicCredentials{
            Username: username,
            Password: password,
        }

        result, err := basicStrategy.Authenticate(c.Request.Context(), credentials)
        if err != nil {
            c.Header("WWW-Authenticate", `Basic realm="web-framework"`)
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    401,
                "message": "Invalid credentials",
            })
            c.Abort()
            return
        }

        // 将用户信息存入context
        c.Set("user", result.User)
        c.Set("user_id", result.User.ID)
        c.Set("username", result.User.Username)

        c.Next()
    }
}
```

## 🔄 认证处理器

### 登录处理器

```go
// LoginHandler 登录处理器
type LoginHandler struct {
    jwtStrategy *JWTStrategy
}

// NewLoginHandler 创建登录处理器
func NewLoginHandler(jwtStrategy *JWTStrategy) *LoginHandler {
    return &LoginHandler{
        jwtStrategy: jwtStrategy,
    }
}

// Login 用户登录
func (h *LoginHandler) Login(c *gin.Context) {
    var loginInfo LoginInfo
    if err := c.ShouldBindJSON(&loginInfo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "code":    400,
            "message": "Invalid request body",
            "error":   err.Error(),
        })
        return
    }

    // 验证用户名密码
    result, err := h.jwtStrategy.Authenticate(c.Request.Context(), &loginInfo)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "code":    401,
            "message": "Invalid credentials",
            "error":   err.Error(),
        })
        return
    }

    // 返回JWT token
    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": gin.H{
            "token": result.Token,
            "user": gin.H{
                "id":       result.User.ID,
                "username": result.User.Username,
                "email":    result.User.Email,
                "status":   result.User.Status,
            },
        },
        "message": "Login successful",
    })
}
```

### 用户信息处理器

```go
// GetUserProfile 获取用户资料
func (h *UserHandler) GetUserProfile(c *gin.Context) {
    // 从JWT token中获取用户信息
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{
            "code":    401,
            "message": "未授权访问",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 200,
        "data": gin.H{
            "user_id": userID,
            "profile": gin.H{
                "id":       userID,
                "username": "demo_user",
                "email":    "demo@example.com",
                "status":   "active",
            },
        },
        "message": "获取用户资料成功",
    })
}
```

## 🎨 设计模式

### 1. 策略模式

通过策略模式实现不同的认证方式。

```go
// 认证策略接口
type AuthStrategy interface {
    Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error)
    Validate(ctx context.Context, token string) (*AuthResult, error)
}

// JWT策略
type JWTStrategy struct {
    config *JWTConfig
}

// Basic策略
type BasicStrategy struct {
    authenticator func(username, password string) bool
}
```

### 2. 工厂模式

使用工厂模式创建不同的认证策略。

```go
// 认证策略工厂
func NewJWTStrategy(config *JWTConfig) *JWTStrategy {
    return &JWTStrategy{config: config}
}

func NewBasicStrategy(authenticator func(username, password string) bool) *BasicStrategy {
    return &BasicStrategy{authenticator: authenticator}
}
```

### 3. 中间件模式

使用中间件模式实现认证逻辑。

```go
// 认证中间件
func JWTMiddleware(strategy *JWTStrategy) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 认证逻辑
    }
}
```

## 📈 扩展指南

### 添加新的认证策略

1.**定义策略接口**

```go
type OAuth2Strategy struct {
    config *OAuth2Config
}

func (o *OAuth2Strategy) Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error) {
    // 实现OAuth2认证逻辑
    return nil, nil
}

func (o *OAuth2Strategy) Validate(ctx context.Context, token string) (*AuthResult, error) {
    // 实现OAuth2 token验证逻辑
    return nil, nil
}
```

2.**创建中间件**

```go
func OAuth2Middleware(strategy *OAuth2Strategy) gin.HandlerFunc {
    return func(c *gin.Context) {
        // OAuth2认证中间件逻辑
    }
}
```

3.**注册到路由**

```go
// 在路由中注册
authGroup := router.Group("/auth")
{
    authGroup.POST("/oauth2/login", oauth2Handler.Login)
    authGroup.GET("/oauth2/callback", oauth2Handler.Callback)
}

// 使用OAuth2中间件保护路由
protected := router.Group("/api/v1")
protected.Use(OAuth2Middleware(oauth2Strategy))
{
    protected.GET("/profile", userHandler.GetProfile)
}
```

## 🧪 测试策略

### 单元测试

```go
func TestJWTStrategy_Authenticate(t *testing.T) {
    config := &JWTConfig{
        Key:     "test-secret",
        Timeout: time.Hour,
    }
    
    strategy := NewJWTStrategy(config)
    
    credentials := &LoginInfo{
        Username: "admin",
        Password: "admin123",
    }
    
    result, err := strategy.Authenticate(context.Background(), credentials)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotEmpty(t, result.Token)
    assert.Equal(t, "admin", result.User.Username)
}
```

### 集成测试

```go
func TestJWTMiddleware(t *testing.T) {
    // 创建测试服务器
    router := gin.New()
    strategy := NewJWTStrategy(testConfig)
    router.Use(JWTMiddleware(strategy))
    
    router.GET("/test", func(c *gin.Context) {
        userID, _ := c.Get("user_id")
        c.JSON(200, gin.H{"user_id": userID})
    })
    
    // 测试有效token
    token := generateTestToken(t, strategy)
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

## 🎯 最佳实践

### 1. 安全配置

```go
// 生产环境JWT配置
jwtConfig := &JWTConfig{
    Key:        "your-very-long-and-secure-secret-key",  // 至少32位
    Timeout:    time.Hour,                               // 1小时过期
    MaxRefresh: time.Hour * 24 * 7,                      // 7天刷新
    Realm:      "web-framework",
}
```

### 2. Token管理

```go
// 使用安全的token生成
func generateSecureToken(user *User) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,
        "exp":      time.Now().Add(time.Hour).Unix(),
        "iat":      time.Now().Unix(),
        "iss":      "web-framework",
        "aud":      "web-framework",
        "jti":      uuid.New().String(),  // 唯一标识
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secretKey))
}
```

### 3. 错误处理

```go
// 统一的认证错误处理
func handleAuthError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, ErrInvalidCredentials):
        c.JSON(http.StatusUnauthorized, gin.H{
            "code":    401,
            "message": "Invalid credentials",
        })
    case errors.Is(err, ErrTokenExpired):
        c.JSON(http.StatusUnauthorized, gin.H{
            "code":    401,
            "message": "Token expired",
        })
    default:
        c.JSON(http.StatusInternalServerError, gin.H{
            "code":    500,
            "message": "Internal server error",
        })
    }
}
```

### 4. 日志记录

```go
// 记录认证事件
func logAuthEvent(c *gin.Context, event string, userID int64, success bool) {
    logger := log.FromContext(c.Request.Context())
    logger.Info("Authentication event",
        log.String("event", event),
        log.Int64("user_id", userID),
        log.Bool("success", success),
        log.String("ip", c.ClientIP()),
        log.String("user_agent", c.GetHeader("User-Agent")),
    )
}
```

## 🔒 安全考虑

### 1. Token安全

- 使用强密钥（至少32位）
- 设置合理的过期时间
- 实现token刷新机制
- 支持token撤销

### 2. 密码安全

- 使用bcrypt等安全哈希算法
- 实施密码复杂度要求
- 限制登录尝试次数
- 实现账户锁定机制

### 3. 传输安全

- 使用HTTPS传输
- 设置安全的Cookie属性
- 实现CSRF保护
- 使用安全的会话管理

### 4. 监控和审计

- 记录所有认证事件
- 监控异常登录行为
- 实现安全告警机制
- 定期安全审计
