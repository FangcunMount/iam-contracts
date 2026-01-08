package suggest

// Term 表示 suggest 的结果项
type Term struct {
	Name   string `json:"name"`
	ID     int64  `json:"id"`
	Mobile string `json:"mobile"`
	Weight int    `json:"weight"`
}
