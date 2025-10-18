package guardianship

import (
	child "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// GuardianshipMapper 监护关系映射器
type GuardianshipMapper struct{}

// NewGuardianshipMapper 创建监护关系映射器
func NewGuardianshipMapper() *GuardianshipMapper {
	return &GuardianshipMapper{}
}

// ToPO 将领域模型转换为持久化对象
func (m *GuardianshipMapper) ToPO(gBO *domain.Guardianship) *GuardianshipPO {
	if gBO == nil {
		return nil
	}

	po := &GuardianshipPO{
		UserID:        gBO.User,
		ChildID:       gBO.Child,
		Relation:      string(gBO.Rel),
		EstablishedAt: gBO.EstablishedAt,
		RevokedAt:     gBO.RevokedAt,
	}

	if gBO.ID > 0 {
		po.ID = idutil.NewID(uint64(gBO.ID))
	}

	return po
}

// ToBO 将持久化对象转换为领域模型
func (m *GuardianshipMapper) ToBO(po *GuardianshipPO) *domain.Guardianship {
	if po == nil {
		return nil
	}

	gBO := &domain.Guardianship{
		ID:            int64(po.ID.Uint64()),
		User:          user.UserID(po.UserID),
		Child:         child.ChildID(po.ChildID),
		Rel:           domain.Relation(po.Relation),
		EstablishedAt: po.EstablishedAt,
		RevokedAt:     po.RevokedAt,
	}

	return gBO
}

// ToBOs 将持久化对象列表转换为领域模型列表
func (m *GuardianshipMapper) ToBOs(pos []*GuardianshipPO) []*domain.Guardianship {
	if pos == nil {
		return nil
	}

	var bos []*domain.Guardianship
	for _, po := range pos {
		bos = append(bos, m.ToBO(po))
	}

	return bos
}

// ToPOs 将领域模型列表转换为持久化对象列表
func (m *GuardianshipMapper) ToPOs(bos []*domain.Guardianship) []*GuardianshipPO {
	if bos == nil {
		return nil
	}

	var pos []*GuardianshipPO
	for _, bo := range bos {
		pos = append(pos, m.ToPO(bo))
	}

	return pos
}
