package request

import "strings"

// RegisterUserRequest 注册用户请求
type RegisterUserRequest struct {
	Name         string `json:"name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	IDCardName   string `json:"id_card_name" binding:"required"`
	IDCardNumber string `json:"id_card_number" binding:"required"`
}

// UpdateContactRequest 更新联系方式请求
type UpdateContactRequest struct {
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}

// IsEmpty 判断是否包含有效字段
func (r UpdateContactRequest) IsEmpty() bool {
	return strings.TrimSpace(r.Phone) == "" && strings.TrimSpace(r.Email) == ""
}

// UpdateIDCardRequest 更新身份证请求
type UpdateIDCardRequest struct {
	Name   string `json:"id_card_name,omitempty"`
	Number string `json:"id_card_number" binding:"required"`
}

// ChangeStatusRequest 修改状态请求
type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
