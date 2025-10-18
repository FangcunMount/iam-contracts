// Package dominguard PEP SDK - 权限检查客户端
// DomainGuard 为业务服务提供简单易用的权限检查 API
package dominguard

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
)

// DomainGuard 权限检查客户端
type DomainGuard struct {
	enforcer        Enforcer
	versionCache    *VersionCache
	resourceMapping map[string]string // 资源 key 到显示名称的映射
}

// Enforcer Casbin Enforcer 接口
type Enforcer interface {
	// Enforce 检查权限
	Enforce(sub, dom, obj, act string) (bool, error)
}

// Config DomainGuard 配置
type Config struct {
	Enforcer     Enforcer      // Casbin Enforcer
	RedisClient  *redis.Client // Redis 客户端（用于监听策略变更）
	CacheTTL     time.Duration // 缓存过期时间
	VersionTopic string        // 策略版本变更主题
}

// NewDomainGuard 创建 DomainGuard 实例
func NewDomainGuard(config Config) (*DomainGuard, error) {
	if config.Enforcer == nil {
		return nil, fmt.Errorf("enforcer is required")
	}

	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	if config.VersionTopic == "" {
		config.VersionTopic = "authz:policy_changed"
	}

	guard := &DomainGuard{
		enforcer:        config.Enforcer,
		versionCache:    NewVersionCache(config.CacheTTL),
		resourceMapping: make(map[string]string),
	}

	// 如果提供了 Redis 客户端，启动策略变更监听
	if config.RedisClient != nil {
		go guard.watchPolicyChanges(config.RedisClient, config.VersionTopic)
	}

	return guard, nil
}

// CheckPermission 检查用户是否有权限执行操作
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - tenantID: 租户ID
//   - resource: 资源标识 (例如: "user", "order")
//   - action: 操作 (例如: "read", "write", "delete")
//
// 返回:
//   - bool: 是否有权限
//   - error: 错误信息
func (g *DomainGuard) CheckPermission(
	ctx context.Context,
	userID string,
	tenantID string,
	resource string,
	action string,
) (bool, error) {
	// 构建主体标识（用户）
	subject := fmt.Sprintf("user:%s", userID)

	// 构建资源标识
	object := fmt.Sprintf("resource:%s", resource)

	// 调用 Casbin 检查权限
	allowed, err := g.enforcer.Enforce(subject, tenantID, object, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return allowed, nil
}

// CheckServicePermission 检查服务是否有权限执行操作
//
// 参数:
//   - ctx: 上下文
//   - serviceID: 服务ID
//   - tenantID: 租户ID
//   - resource: 资源标识
//   - action: 操作
func (g *DomainGuard) CheckServicePermission(
	ctx context.Context,
	serviceID string,
	tenantID string,
	resource string,
	action string,
) (bool, error) {
	// 构建主体标识（服务）
	subject := fmt.Sprintf("service:%s", serviceID)

	// 构建资源标识
	object := fmt.Sprintf("resource:%s", resource)

	// 调用 Casbin 检查权限
	allowed, err := g.enforcer.Enforce(subject, tenantID, object, action)
	if err != nil {
		return false, fmt.Errorf("failed to check service permission: %w", err)
	}

	return allowed, nil
}

// BatchCheckPermissions 批量检查权限
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - tenantID: 租户ID
//   - permissions: 权限检查列表 [{resource, action}, ...]
//
// 返回:
//   - map[string]bool: 权限检查结果 {"resource:action": true/false}
//   - error: 错误信息
func (g *DomainGuard) BatchCheckPermissions(
	ctx context.Context,
	userID string,
	tenantID string,
	permissions []Permission,
) (map[string]bool, error) {
	results := make(map[string]bool, len(permissions))

	for _, perm := range permissions {
		allowed, err := g.CheckPermission(ctx, userID, tenantID, perm.Resource, perm.Action)
		if err != nil {
			return nil, fmt.Errorf("failed to check permission for %s:%s: %w", perm.Resource, perm.Action, err)
		}

		key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
		results[key] = allowed
	}

	return results, nil
}

// Permission 权限定义
type Permission struct {
	Resource string // 资源
	Action   string // 操作
}

// RegisterResource 注册资源映射（用于友好的错误提示）
func (g *DomainGuard) RegisterResource(key string, displayName string) {
	g.resourceMapping[key] = displayName
}

// GetResourceDisplayName 获取资源显示名称
func (g *DomainGuard) GetResourceDisplayName(key string) string {
	if displayName, exists := g.resourceMapping[key]; exists {
		return displayName
	}
	return key
}

// watchPolicyChanges 监听策略变更
func (g *DomainGuard) watchPolicyChanges(redisClient *redis.Client, topic string) {
	pubsub := redisClient.Subscribe(topic)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// 收到策略变更通知，清空版本缓存
		g.versionCache.Clear()

		// 可以在这里添加日志
		fmt.Printf("Policy changed: %s\n", msg.Payload)
	}
}

// GetCachedVersion 获取缓存的版本号
func (g *DomainGuard) GetCachedVersion(tenantID string) (int64, bool) {
	return g.versionCache.Get(tenantID)
}

// SetCachedVersion 设置缓存的版本号
func (g *DomainGuard) SetCachedVersion(tenantID string, version int64) {
	g.versionCache.Set(tenantID, version)
}
