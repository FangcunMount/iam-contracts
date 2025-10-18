// Package dto 赋权相关的 DTO 定义
package dto

// GrantRequest 授权请求
type GrantRequest struct {
	SubjectType string `json:"subject_type" binding:"required,oneof=user group"`
	SubjectID   string `json:"subject_id" binding:"required"`
	RoleID      uint64 `json:"role_id" binding:"required"`
	GrantedBy   string `json:"granted_by" binding:"required"`
}

// RevokeRequest 撤销授权请求
type RevokeRequest struct {
	SubjectType string `json:"subject_type" binding:"required,oneof=user group"`
	SubjectID   string `json:"subject_id" binding:"required"`
	RoleID      uint64 `json:"role_id" binding:"required"`
}

// AssignmentResponse 赋权响应
type AssignmentResponse struct {
	ID          uint64 `json:"id"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	RoleID      uint64 `json:"role_id"`
	TenantID    string `json:"tenant_id"`
	GrantedBy   string `json:"granted_by"`
}

// ListAssignmentQuery 列出赋权查询参数
type ListAssignmentQuery struct {
	SubjectType string `form:"subject_type"`
	SubjectID   string `form:"subject_id"`
	RoleID      uint64 `form:"role_id"`
}
