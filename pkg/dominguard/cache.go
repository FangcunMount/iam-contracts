// Package dominguard 版本缓存
package dominguard

import (
	"sync"
	"time"
)

// VersionCache 策略版本缓存
type VersionCache struct {
	cache map[string]*cacheEntry
	ttl   time.Duration
	mu    sync.RWMutex
}

// cacheEntry 缓存条目
type cacheEntry struct {
	version   int64
	expiredAt time.Time
}

// NewVersionCache 创建版本缓存
func NewVersionCache(ttl time.Duration) *VersionCache {
	vc := &VersionCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	// 启动清理协程
	go vc.cleanupExpired()

	return vc
}

// Get 获取缓存的版本号
func (vc *VersionCache) Get(tenantID string) (int64, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	entry, exists := vc.cache[tenantID]
	if !exists {
		return 0, false
	}

	// 检查是否过期
	if time.Now().After(entry.expiredAt) {
		return 0, false
	}

	return entry.version, true
}

// Set 设置缓存的版本号
func (vc *VersionCache) Set(tenantID string, version int64) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.cache[tenantID] = &cacheEntry{
		version:   version,
		expiredAt: time.Now().Add(vc.ttl),
	}
}

// Clear 清空所有缓存
func (vc *VersionCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.cache = make(map[string]*cacheEntry)
}

// Delete 删除指定租户的缓存
func (vc *VersionCache) Delete(tenantID string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	delete(vc.cache, tenantID)
}

// cleanupExpired 定期清理过期的缓存
func (vc *VersionCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		vc.mu.Lock()
		now := time.Now()
		for tenantID, entry := range vc.cache {
			if now.After(entry.expiredAt) {
				delete(vc.cache, tenantID)
			}
		}
		vc.mu.Unlock()
	}
}
