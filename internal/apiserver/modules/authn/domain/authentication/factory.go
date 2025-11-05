package authentication

import "context"

// 认证策略（领域服务接口）
type AuthStrategy interface {
	Kind() Scenario
	Authenticate(ctx context.Context, in AuthInput) (AuthDecision, error)
}

// AuthStrategyFactory 认证器工厂函数
type AuthStrategyFactory func() AuthStrategy

// 注册表
var registry = make(map[Scenario]AuthStrategyFactory)

// RegisterAuthStrategyFactory 注册认证器工厂函数
func RegisterAuthStrategyFactory(kind Scenario, factory AuthStrategyFactory) {
	if _, exists := registry[kind]; exists {
		return
	}

	registry[kind] = factory
}
