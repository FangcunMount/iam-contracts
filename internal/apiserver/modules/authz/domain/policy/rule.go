package policy

// PolicyRule 策略规则值对象（p 规则）
type PolicyRule struct {
	Sub string // 主体（角色）
	Dom string // 域（租户）
	Obj string // 对象（资源）
	Act string // 动作
}

// NewPolicyRule 创建策略规则
func NewPolicyRule(sub, dom, obj, act string) PolicyRule {
	return PolicyRule{
		Sub: sub,
		Dom: dom,
		Obj: obj,
		Act: act,
	}
}

// GroupingRule 分组规则值对象（g 规则：用户/组 → 角色）
type GroupingRule struct {
	Sub  string // 主体（用户/组）
	Dom  string // 域（租户）
	Role string // 角色
}

// NewGroupingRule 创建分组规则
func NewGroupingRule(sub, dom, role string) GroupingRule {
	return GroupingRule{
		Sub:  sub,
		Dom:  dom,
		Role: role,
	}
}
