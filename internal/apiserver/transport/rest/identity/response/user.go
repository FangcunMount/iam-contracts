package response

import "time"

// VerifiedContactResponse 用户联系方式展示
type VerifiedContactResponse struct {
	Type       string     `json:"type"`
	Value      string     `json:"value"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

// UserResponse 用户响应
type UserResponse struct {
	Permissions []PermissionResponse      `json:"permissions"`
	ID          string                    `json:"id"`
	Status      string                    `json:"status"`
	Nickname    string                    `json:"nickname,omitempty"`
	Avatar      string                    `json:"avatar,omitempty"`
	Contacts    []VerifiedContactResponse `json:"contacts,omitempty"`
	Roles       []string                  `json:"roles,omitempty"`
	CreatedAt   *time.Time                `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time                `json:"updatedAt,omitempty"`
}

// PermissionResponse contains the current user's action facts for navigation only.
type PermissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Mode     string `json:"mode"`
}
