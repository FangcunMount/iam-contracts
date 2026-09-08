package authorization

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// RoleResolver 根据稳定角色 ID 解析主体的直接角色及继承角色。
type RoleResolver interface {
	DirectRoles(subject.Ref) ([]meta.ID, error)
	EffectiveRoles(subject.Ref) ([]meta.ID, error)
}
