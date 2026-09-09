package roleinheritance

import (
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Inheritance 角色继承实体
type Inheritance struct {
	ID meta.ID // 继承事实ID

	// ---- 继承事实 ----
	RoleID          meta.ID // 发起继承、获得权限的角色 ID
	InheritedRoleID meta.ID // 被继承、提供权限的角色 ID

	// ---- 关系建立来源 ----
	GrantedBy string    // 关系建立者标识
	GrantedAt time.Time // 关系建立时间

	// ---- 撤销状态与记录版本 ----
	RevokedAt *time.Time // 撤销时间
	Version   uint32     // 继承记录版本，非全局授权事实版本
}

// New 创建新角色继承实体
func New(roleID, inheritedRoleID meta.ID, grantedBy string) (Inheritance, error) {
	if roleID.IsZero() || inheritedRoleID.IsZero() {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "role ids are required")
	}
	if roleID == inheritedRoleID {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "role cannot inherit itself")
	}
	grantedBy = strings.TrimSpace(grantedBy)
	if grantedBy == "" {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "granted by is required")
	}
	return Inheritance{
		RoleID:          roleID,
		InheritedRoleID: inheritedRoleID,
		GrantedBy:       grantedBy,
		Version:         1,
	}, nil
}

// Revoke 撤销角色继承实体
func (i *Inheritance) Revoke(at time.Time) error {
	if i == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "role inheritance is required")
	}
	if i.RevokedAt != nil {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	i.RevokedAt = &at
	return nil
}

// RestoreOptions 从持久化数据恢复继承关系所需的标识、时间和状态。
type RestoreOptions struct {
	// ---- 持久化恢复信息 ----
	ID        meta.ID    // 继承事实ID
	GrantedAt time.Time  // 关系建立时间
	RevokedAt *time.Time // 撤销时间
	Version   uint32     // 继承记录版本，非全局授权事实版本
}

// Restore 恢复角色继承实体
func Restore(roleID, inheritedRoleID meta.ID, grantedBy string, options RestoreOptions) (Inheritance, error) {
	inheritance, err := New(roleID, inheritedRoleID, grantedBy)
	if err != nil {
		return Inheritance{}, err
	}
	inheritance.ID = options.ID
	inheritance.GrantedAt = options.GrantedAt
	inheritance.RevokedAt = options.RevokedAt
	if options.Version > 0 {
		inheritance.Version = options.Version
	}
	return inheritance, nil
}

// IsActive 判断角色继承实体是否活动
func (i Inheritance) IsActive() bool { return i.RevokedAt == nil }
