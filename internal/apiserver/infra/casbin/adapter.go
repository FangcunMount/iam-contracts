package casbin

import (
	"context"
	"sync"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// CasbinAdapter Casbin 适配器实现
type CasbinAdapter struct {
	enforcer *casbin.CachedEnforcer
	mu       sync.RWMutex
}

var _ domain.CasbinAdapter = (*CasbinAdapter)(nil)

// NewCasbinAdapter 创建 Casbin 适配器
func NewCasbinAdapter(db *gorm.DB, modelPath string) (domain.CasbinAdapter, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	enforcer, err := casbin.NewCachedEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}

	// 启用自动保存（每次策略变更自动持久化）
	enforcer.EnableAutoSave(true)

	// 加载策略
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	return &CasbinAdapter{
		enforcer: enforcer,
	}, nil
}

// AddPolicy 添加 p 规则
func (c *CasbinAdapter) AddPolicy(ctx context.Context, rules ...domain.PolicyRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, rule := range rules {
		_, err := c.enforcer.AddPolicy(rule.Sub, rule.Dom, rule.Obj, rule.Act)
		if err != nil {
			return err
		}
	}
	return nil
}

// RemovePolicy 删除 p 规则
func (c *CasbinAdapter) RemovePolicy(ctx context.Context, rules ...domain.PolicyRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, rule := range rules {
		_, err := c.enforcer.RemovePolicy(rule.Sub, rule.Dom, rule.Obj, rule.Act)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddGroupingPolicy 添加 g 规则
func (c *CasbinAdapter) AddGroupingPolicy(ctx context.Context, rules ...domain.GroupingRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, rule := range rules {
		_, err := c.enforcer.AddGroupingPolicy(rule.Sub, rule.Role, rule.Dom)
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveGroupingPolicy 删除 g 规则
func (c *CasbinAdapter) RemoveGroupingPolicy(ctx context.Context, rules ...domain.GroupingRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, rule := range rules {
		_, err := c.enforcer.RemoveGroupingPolicy(rule.Sub, rule.Role, rule.Dom)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPoliciesByRole 获取角色的所有 p 规则
func (c *CasbinAdapter) GetPoliciesByRole(ctx context.Context, role, domainStr string) ([]domain.PolicyRule, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	policies, err := c.enforcer.GetFilteredPolicy(0, role, domainStr)
	if err != nil {
		return nil, err
	}
	rules := make([]domain.PolicyRule, 0, len(policies))

	for _, p := range policies {
		if len(p) >= 4 {
			rules = append(rules, domain.PolicyRule{
				Sub: p[0],
				Dom: p[1],
				Obj: p[2],
				Act: p[3],
			})
		}
	}

	return rules, nil
}

// GetGroupingsBySubject 获取主体的所有 g 规则
func (c *CasbinAdapter) GetGroupingsBySubject(ctx context.Context, subject, domainStr string) ([]domain.GroupingRule, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	groupings, err := c.enforcer.GetFilteredGroupingPolicy(0, subject, "", domainStr)
	if err != nil {
		return nil, err
	}
	rules := make([]domain.GroupingRule, 0, len(groupings))

	for _, g := range groupings {
		if len(g) >= 3 {
			rules = append(rules, domain.GroupingRule{
				Sub:  g[0],
				Role: g[1],
				Dom:  g[2],
			})
		}
	}

	return rules, nil
}

// LoadPolicy 重新加载策略（用于缓存刷新）
func (c *CasbinAdapter) LoadPolicy(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.enforcer.LoadPolicy()
}

// Enforcer 获取 Enforcer 实例（用于 PEP）
func (c *CasbinAdapter) Enforcer() *casbin.CachedEnforcer {
	return c.enforcer
}

// InvalidateCache 清除缓存
func (c *CasbinAdapter) InvalidateCache() {
	_ = c.enforcer.InvalidateCache()
}
