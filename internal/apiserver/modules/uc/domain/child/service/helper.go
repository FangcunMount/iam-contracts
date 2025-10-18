package service

import domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"

// toChildPointers 现在直接接受仓储返回的指针切片，做简单的 nil/空处理并返回同一切片
func toChildPointers(children []*domain.Child) []*domain.Child {
	if len(children) == 0 {
		return []*domain.Child{}
	}
	return children
}

// Latest 从指针切片中选择 ID 最大的儿童（最新插入或最大雪花 id）
func Latest(children []*domain.Child) *domain.Child {
	if len(children) == 0 {
		return nil
	}

	var selected *domain.Child
	var maxID uint64
	for _, c := range children {
		if c == nil {
			continue
		}
		value := c.ID.Uint64()
		if selected == nil || value > maxID {
			selected = c
			maxID = value
		}
	}

	return selected
}
