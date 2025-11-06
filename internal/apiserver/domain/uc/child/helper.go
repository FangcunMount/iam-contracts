package child

// toChildPointers 现在直接接受仓储返回的指针切片，做简单的 nil/空处理并返回同一切片
func toChildPointers(children []*Child) []*Child {
	if len(children) == 0 {
		return []*Child{}
	}
	return children
}

// Latest 从指针切片中选择 ID 最大的儿童（最新插入或最大雪花 id）
func Latest(children []*Child) *Child {
	if len(children) == 0 {
		return nil
	}

	var selected *Child
	var maxID uint64
	for _, c := range children {
		if c == nil {
			continue
		}
		value := c.ID.ToUint64()
		if selected == nil || value > maxID {
			selected = c
			maxID = value
		}
	}

	return selected
}
