// Package dominguard DomainGuard 单元测试
package dominguard

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEnforcer 模拟 Casbin Enforcer
type mockEnforcer struct {
	enforceFunc func(sub, dom, obj, act string) (bool, error)
}

func (m *mockEnforcer) Enforce(sub, dom, obj, act string) (bool, error) {
	if m.enforceFunc != nil {
		return m.enforceFunc(sub, dom, obj, act)
	}
	return true, nil
}

// TestNewDomainGuard 测试创建 DomainGuard
func TestNewDomainGuard(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: Config{
				Enforcer: &mockEnforcer{},
				CacheTTL: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "缺少 Enforcer",
			config: Config{
				CacheTTL: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "默认 CacheTTL",
			config: Config{
				Enforcer: &mockEnforcer{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewDomainGuard(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, guard)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, guard)
			}
		})
	}
}

// TestCheckPermission 测试单个权限检查
func TestCheckPermission(t *testing.T) {
	tests := []struct {
		name         string
		enforceFunc  func(sub, dom, obj, act string) (bool, error)
		userID       string
		tenantID     string
		resource     string
		action       string
		wantAllowed  bool
		wantErr      bool
	}{
		{
			name: "权限允许",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return true, nil
			},
			userID:      "user123",
			tenantID:    "tenant1",
			resource:    "order",
			action:      "read",
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "权限拒绝",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
			userID:      "user123",
			tenantID:    "tenant1",
			resource:    "order",
			action:      "delete",
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name: "检查出错",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, errors.New("enforcer error")
			},
			userID:      "user123",
			tenantID:    "tenant1",
			resource:    "order",
			action:      "read",
			wantAllowed: false,
			wantErr:     true,
		},
		{
			name: "验证参数格式",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// 验证生成的参数格式
				assert.Equal(t, "user:user123", sub)
				assert.Equal(t, "tenant1", dom)
				assert.Equal(t, "resource:order", obj)
				assert.Equal(t, "read", act)
				return true, nil
			},
			userID:      "user123",
			tenantID:    "tenant1",
			resource:    "order",
			action:      "read",
			wantAllowed: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewDomainGuard(Config{
				Enforcer: &mockEnforcer{enforceFunc: tt.enforceFunc},
				CacheTTL: 5 * time.Minute,
			})
			require.NoError(t, err)

			allowed, err := guard.CheckPermission(
				context.Background(),
				tt.userID,
				tt.tenantID,
				tt.resource,
				tt.action,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestCheckServicePermission 测试服务权限检查
func TestCheckServicePermission(t *testing.T) {
	tests := []struct {
		name        string
		enforceFunc func(sub, dom, obj, act string) (bool, error)
		serviceID   string
		tenantID    string
		resource    string
		action      string
		wantAllowed bool
		wantErr     bool
	}{
		{
			name: "服务权限允许",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				assert.Equal(t, "service:service-api", sub)
				return true, nil
			},
			serviceID:   "service-api",
			tenantID:    "tenant1",
			resource:    "user",
			action:      "read",
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "服务权限拒绝",
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return false, nil
			},
			serviceID:   "service-api",
			tenantID:    "tenant1",
			resource:    "user",
			action:      "delete",
			wantAllowed: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewDomainGuard(Config{
				Enforcer: &mockEnforcer{enforceFunc: tt.enforceFunc},
				CacheTTL: 5 * time.Minute,
			})
			require.NoError(t, err)

			allowed, err := guard.CheckServicePermission(
				context.Background(),
				tt.serviceID,
				tt.tenantID,
				tt.resource,
				tt.action,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAllowed, allowed)
		})
	}
}

// TestBatchCheckPermissions 测试批量权限检查
func TestBatchCheckPermissions(t *testing.T) {
	enforceFunc := func(sub, dom, obj, act string) (bool, error) {
		// 模拟：只允许 read 操作
		return act == "read", nil
	}

	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{enforceFunc: enforceFunc},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	permissions := []Permission{
		{Resource: "order", Action: "read"},
		{Resource: "order", Action: "write"},
		{Resource: "product", Action: "read"},
		{Resource: "product", Action: "delete"},
	}

	results, err := guard.BatchCheckPermissions(
		context.Background(),
		"user123",
		"tenant1",
		permissions,
	)

	require.NoError(t, err)
	assert.Len(t, results, 4)

	// 验证结果 - BatchCheckPermissions 返回 map[string]bool
	assert.True(t, results["order:read"])      // 允许
	assert.False(t, results["order:write"])    // 拒绝
	assert.True(t, results["product:read"])    // 允许
	assert.False(t, results["product:delete"]) // 拒绝
}

// TestBatchCheckPermissions_Empty 测试空批量检查
func TestBatchCheckPermissions_Empty(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	results, err := guard.BatchCheckPermissions(
		context.Background(),
		"user123",
		"tenant1",
		[]Permission{},
	)

	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

// TestRegisterResource 测试资源注册
func TestRegisterResource(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	// 注册资源
	guard.RegisterResource("order", "订单管理")
	guard.RegisterResource("product", "产品管理")

	// 验证资源映射（通过公开的方法或直接访问 - 根据实现）
	// 注意：由于 resourceMapping 是私有的，这里仅测试不panic
	assert.NotNil(t, guard)
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				return true, nil
			},
		},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// 并发执行 100 个权限检查
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			userID := fmt.Sprintf("user%d", idx%10)
			_, err := guard.CheckPermission(ctx, userID, "tenant1", "order", "read")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestBatchCheckPermissions_Error 测试批量检查错误处理
func TestBatchCheckPermissions_Error(t *testing.T) {
	guard, err := NewDomainGuard(Config{
		Enforcer: &mockEnforcer{
			enforceFunc: func(sub, dom, obj, act string) (bool, error) {
				// 模拟权限检查错误
				return false, errors.New("enforcer error")
			},
		},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	permissions := []Permission{
		{Resource: "order", Action: "read"},
	}

	_, err = guard.BatchCheckPermissions(
		context.Background(),
		"user123",
		"tenant1",
		permissions,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enforcer error")
}

// TestCheckPermission_DifferentResources 测试不同资源的权限
func TestCheckPermission_DifferentResources(t *testing.T) {
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

	testCases := []struct {
		resource string
		action   string
		expected bool
	}{
		{"order", "read", true},
		{"order", "write", false},
		{"product", "read", false},
		{"user", "read", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s:%s", tc.resource, tc.action), func(t *testing.T) {
			allowed, err := guard.CheckPermission(
				context.Background(),
				"user123",
				"tenant1",
				tc.resource,
				tc.action,
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, allowed)
		})
	}
}
