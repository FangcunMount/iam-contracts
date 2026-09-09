package authorization

import (
	"sort"
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type Reason string

const (
	ReasonAllowed          Reason = "allowed"
	ReasonNotMatched       Reason = "not_matched"
	ReasonAttributeMissing Reason = "attribute_missing"
)

const (
	DenyCodePolicyNotMatched = "policy_not_matched"
	DenyCodeAttributeMissing = "attribute_missing"
)

// Decision 一次授权判定的结果及其依据，不是持久化授权事实。
type Decision struct {
	// ---- 判定结果 ----
	Allowed  bool   // 是否允许本次访问
	Reason   Reason // 判定原因
	DenyCode string // 拒绝码，允许时为空

	// ---- 命中依据 ----
	MatchedGrantID meta.ID // 允许访问时命中的权限授予 ID
	MatchedRole    string  // 命中授权所属角色的稳定业务名称

	// ---- 判定版本 ----
	PolicyVersion int64 // 本次判定使用的授权事实版本

	// ---- 拒绝依据 ----
	MissingAttributeKeys []string // 因缺失属性拒绝时报告的属性键

	// ---- 判定时间 ----
	EvaluatedAt time.Time // 本次决策的求值时间
}

func Allow(grantID meta.ID, role string, policyVersion int64, at time.Time) Decision {
	if at.IsZero() {
		at = time.Now()
	}
	return Decision{
		Allowed: true, Reason: ReasonAllowed, MatchedGrantID: grantID,
		MatchedRole: role, PolicyVersion: policyVersion, EvaluatedAt: at,
	}
}

func Deny(policyVersion int64, missing []string, at time.Time) Decision {
	if at.IsZero() {
		at = time.Now()
	}
	missing = uniqueSorted(missing)
	if len(missing) > 0 {
		return Decision{
			Reason: ReasonAttributeMissing, DenyCode: DenyCodeAttributeMissing,
			PolicyVersion: policyVersion, MissingAttributeKeys: missing, EvaluatedAt: at,
		}
	}
	return Decision{
		Reason: ReasonNotMatched, DenyCode: DenyCodePolicyNotMatched,
		PolicyVersion: policyVersion, EvaluatedAt: at,
	}
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
