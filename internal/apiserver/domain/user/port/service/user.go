package service

import (
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// UserRegister 用户注册服务接口
type UserRegister interface {
	Register(name string, phone meta.Phone) (u *user.User, err error)
}

// UserStatusChanger 用户状态变更服务接口
type UserStatusChanger interface {
	Activate(userID user.UserID) error
	Deactivate(userID user.UserID) error
	Block(userID user.UserID) error
}

// UserProfileEditor 用户资料编辑服务接口
type UserProfileEditor interface {
	UpdateContact(userID user.UserID, phone meta.Phone, email meta.Email) error
	UpdateIDCard(userID user.UserID, idCard meta.IDCard) error
}
