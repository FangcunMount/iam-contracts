package resource

import (
	"regexp"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

var concreteActionSyntax = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Action 表达具体动作或全部动作；资源、请求和普通赋权须校验为具体动作。
type Action string

// WildcardAction 表达全部动作，仅可信系统赋权允许使用。
const WildcardAction Action = "*"

// NewAction 规范化动作，接受具体动作或 *，不支持正则或前缀表达式。
func NewAction(value string) (Action, error) {
	a := Action(strings.TrimSpace(value))
	if a.IsWildcard() {
		return a, nil
	}
	if err := a.ValidateConcrete(); err != nil {
		return "", err
	}
	return a, nil
}

func (a Action) String() string { return string(a) }

// IsWildcard 判断是否为全部动作。
func (a Action) IsWildcard() bool { return a == WildcardAction }

// ValidateConcrete 校验具体动作的完整格式，包括直接类型转换产生的值。
func (a Action) ValidateConcrete() error {
	if a == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "action is required")
	}
	if !concreteActionSyntax.MatchString(a.String()) {
		return perrors.WithCode(code.ErrInvalidArgument, "action must be a concrete operation")
	}
	return nil
}

// Matches 判断授予动作是否覆盖请求动作；请求的具体动作约束由入口校验。
func (a Action) Matches(requested Action) bool {
	granted := strings.TrimSpace(a.String())
	concrete := strings.TrimSpace(requested.String())
	if granted == "" || concrete == "" {
		return false
	}
	return granted == WildcardAction.String() || granted == concrete
}
