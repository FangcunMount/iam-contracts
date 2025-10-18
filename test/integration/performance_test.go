// Package test 性能测试
package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fangcun-mount/iam-contracts/pkg/dominguard"
)

// BenchmarkPermissionCheck 权限检查性能基准测试
func BenchmarkPermissionCheck(b *testing.B) {
	ctx := context.Background()

	guard, err := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = guard.CheckPermission(ctx, "user123", "tenant1", "order", "read")
	}
}

// BenchmarkBatchPermissionCheck 批量权限检查性能基准测试
func BenchmarkBatchPermissionCheck(b *testing.B) {
	ctx := context.Background()

	guard, err := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(b, err)

	permissions := []dominguard.Permission{
		{Resource: "order", Action: "read"},
		{Resource: "order", Action: "write"},
		{Resource: "product", Action: "read"},
		{Resource: "product", Action: "write"},
		{Resource: "user", Action: "read"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = guard.BatchCheckPermissions(ctx, "user123", "tenant1", permissions)
	}
}

// TestConcurrentPermissionCheck 并发权限检查测试
func TestConcurrentPermissionCheck(t *testing.T) {
	ctx := context.Background()

	guard, err := dominguard.NewDomainGuard(dominguard.Config{
		Enforcer: &mockEnforcer{},
		CacheTTL: 5 * time.Minute,
	})
	require.NoError(t, err)

	// 并发执行 100 个权限检查
	concurrency := 100
	done := make(chan bool, concurrency)

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			userID := fmt.Sprintf("user%d", id)
			_, err := guard.CheckPermission(ctx, userID, "tenant1", "order", "read")
			if err != nil {
				t.Errorf("并发检查失败: %v", err)
			}
			done <- true
		}(i)
	}

	// 等待所有检查完成
	for i := 0; i < concurrency; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("并发检查 %d 个权限，耗时: %v", concurrency, elapsed)
}
