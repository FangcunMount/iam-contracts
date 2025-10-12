package port

import (
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ChildRegister 儿童档案注册服务接口
type ChildRegister interface {
	Register(name string, gender meta.Gender, birthday meta.Birthday) (child *child.Child, err error)
}

// ChildProfileEditor 儿童档案编辑服务接口
type ChildProfileEditor interface {
	Rename(childID child.ChildID, name string) error
	UpdateIDCard(childID child.ChildID, idCard meta.IDCard) error
	UpdateProfile(childID child.ChildID, gender meta.Gender, birthday meta.Birthday) error
	UpdateHeightWeight(childID child.ChildID, height meta.Height, weight meta.Weight) error
}

// SimilarChildFinder 相似儿童档案查找服务接口
type SimilarChildFinder interface {
	FindChilds(name string, gender meta.Gender, birthday meta.Birthday) (children []*child.Child, err error)
}
