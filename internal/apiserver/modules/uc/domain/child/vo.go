package child

import "github.com/FangcunMount/component-base/pkg/util/idutil"

// ChildID 儿童唯一标识
type ChildID = idutil.ID

// NewChildID 创建儿童ID
func NewChildID(value uint64) ChildID {
	return idutil.NewID(value)
}
