package role

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// Role 角色领域对象（聚合根）
type Role struct {
	ID                   meta.ID
	ManagementProtection ManagementProtection
	Name                 Name   // 角色名称
	DisplayName          string // 显示名称

	Description string // 描述
}

// NewRole 创建新角色。
func NewRole(name, displayName string, opts ...RoleOption) (Role, error) {
	roleName, err := NewName(name)
	if err != nil {
		return Role{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return Role{}, perrors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	role := Role{
		Name:                 roleName,
		ManagementProtection: ManagementStandard,
		DisplayName:          displayName,
	}
	for _, opt := range opts {
		opt(&role)
	}
	if err := role.ManagementProtection.Validate(); err != nil {
		return Role{}, err
	}
	return role, nil
}

// RoleOption 角色选项
type RoleOption func(*Role)

func WithID(id meta.ID) RoleOption           { return func(r *Role) { r.ID = id } }
func WithDescription(desc string) RoleOption { return func(r *Role) { r.Description = desc } }

func (r Role) NameString() string {
	return r.Name.String()
}

func (r *Role) Rename(displayName string) error {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	r.DisplayName = displayName
	return nil
}

func (r *Role) ChangeDescription(description string) {
	r.Description = description
}
