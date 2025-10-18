package role

import "github.com/fangcun-mount/iam-contracts/pkg/util/idutil"

// Role 角色领域对象（聚合根）
type Role struct {
	ID          RoleID
	Name        string // 角色名称
	DisplayName string // 显示名称
	TenantID    string // 租户ID
	Description string // 描述
}

// NewRole 创建新角色
func NewRole(name, displayName, tenantID string, opts ...RoleOption) Role {
	role := Role{
		Name:        name,
		DisplayName: displayName,
		TenantID:    tenantID,
	}
	for _, opt := range opts {
		opt(&role)
	}
	return role
}

// RoleOption 角色选项
type RoleOption func(*Role)

func WithID(id RoleID) RoleOption            { return func(r *Role) { r.ID = id } }
func WithDescription(desc string) RoleOption { return func(r *Role) { r.Description = desc } }

// Key 返回 Casbin 中的角色标识
func (r *Role) Key() string {
	return "role:" + r.Name
}

// RoleID 角色ID值对象
type RoleID idutil.ID

func NewRoleID(value uint64) RoleID {
	return RoleID(idutil.NewID(value))
}

func (id RoleID) Uint64() uint64 {
	return idutil.ID(id).Uint64()
}

func (id RoleID) String() string {
	return idutil.ID(id).String()
}
