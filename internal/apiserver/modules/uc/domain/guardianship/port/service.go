package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	guardianship "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
)

// GuardianshipManager 监护人管理服务接口
type GuardianshipManager interface {
	AddGuardian(ctx context.Context, childID child.ChildID, userID user.UserID, relation guardianship.Relation) error
	RemoveGuardian(ctx context.Context, childID child.ChildID, userID user.UserID) error
}

// PaternityExaminer 亲子鉴定服务接口
type PaternityExaminer interface {
	// IsGuardian 检查是否为监护人
	IsGuardian(ctx context.Context, childID child.ChildID, userID user.UserID) (ok bool, err error)
}

// GuardianshipQueryer 监护关系查询服务接口
type GuardianshipQueryer interface {
	// FindByUserIDAndChildID 根据用户ID和儿童ID查询监护关系
	FindByUserIDAndChildID(ctx context.Context, userID user.UserID, childID child.ChildID) (*guardianship.Guardianship, error)
	// FindByUserIDAndChildName 根据用户ID和儿童姓名查询监护关系
	FindByUserIDAndChildName(ctx context.Context, userID user.UserID, childName string) ([]*guardianship.Guardianship, error)
	// FindListByChildID 列出儿童的所有监护人
	FindListByChildID(ctx context.Context, childID child.ChildID) (guardianships []*guardianship.Guardianship, err error)
	// FindListByUserID 列出用户监护的所有儿童
	FindListByUserID(ctx context.Context, userID user.UserID) (guardianships []*guardianship.Guardianship, err error)
}
