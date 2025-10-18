# 认证中心 - API 参考

> 详细介绍 REST API 接口、集成方案和客户端示例代码

📖 [返回主文档](./README.md)

---

{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkstMjAyNS0xMCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkstMjAyNS0xMCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}

## 刷新 Token

POST /api/v1/auth/token:refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkstMjAyNS0xMCJ9..."
}

Response: 200 OK
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkstMjAyNS0xMCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkstMjAyNS0xMCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}

## 登出（撤销 Token）

POST /api/v1/auth:logout
Authorization: Bearer {access_token}

Response: 204 No Content

## 本地密码登录

POST /api/v1/auth:login
Content-Type: application/json

{
  "phone": "13800138000",
  "password": "P@ssw0rd123"
}

Response: 200 OK
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 900
}

### 7.2 公钥 API

```http
# JWKS 公钥集
GET /.well-known/jwks.json

Response: 200 OK
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "K-2025-10",
      "use": "sig",
      "alg": "RS256",
      "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx...",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "kid": "K-2025-09",
      "use": "sig",
      "alg": "RS256",
      "n": "xjwU2L9sTxMvXLh5YU8k8qS7wX9_Vkj3sP2nL8mQ5zRtYpO...",
      "e": "AQAB"
    }
  ]
}

# OpenID Connect Discovery
GET /.well-known/openid-configuration

Response: 200 OK
{
  "issuer": "https://iam.example.com",
  "authorization_endpoint": "https://iam.example.com/auth/authorize",
  "token_endpoint": "https://iam.example.com/auth/token",
  "jwks_uri": "https://iam.example.com/.well-known/jwks.json",
  "response_types_supported": ["code", "token"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"]
}
```

---

## 8. 集成方案

### 8.1 业务服务集成（Middleware）

```go
// 业务服务中间件
package middleware

import (
    "context"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
    jwksURL    string
    publicKeys map[string]*rsa.PublicKey // kid -> public key
    cacheTTL   time.Duration
}

func NewAuthMiddleware(jwksURL string) *AuthMiddleware {
    m := &AuthMiddleware{
        jwksURL:    jwksURL,
        publicKeys: make(map[string]*rsa.PublicKey),
        cacheTTL:   1 * time.Hour,
    }
    
    // 启动时加载公钥
    m.RefreshPublicKeys()
    
    // 定期刷新
    go m.periodicRefresh()
    
    return m
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取 Token
        tokenString := extractToken(c)
        if tokenString == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
            return
        }
        
        // 2. 解析 Token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // 获取 kid
            kid, ok := token.Header["kid"].(string)
            if !ok {
                return nil, errors.New("missing kid in token header")
            }
            
            // 查找公钥
            publicKey, ok := m.publicKeys[kid]
            if !ok {
                // 尝试刷新公钥
                m.RefreshPublicKeys()
                publicKey, ok = m.publicKeys[kid]
                if !ok {
                    return nil, errors.New("unknown kid")
                }
            }
            
            return publicKey, nil
        })
        
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        
        // 3. 提取 Claims
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid claims"})
            return
        }
        
        // 4. 验证 Claims
        if err := m.validateClaims(claims); err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
            return
        }
        
        // 5. 设置用户上下文
        userID := claims["sub"].(string)
        c.Set("user_id", userID)
        c.Set("claims", claims)
        
        c.Next()
    }
}

func (m *AuthMiddleware) RefreshPublicKeys() error {
    resp, err := http.Get(m.jwksURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var jwks struct {
        Keys []struct {
            Kid string `json:"kid"`
            N   string `json:"n"`
            E   string `json:"e"`
        } `json:"keys"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
        return err
    }
    
    newKeys := make(map[string]*rsa.PublicKey)
    for _, key := range jwks.Keys {
        pubKey, err := jwkToPublicKey(key.N, key.E)
        if err != nil {
            continue
        }
        newKeys[key.Kid] = pubKey
    }
    
    m.publicKeys = newKeys
    return nil
}

func (m *AuthMiddleware) validateClaims(claims jwt.MapClaims) error {
    // 验证 Issuer
    if claims["iss"] != "iam-auth-service" {
        return errors.New("invalid issuer")
    }
    
    // 验证 Audience
    if claims["aud"] != "iam-platform" {
        return errors.New("invalid audience")
    }
    
    // 验证 Expiration (jwt 库已自动验证)
    
    // 验证 Token Type
    if claims["type"] != "access" {
        return errors.New("invalid token type")
    }
    
    return nil
}
```

### 8.2 使用示例

```go
// main.go
func main() {
    r := gin.Default()
    
    // 创建认证中间件
    authMiddleware := middleware.NewAuthMiddleware(
        "https://iam.example.com/.well-known/jwks.json",
    )
    
    // 公开路由（无需认证）
    r.GET("/health", healthCheck)
    
    // 受保护路由（需要认证）
    authorized := r.Group("/api/v1")
    authorized.Use(authMiddleware.Authenticate())
    {
        authorized.GET("/users/me", getUserProfile)
        authorized.GET("/children", listChildren)
    }
    
    r.Run(":8080")
}

func getUserProfile(c *gin.Context) {
    userID := c.GetString("user_id")
    claims := c.MustGet("claims").(jwt.MapClaims)
    
    // 业务逻辑
    user := fetchUser(userID)
    c.JSON(200, user)
}
```

### 8.3 客户端集成（小程序）

```javascript
// utils/auth.js
class AuthManager {
  constructor() {
    this.accessToken = wx.getStorageSync('access_token') || '';
    this.refreshToken = wx.getStorageSync('refresh_token') || '';
    this.expiresAt = wx.getStorageSync('expires_at') || 0;
  }
  
  // 微信登录
  async loginWithWechat() {
    // 1. 获取微信 code
    const { code } = await wx.login();
    
    // 2. 调用后端登录接口
    const res = await wx.request({
      url: 'https://api.example.com/api/v1/auth/wechat:login',
      method: 'POST',
      data: {
        code: code,
        device_id: this.getDeviceId()
      }
    });
    
    // 3. 保存 Token
    this.saveTokens(res.data);
    
    return res.data;
  }
  
  // 保存 Token
  saveTokens(data) {
    this.accessToken = data.access_token;
    this.refreshToken = data.refresh_token;
    this.expiresAt = Date.now() + data.expires_in * 1000;
    
    wx.setStorageSync('access_token', this.accessToken);
    wx.setStorageSync('refresh_token', this.refreshToken);
    wx.setStorageSync('expires_at', this.expiresAt);
  }
  
  // 自动刷新 Token
  async autoRefreshToken() {
    // 提前 1 分钟刷新
    if (Date.now() < this.expiresAt - 60 * 1000) {
      return;
    }
    
    try {
      const res = await wx.request({
        url: 'https://api.example.com/api/v1/auth/token:refresh',
        method: 'POST',
        data: {
          refresh_token: this.refreshToken
        }
      });
      
      this.saveTokens(res.data);
    } catch (err) {
      // Refresh Token 也过期，需要重新登录
      this.logout();
      wx.reLaunch({ url: '/pages/login/login' });
    }
  }
  
  // HTTP 请求拦截器
  async request(options) {
    // 自动刷新
    await this.autoRefreshToken();
    
    // 添加 Authorization Header
    options.header = options.header || {};
    options.header['Authorization'] = `Bearer ${this.accessToken}`;
    
    const res = await wx.request(options);
    
    // 处理 401
    if (res.statusCode === 401) {
      await this.loginWithWechat();
      // 重试
      return this.request(options);
    }
    
    return res;
  }
  
  // 登出
  logout() {
    this.accessToken = '';
    this.refreshToken = '';
    this.expiresAt = 0;
    
    wx.removeStorageSync('access_token');
    wx.removeStorageSync('refresh_token');
    wx.removeStorageSync('expires_at');
  }
}

export default new AuthManager();

// 使用示例
import auth from './utils/auth';

// 登录
await auth.loginWithWechat();

// 调用 API
const res = await auth.request({
  url: 'https://api.example.com/api/v1/users/me',
  method: 'GET'
});
```

---

## 9. 总结

### 9.1 核心优势

- ✅ **多渠道支持**: 微信、企业微信、本地密码等多种认证方式
- ✅ **标准化**: 基于 JWT + JWKS 标准，易于集成
- ✅ **高性能**: 本地 Token 验证，无需每次调用认证服务
- ✅ **安全性**: RS256 签名、密钥轮换、黑名单机制
- ✅ **易扩展**: 新增认证方式只需实现 Adapter 接口

### 9.2 最佳实践

1. **Token 短期有效**: Access Token 15分钟，减少泄露风险
