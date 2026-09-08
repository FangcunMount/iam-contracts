package role

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// ManagementProtection 仅约束角色管理，不划分授权空间。
type ManagementProtection string

const (
	ManagementStandard      ManagementProtection = "standard"
	ManagementProtected     ManagementProtection = "protected"
	ManageProtectedResource                      = "iam:authz:collection:roles"
	ManageProtectedAction                        = "manage_protected"
)

func (p ManagementProtection) Validate() error {
	if p != ManagementStandard && p != ManagementProtected {
		return perrors.WithCode(code.ErrInvalidArgument, "无效的角色管理保护属性")
	}
	return nil
}

func WithManagementProtection(p ManagementProtection) RoleOption {
	return func(r *Role) { r.ManagementProtection = p }
}

func (r Role) IsProtected() bool { return r.ManagementProtection == ManagementProtected }

// RequiresProtectedRole 按实际覆盖范围识别敏感能力，包括资源和动作通配符。
func RequiresProtectedRole(pattern resource.Pattern, action resource.ActionPattern) bool {
	for _, capability := range []struct{ resource, action string }{
		{ManageProtectedResource, ManageProtectedAction},
		{"iam:authz:collection:resources", "create"},
		{"iam:authz:collection:resources", "update"},
		{"iam:authz:collection:resources", "delete"},
		{"iam:identity:collection:profiles", "list_all"},
		{"iam:identity:collection:profiles", "search_by_mobile_all"},
	} {
		if pattern.Covers(resource.Pattern(capability.resource)) && (action.String() == "*" || action.String() == capability.action) {
			return true
		}
	}
	return false
}

func (r Role) ValidateGrant(pattern resource.Pattern, action resource.ActionPattern) error {
	if err := r.ManagementProtection.Validate(); err != nil {
		return err
	}
	if !r.IsProtected() && RequiresProtectedRole(pattern, action) {
		return perrors.WithCode(code.ErrPermissionDenied, "敏感授权只能授予受保护角色")
	}
	return nil
}
