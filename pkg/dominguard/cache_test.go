// Package dominguard 缓存测试
package dominguard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewVersionCache 测试创建缓存
func TestNewVersionCache(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
}

// TestVersionCache_SetAndGet 测试缓存设置和获取
func TestVersionCache_SetAndGet(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	// 设置缓存
	cache.Set("tenant1", 1)

	// 获取缓存
	version, ok := cache.Get("tenant1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), version)

	// 获取不存在的缓存
	_, ok = cache.Get("tenant2")
	assert.False(t, ok)
}

// TestVersionCache_Update 测试缓存更新
func TestVersionCache_Update(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	// 设置初始版本
	cache.Set("tenant1", 1)

	// 更新版本
	cache.Set("tenant1", 2)

	// 验证更新
	version, ok := cache.Get("tenant1")
	assert.True(t, ok)
	assert.Equal(t, int64(2), version)
}

// TestVersionCache_Expiry 测试缓存过期
func TestVersionCache_Expiry(t *testing.T) {
	// 使用短过期时间
	cache := NewVersionCache(100 * time.Millisecond)

	// 设置缓存
	cache.Set("tenant1", 1)

	// 立即获取应该成功
	version, ok := cache.Get("tenant1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), version)

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 获取应该失败（已过期）
	_, ok = cache.Get("tenant1")
	assert.False(t, ok)
}

// TestVersionCache_Clear 测试清空缓存
func TestVersionCache_Clear(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	// 设置多个缓存
	cache.Set("tenant1", 1)
	cache.Set("tenant2", 2)
	cache.Set("tenant3", 3)

	// 清空缓存
	cache.Clear()

	// 验证所有缓存都被清空
	_, ok1 := cache.Get("tenant1")
	_, ok2 := cache.Get("tenant2")
	_, ok3 := cache.Get("tenant3")

	assert.False(t, ok1)
	assert.False(t, ok2)
	assert.False(t, ok3)
}

// TestVersionCache_ConcurrentAccess 测试并发访问
func TestVersionCache_ConcurrentAccess(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	done := make(chan bool, 100)

	// 并发写入
	for i := 0; i < 50; i++ {
		go func(idx int) {
			tenantID := "tenant1"
			version := int64(1)
			cache.Set(tenantID, version)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 50; i++ {
		go func(idx int) {
			tenantID := "tenant1"
			_, _ = cache.Get(tenantID)
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证最终状态
	version, ok := cache.Get("tenant1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), version)
}

// TestVersionCache_ConcurrentClear 测试并发清空
func TestVersionCache_ConcurrentClear(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	// 设置初始数据
	cache.Set("tenant1", 1)
	cache.Set("tenant2", 2)

	done := make(chan bool, 100)

	// 并发操作：读、写、清空
	for i := 0; i < 30; i++ {
		go func() {
			cache.Set("tenant1", 1)
			done <- true
		}()
	}

	for i := 0; i < 30; i++ {
		go func() {
			_, _ = cache.Get("tenant1")
			done <- true
		}()
	}

	for i := 0; i < 40; i++ {
		go func() {
			cache.Clear()
			done <- true
		}()
	}

	// 等待所有操作完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 测试应该不会 panic
	assert.NotNil(t, cache)
}

// TestVersionCache_CleanupExpired 测试过期清理
func TestVersionCache_CleanupExpired(t *testing.T) {
	// 使用短过期时间
	cache := NewVersionCache(50 * time.Millisecond)

	// 设置多个缓存
	cache.Set("tenant1", 1)
	cache.Set("tenant2", 2)

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// 旧缓存应该已过期
	_, ok1 := cache.Get("tenant1")
	_, ok2 := cache.Get("tenant2")
	assert.False(t, ok1)
	assert.False(t, ok2)

	// 设置新缓存
	cache.Set("tenant3", 3)

	// tenant3 应该存在（刚设置）
	_, ok3 := cache.Get("tenant3")
	assert.True(t, ok3)
}

// TestVersionCache_MultipleKeys 测试多个键的管理
func TestVersionCache_MultipleKeys(t *testing.T) {
	cache := NewVersionCache(5 * time.Minute)

	// 设置多个不同租户的版本
	tenants := []string{"tenant1", "tenant2", "tenant3", "tenant4", "tenant5"}
	for i, tid := range tenants {
		version := int64(i + 1)
		cache.Set(tid, version)
	}

	// 验证所有缓存
	for i, tid := range tenants {
		expected := int64(i + 1)
		version, ok := cache.Get(tid)
		assert.True(t, ok, "tenant %s should exist", tid)
		assert.Equal(t, expected, version, "tenant %s version mismatch", tid)
	}
}

// BenchmarkVersionCache_Set 基准测试：设置缓存
func BenchmarkVersionCache_Set(b *testing.B) {
	cache := NewVersionCache(5 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("tenant1", 1)
	}
}

// BenchmarkVersionCache_Get 基准测试：获取缓存
func BenchmarkVersionCache_Get(b *testing.B) {
	cache := NewVersionCache(5 * time.Minute)
	cache.Set("tenant1", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("tenant1")
	}
}

// BenchmarkVersionCache_Concurrent 基准测试：并发访问
func BenchmarkVersionCache_Concurrent(b *testing.B) {
	cache := NewVersionCache(5 * time.Minute)
	cache.Set("tenant1", 1)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get("tenant1")
		}
	})
}
