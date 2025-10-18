// Package dominguard 中间件测试
package dominguard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAuthMiddleware_RequirePermission 测试单个权限要求
func TestAuthMiddleware_RequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		enforceFunc    func(sub, dom, obj, act string) (bool, error)
		userID         string
		tenantID       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "权限允许",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return true, nil
			},
			userID:         "user123",
			tenantID:       "tenant1",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "权限拒绝",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
			userID:         "user123",
			tenantID:       "tenant1",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name: "权限检查错误",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, errors.New("enforcer error")
			},
			userID:         "user123",
			tenantID:       "tenant1",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 DomainGuard
			guard, err := NewDomainGuard(Config{
				Enforcer: &mockEnforcer{enforceFunc: tt.enforceFunc},
				CacheTTL: 5 * time.Minute,
			})
			require.NoError(t, err)

			// 创建中间件
			authMiddleware := NewAuthMiddleware(MiddlewareConfig{
				Guard:       guard,
				GetUserID:   func(c *gin.Context) string { return tt.userID },
				GetTenantID: func(c *gin.Context) string { return tt.tenantID },
			})

			// 创建测试路由
			router := gin.New()
			router.GET("/orders",
				authMiddleware.RequirePermission("order", "read"),
				func(c *gin.Context) {
					c.String(http.StatusOK, "success")
				},
			)

			// 发送测试请求
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/orders", nil)
			router.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestAuthMiddleware_RequireAnyPermission 测试任意权限要求
func TestAuthMiddleware_RequireAnyPermission(t *testing.T) {
	tests := []struct {
		name           string
		enforceFunc    func(sub, dom, obj, act string) (bool, error)
		expectedStatus int
	}{
		{
			name: "第一个权限满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// order:read 允许
				if obj == "resource:order" && act == "read" {
					return true, nil
				}
				return false, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "第二个权限满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// order:write 允许
				if obj == "resource:order" && act == "write" {
					return true, nil
				}
				return false, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "所有权限都不满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewDomainGuard(Config{
				Enforcer: &mockEnforcer{enforceFunc: tt.enforceFunc},
				CacheTTL: 5 * time.Minute,
			})
			require.NoError(t, err)

			authMiddleware := NewAuthMiddleware(MiddlewareConfig{
				Guard:       guard,
				GetUserID:   func(c *gin.Context) string { return "user123" },
				GetTenantID: func(c *gin.Context) string { return "tenant1" },
			})

			router := gin.New()
			router.GET("/orders",
				authMiddleware.RequireAnyPermission([]Permission{
					{Resource: "order", Action: "read"},
					{Resource: "order", Action: "write"},
				}),
				func(c *gin.Context) {
					c.String(http.StatusOK, "success")
				},
			)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/orders", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestAuthMiddleware_RequireAllPermissions 测试所有权限要求
func TestAuthMiddleware_RequireAllPermissions(t *testing.T) {
	tests := []struct {
		name           string
		enforceFunc    func(sub, dom, obj, act string) (bool, error)
		expectedStatus int
	}{
		{
			name: "所有权限都满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return true, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "第一个权限不满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// order:read 拒绝
				if obj == "resource:order" && act == "read" {
					return false, nil
				}
				return true, nil
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "第二个权限不满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// order:write 拒绝
				if obj == "resource:order" && act == "write" {
					return false, nil
				}
				return true, nil
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "所有权限都不满足",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewDomainGuard(Config{
				Enforcer: &mockEnforcer{enforceFunc: tt.enforceFunc},
				CacheTTL: 5 * time.Minute,
			})
			require.NoError(t, err)

			authMiddleware := NewAuthMiddleware(MiddlewareConfig{
				Guard:       guard,
				GetUserID:   func(c *gin.Context) string { return "user123" },
				GetTenantID: func(c *gin.Context) string { return "tenant1" },
			})

			router := gin.New()
			router.GET("/orders",
				authMiddleware.RequireAllPermissions([]Permission{
					{Resource: "order", Action: "read"},
					{Resource: "order", Action: "write"},
				}),
				func(c *gin.Context) {
					c.String(http.StatusOK, "success")
				},
			)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/orders", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestAuthMiddleware_SkipPaths 测试跳过路径
func TestAuthMiddleware_SkipPaths(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil // 总是拒绝
			},
		},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(MiddlewareConfig{
		Guard:       guard,
		GetUserID:   func(c *gin.Context) string { return "user123" },
		GetTenantID: func(c *gin.Context) string { return "tenant1" },
		SkipPaths:   []string{"/health", "/public"},
	})

	router := gin.New()

	// 需要权限的路径
	router.GET("/orders",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "orders")
		},
	)

	// 跳过权限检查的路径
	router.GET("/health",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "healthy")
		},
	)

	// 测试需要权限的路径 - 应该被拒绝
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/orders", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// 测试跳过的路径 - 应该成功
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "healthy")
}

// TestAuthMiddleware_CustomErrorHandler 测试自定义错误处理
func TestAuthMiddleware_CustomErrorHandler(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
		},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	customErrorCalled := false
	authMiddleware := NewAuthMiddleware(MiddlewareConfig{
		Guard:       guard,
		GetUserID:   func(c *gin.Context) string { return "user123" },
		GetTenantID: func(c *gin.Context) string { return "tenant1" },
		ErrorHandler: func(c *gin.Context, err error) {
			customErrorCalled = true
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "custom_error",
				"message": err.Error(),
			})
		},
	})

	router := gin.New()
	router.GET("/orders",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/orders", nil)
	router.ServeHTTP(w, req)

	assert.True(t, customErrorCalled)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "custom_error")
}

// TestExtractBearerToken 测试提取 Bearer Token
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		expectedToken string
	}{
		{
			name:          "有效的 Bearer Token",
			authorization: "Bearer abc123token",
			expectedToken: "abc123token",
		},
		{
			name:          "无 Bearer 前缀",
			authorization: "abc123token",
			expectedToken: "",
		},
		{
			name:          "空 Authorization",
			authorization: "",
			expectedToken: "",
		},
		{
			name:          "只有 Bearer",
			authorization: "Bearer",
			expectedToken: "",
		},
		{
			name:          "Bearer 后多个空格",
			authorization: "Bearer   abc123token",
			expectedToken: "  abc123token", // ExtractBearerToken 不会 trim 空格
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			c.Request = req

			// 提取 token
			token := ExtractBearerToken(c)
			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

// TestAuthMiddleware_MissingUserID 测试缺失用户ID
func TestAuthMiddleware_MissingUserID(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(MiddlewareConfig{
		Guard:       guard,
		GetUserID:   func(c *gin.Context) string { return "" }, // 返回空用户ID
		GetTenantID: func(c *gin.Context) string { return "tenant1" },
	})

	router := gin.New()
	router.GET("/orders",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/orders", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_MissingTenantID 测试缺失租户ID
func TestAuthMiddleware_MissingTenantID(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(MiddlewareConfig{
		Guard:       guard,
		GetUserID:   func(c *gin.Context) string { return "user123" },
		GetTenantID: func(c *gin.Context) string { return "" }, // 返回空租户ID
	})

	router := gin.New()
	router.GET("/orders",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		},
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/orders", nil)
	router.ServeHTTP(w, req)

	// INVALID_TENANT 在 defaultErrorHandler 中返回 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestAuthMiddleware_MultipleRoutes 测试多个路由
func TestAuthMiddleware_MultipleRoutes(t *testing.T) {
	enforceFunc := func(sub, dom, obj, act string) (bool, error) {
		// 只允许 order 的 read 操作
		if obj == "resource:order" && act == "read" {
			return true, nil
		}
		return false, nil
	}

	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{enforceFunc: enforceFunc},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(MiddlewareConfig{
		Guard:       guard,
		GetUserID:   func(c *gin.Context) string { return "user123" },
		GetTenantID: func(c *gin.Context) string { return "tenant1" },
	})

	router := gin.New()
	router.GET("/orders",
		authMiddleware.RequirePermission("order", "read"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "orders")
		},
	)
	router.POST("/orders",
		authMiddleware.RequirePermission("order", "write"),
		func(c *gin.Context) {
			c.String(http.StatusOK, "created")
		},
	)

	// GET /orders - 应该成功
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/orders", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// POST /orders - 应该失败
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/orders", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}
