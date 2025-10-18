// Package dto 策略相关的 DTO 定义
package dto

// AddPolicyRequest 添加策略规则请求
type AddPolicyRequest struct {
	RoleID     uint64 `json:"role_id" binding:"required"`
	ResourceID uint64 `json:"resource_id" binding:"required"`
	Action     string `json:"action" binding:"required"`
	ChangedBy  string `json:"changed_by" binding:"required"`
	Reason     string `json:"reason"`
}

// RemovePolicyRequest 移除策略规则请求
type RemovePolicyRequest struct {
	RoleID     uint64 `json:"role_id" binding:"required"`
	ResourceID uint64 `json:"resource_id" binding:"required"`
	Action     string `json:"action" binding:"required"`
	ChangedBy  string `json:"changed_by" binding:"required"`
	Reason     string `json:"reason"`
}

// PolicyRuleResponse 策略规则响应
type PolicyRuleResponse struct {
	Subject string `json:"subject"`
	Domain  string `json:"domain"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

// PolicyVersionResponse 策略版本响应
type PolicyVersionResponse struct {
	TenantID  string `json:"tenant_id"`
	Version   int64  `json:"version"`
	ChangedBy string `json:"changed_by"`
	Reason    string `json:"reason"`
}
