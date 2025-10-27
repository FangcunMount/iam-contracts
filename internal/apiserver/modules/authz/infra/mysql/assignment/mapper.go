package assignment

import (
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
)

// Mapper Assignment BO 和 PO 转换器
type Mapper struct{}

// NewMapper 创建 Mapper
func NewMapper() *Mapper {
	return &Mapper{}
}

// ToBO 将 PO 转换为 BO
func (m *Mapper) ToBO(po *AssignmentPO) *assignment.Assignment {
	if po == nil {
		return nil
	}

	a := &assignment.Assignment{
		ID:          assignment.AssignmentID(po.ID),
		SubjectType: assignment.SubjectType(po.SubjectType),
		SubjectID:   po.SubjectID,
		RoleID:      po.RoleID,
		TenantID:    po.TenantID,
		GrantedBy:   po.GrantedBy,
	}

	return a
}

// ToPO 将 BO 转换为 PO
func (m *Mapper) ToPO(bo *assignment.Assignment) *AssignmentPO {
	if bo == nil {
		return nil
	}

	po := &AssignmentPO{
		SubjectType: string(bo.SubjectType),
		SubjectID:   bo.SubjectID,
		RoleID:      bo.RoleID,
		TenantID:    bo.TenantID,
		GrantedBy:   bo.GrantedBy,
	}
	po.ID = idutil.NewID(bo.ID.Uint64())

	return po
}

// ToBOList 将 PO 列表转换为 BO 列表
func (m *Mapper) ToBOList(pos []*AssignmentPO) []*assignment.Assignment {
	if len(pos) == 0 {
		return nil
	}

	bos := make([]*assignment.Assignment, 0, len(pos))
	for _, po := range pos {
		if bo := m.ToBO(po); bo != nil {
			bos = append(bos, bo)
		}
	}

	return bos
}
