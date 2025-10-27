package role

import (
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	base "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
)

// Mapper 领域对象与PO的转换器
type Mapper struct{}

// NewMapper 创建转换器
func NewMapper() *Mapper {
	return &Mapper{}
}

// ToRoleBO 将PO转换为领域对象
func (m *Mapper) ToRoleBO(po *RolePO) *domain.Role {
	if po == nil {
		return nil
	}
	return &domain.Role{
		ID:          domain.RoleID(po.ID),
		Name:        po.Name,
		DisplayName: po.DisplayName,
		TenantID:    po.TenantID,
		Description: po.Description,
	}
}

// ToRolePO 将领域对象转换为PO
func (m *Mapper) ToRolePO(role *domain.Role) *RolePO {
	if role == nil {
		return nil
	}
	return &RolePO{
		AuditFields: base.AuditFields{
			ID: idutil.ID(role.ID),
		},
		Name:        role.Name,
		DisplayName: role.DisplayName,
		TenantID:    role.TenantID,
		Description: role.Description,
	}
}
