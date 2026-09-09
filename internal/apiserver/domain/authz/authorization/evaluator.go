// Package authorization owns the domain language and policies used to produce
// an authorization decision from immutable authorization facts.
package authorization

import (
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// EvaluationContext 是为一次鉴权准备的授权事实投影，不持久化。
// 调用方提供已验证、未撤销且按所属角色分组的候选授权，求值期间按只读方式使用。
type EvaluationContext struct {
	EffectiveRoles []meta.ID                            // 主体直接及继承获得的有效角色 ID
	RoleNames      map[meta.ID]role.Name                // 角色 ID 到稳定业务名称的映射
	GrantsByRole   map[meta.ID][]*permissiongrant.Grant // 按角色 ID 分组的有效候选授权
	Resource       *resource.Resource                   // 请求对应的资源定义，未注册时可为空
	PolicyVersion  int64                                // 本次判定使用的授权事实版本
}

// Evaluator 匹配权限授予与条件并生成授权决策，不负责持久化或快照管理。
type Evaluator struct{}

// NewEvaluator 创建评估器
func NewEvaluator() Evaluator { return Evaluator{} }

// Evaluate 求值访问请求；未匹配是正常拒绝，属性契约不合法时返回错误。
func (Evaluator) Evaluate(
	request Request,
	context EvaluationContext,
	evaluatedAt time.Time,
) (Decision, error) {
	if context.Resource != nil {
		if err := ValidateAttributes(context.Resource.AttributeSchema, request.Object.Attributes); err != nil {
			return Decision{}, err
		}
	} else if len(request.Object.Attributes) > 0 {
		return Decision{}, perrors.WithCode(
			code.ErrInvalidArgument,
			"object attributes require a registered resource",
		)
	}

	missing := make([]string, 0)
	for _, roleID := range context.EffectiveRoles {
		for _, grant := range context.GrantsByRole[roleID] {
			if grant == nil {
				return Decision{}, perrors.WithCode(
					code.ErrInvalidArgument,
					"authorization candidate grant is required",
				)
			}
			if !grant.CoversResource(request.ResourceKey) || !grant.MatchesAction(request.Action) {
				continue
			}
			evaluation, err := grant.Evaluate(request.Object.Attributes)
			if err != nil {
				return Decision{}, err
			}
			if evaluation.Matched {
				return Allow(grant.ID, context.RoleNames[roleID].String(), context.PolicyVersion, evaluatedAt), nil
			}
			missing = append(missing, evaluation.MissingAttributeKeys...)
		}
	}
	return Deny(context.PolicyVersion, missing, evaluatedAt), nil
}
