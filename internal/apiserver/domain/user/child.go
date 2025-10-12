package user

import (
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// Child 孩子（儿童档案）
type Child struct {
	ID       ChildID
	Name     string
	IDCard   meta.IDCard
	Gender   meta.Gender
	Birthday time.Time
	Height   meta.Height
	Weight   meta.Weight
}

// Rename 重命名
func (c *Child) Rename(name string) { c.Name = name }

// UpdateIDCard 更新身份证
func (c *Child) UpdateIDCard(idc meta.IDCard) { c.IDCard = idc }

// UpdateProfile 更新基本信息
func (c *Child) UpdateProfile(g meta.Gender, day time.Time) {
	c.Gender, c.Birthday = g, day
}

// UpdateHeight 更新身高体重
func (c *Child) UpdateHeightWeight(h meta.Height, w meta.Weight) {
	c.Height, c.Weight = h, w
}
