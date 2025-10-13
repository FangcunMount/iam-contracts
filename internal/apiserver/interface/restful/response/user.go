package response

// UserResponse 用户响应
type UserResponse struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email,omitempty"`
	IDCard   string `json:"id_card,omitempty"`
	Status   string `json:"status"`
	StatusID uint8  `json:"status_id"`
}
