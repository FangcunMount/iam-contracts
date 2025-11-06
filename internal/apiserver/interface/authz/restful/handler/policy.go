// Package handler 策略管理处理器
package handler

import (
	"strconv"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy/port/driving"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authz/restful/dto"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/gin-gonic/gin"
)

// PolicyHandler 策略处理器
type PolicyHandler struct {
	commander driving.PolicyCommander
	queryer   driving.PolicyQueryer
}

// NewPolicyHandler 创建策略处理器
func NewPolicyHandler(commander driving.PolicyCommander, queryer driving.PolicyQueryer) *PolicyHandler {
	return &PolicyHandler{
		commander: commander,
		queryer:   queryer,
	}
}

// AddPolicyRule 添加策略规则
// @Summary 添加策略规则
// @Tags Authorization-Policies
// @Accept json
// @Produce json
// @Param request body dto.AddPolicyRequest true "添加策略请求"
// @Success 200 {object} dto.Response
// @Router /authz/policies [post]
func (h *PolicyHandler) AddPolicyRule(c *gin.Context) {
	var req dto.AddPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)
	changedBy := getUserID(c)

	cmd := driving.AddPolicyRuleCommand{
		RoleID:     req.RoleID,
		ResourceID: resource.NewResourceID(req.ResourceID),
		Action:     req.Action,
		TenantID:   tenantID,
		ChangedBy:  changedBy,
		Reason:     req.Reason,
	}

	err := h.commander.AddPolicyRule(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// RemovePolicyRule 移除策略规则
// @Summary 移除策略规则
// @Tags Authorization-Policies
// @Accept json
// @Produce json
// @Param request body dto.RemovePolicyRequest true "移除策略请求"
// @Success 200 {object} dto.Response
// @Router /authz/policies [delete]
func (h *PolicyHandler) RemovePolicyRule(c *gin.Context) {
	var req dto.RemovePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)
	changedBy := getUserID(c)

	cmd := driving.RemovePolicyRuleCommand{
		RoleID:     req.RoleID,
		ResourceID: resource.NewResourceID(req.ResourceID),
		Action:     req.Action,
		TenantID:   tenantID,
		ChangedBy:  changedBy,
		Reason:     req.Reason,
	}

	err := h.commander.RemovePolicyRule(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// GetPoliciesByRole 获取角色的策略列表
// @Summary 获取角色的策略列表
// @Tags Authorization-Policies
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} dto.Response{data=[]dto.PolicyRuleResponse}
// @Router /authz/roles/{id}/policies [get]
func (h *PolicyHandler) GetPoliciesByRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "角色ID格式错误"))
		return
	}

	tenantID := getTenantID(c)

	query := driving.GetPoliciesByRoleQuery{
		RoleID:   roleID,
		TenantID: tenantID,
	}

	rules, err := h.queryer.GetPoliciesByRole(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	policyRules := make([]dto.PolicyRuleResponse, 0, len(rules))
	for _, rule := range rules {
		policyRules = append(policyRules, dto.PolicyRuleResponse{
			Subject: rule.Sub,
			Domain:  rule.Dom,
			Object:  rule.Obj,
			Action:  rule.Act,
		})
	}

	success(c, policyRules)
}

// GetCurrentVersion 获取当前策略版本
// @Summary 获取当前策略版本
// @Tags Authorization-Policies
// @Produce json
// @Success 200 {object} dto.Response{data=dto.PolicyVersionResponse}
// @Router /authz/policies/version [get]
func (h *PolicyHandler) GetCurrentVersion(c *gin.Context) {
	tenantID := getTenantID(c)

	query := driving.GetCurrentVersionQuery{
		TenantID: tenantID,
	}

	version, err := h.queryer.GetCurrentVersion(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, dto.PolicyVersionResponse{
		TenantID:  version.TenantID,
		Version:   version.Version,
		ChangedBy: version.ChangedBy,
		Reason:    version.Reason,
	})
}
