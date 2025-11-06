package child

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
