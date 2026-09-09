package role

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// ManagementProtection 仅约束角色管理，不划分授权空间。
type ManagementProtection string

const (
	ManagementStandard      ManagementProtection = "standard"                   // 普通角色的管理保护类别
	ManagementProtected     ManagementProtection = "protected"                  // 受保护角色的管理保护类别
	ManageProtectedResource                      = "iam:authz:collection:roles" // 管理受保护角色所检查的资源键
	ManageProtectedAction                        = "manage_protected"           // 管理受保护角色所检查的动作
)

// Validate 验证管理保护属性
func (p ManagementProtection) Validate() error {
	if p != ManagementStandard && p != ManagementProtected {
		return perrors.WithCode(code.ErrInvalidArgument, "无效的角色管理保护属性")
	}
	return nil
}

// WithManagementProtection 设置管理保护属性
func WithManagementProtection(p ManagementProtection) RoleOption {
	return func(r *Role) { r.ManagementProtection = p }
}

// IsProtected 判断角色是否受保护
func (r Role) IsProtected() bool { return r.ManagementProtection == ManagementProtected }

// RequiresProtectedRole 按实际覆盖范围识别敏感能力，包括资源和动作通配符
func RequiresProtectedRole(resourceKey resource.Key, action resource.Action) bool {
	for _, capability := range []struct{ resource, action string }{
		{ManageProtectedResource, ManageProtectedAction},
		{"iam:authz:collection:resources", "create"},
		{"iam:authz:collection:resources", "update"},
		{"iam:authz:collection:resources", "delete"},
		{"iam:identity:collection:profiles", "list_all"},
		{"iam:identity:collection:profiles", "search_by_mobile_all"},
	} {
		if resourceKey.Covers(resource.Key(capability.resource)) && (action.IsWildcard() || action.String() == capability.action) {
			return true
		}
	}
	return false
}

// ValidateGrant 校验角色的管理保护属性是否允许承载目标权限。
func (r Role) ValidateGrant(resourceKey resource.Key, action resource.Action) error {
	if err := r.ManagementProtection.Validate(); err != nil {
		return err
	}
	if !r.IsProtected() && RequiresProtectedRole(resourceKey, action) {
		return perrors.WithCode(code.ErrPermissionDenied, "敏感授权只能授予受保护角色")
	}
	return nil
}
