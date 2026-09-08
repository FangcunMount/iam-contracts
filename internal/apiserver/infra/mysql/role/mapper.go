package role

import (
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
)

// Mapper 领域对象与PO的转换器
type Mapper struct{}

// NewMapper 创建转换器
func NewMapper() *Mapper {
	return &Mapper{}
}

// ToRoleBO 将PO转换为领域对象
func (m *Mapper) ToRoleBO(po *RolePO) (*domain.Role, error) {
	if po == nil {
		return nil, nil
	}
	role, err := domain.NewRole(
		po.Name,
		po.DisplayName,

		domain.WithID(po.ID),
		domain.WithManagementProtection(domain.ManagementProtection(po.ManagementProtection)),
		domain.WithDescription(po.Description),
	)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ToRolePO 将领域对象转换为PO
func (m *Mapper) ToRolePO(role *domain.Role) *RolePO {
	if role == nil {
		return nil
	}
	return &RolePO{
		AuditFields: base.AuditFields{
			ID: role.ID,
		},
		Name:                 role.NameString(),
		ManagementProtection: string(role.ManagementProtection),
		DisplayName:          role.DisplayName,

		Description: role.Description,
	}
}
