package service

import "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"

// GuardianshipManager 监护人管理服务接口
type GuardianshipManager interface {
	AddGuardian(childID user.ChildID, userID user.UserID, relation user.Relation) error
	RemoveGuardian(childID user.ChildID, userID user.UserID) error
}

// PaternityExaminer 亲子鉴定服务接口
type PaternityExaminer interface {
	// IsGuardian 检查是否为监护人
	IsGuardian(childID user.ChildID, userID user.UserID) (ok bool, err error)
}
