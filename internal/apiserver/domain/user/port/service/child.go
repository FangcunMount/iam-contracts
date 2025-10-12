package service

import (
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ChildRegister 儿童档案注册服务接口
type ChildRegister interface {
	Register(name string, gender meta.Gender, birthday meta.Birthday) (child *user.Child, err error)
}

// ChildProfileEditor 儿童档案编辑服务接口
type ChildProfileEditor interface {
	Rename(childID user.ChildID, name string) error
	UpdateIDCard(childID user.ChildID, idCard meta.IDCard) error
	UpdateProfile(childID user.ChildID, gender meta.Gender, birthday meta.Birthday) error
	UpdateHeightWeight(childID user.ChildID, height meta.Height, weight meta.Weight) error
}

// SimilarChildFinder 相似儿童档案查找服务接口
type SimilarChildFinder interface {
	FindChilds(name string, gender meta.Gender, birthday meta.Birthday) (children []*user.Child, err error)
}
