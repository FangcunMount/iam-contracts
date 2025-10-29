// Package handler 角色分配处理器
package handler

import (
	"strconv"

	"github.com/FangcunMount/component-base/pkg/errors"
	domainAssignment "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/port/driving"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/interface/restful/dto"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/gin-gonic/gin"
)

// AssignmentHandler 角色分配处理器
type AssignmentHandler struct {
	commander driving.AssignmentCommander
	queryer   driving.AssignmentQueryer
}

// NewAssignmentHandler 创建角色分配处理器
func NewAssignmentHandler(commander driving.AssignmentCommander, queryer driving.AssignmentQueryer) *AssignmentHandler {
	return &AssignmentHandler{
		commander: commander,
		queryer:   queryer,
	}
}

// convertToSubjectType 将字符串转换为 SubjectType
func convertToSubjectType(s string) (domainAssignment.SubjectType, error) {
	switch s {
	case "user":
		return domainAssignment.SubjectTypeUser, nil
	case "group":
		return domainAssignment.SubjectTypeGroup, nil
	default:
		return "", errors.WithCode(code.ErrInvalidArgument, "无效的主体类型: %s", s)
	}
}

// GrantRole 授予角色
// @Summary 授予角色
// @Tags Authorization-Assignments
// @Accept json
// @Produce json
// @Param request body dto.GrantRequest true "授予角色请求"
// @Success 200 {object} dto.Response{data=dto.AssignmentResponse}
// @Router /authz/assignments/grant [post]
func (h *AssignmentHandler) GrantRole(c *gin.Context) {
	var req dto.GrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)
	grantedBy := getUserID(c)

	subjectType, err := convertToSubjectType(req.SubjectType)
	if err != nil {
		handleError(c, err)
		return
	}

	cmd := driving.GrantCommand{
		SubjectType: subjectType,
		SubjectID:   req.SubjectID,
		RoleID:      req.RoleID,
		TenantID:    tenantID,
		GrantedBy:   grantedBy,
	}

	grantedAssignment, err := h.commander.Grant(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toAssignmentResponse(grantedAssignment))
}

// RevokeRole 撤销角色
// @Summary 撤销角色
// @Tags Authorization-Assignments
// @Accept json
// @Produce json
// @Param request body dto.RevokeRequest true "撤销角色请求"
// @Success 200 {object} dto.Response
// @Router /authz/assignments/revoke [post]
func (h *AssignmentHandler) RevokeRole(c *gin.Context) {
	var req dto.RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)

	subjectType, err := convertToSubjectType(req.SubjectType)
	if err != nil {
		handleError(c, err)
		return
	}

	cmd := driving.RevokeCommand{
		SubjectType: subjectType,
		SubjectID:   req.SubjectID,
		RoleID:      req.RoleID,
		TenantID:    tenantID,
	}

	err = h.commander.Revoke(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// RevokeRoleByID 根据分配ID撤销角色
// @Summary 根据分配ID撤销角色
// @Tags Authorization-Assignments
// @Param id path string true "分配ID"
// @Success 200 {object} dto.Response
// @Router /authz/assignments/{id} [delete]
func (h *AssignmentHandler) RevokeRoleByID(c *gin.Context) {
	assignmentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "分配ID格式错误"))
		return
	}

	tenantID := getTenantID(c)

	cmd := driving.RevokeByIDCommand{
		AssignmentID: domainAssignment.NewAssignmentID(assignmentID),
		TenantID:     tenantID,
	}

	err = h.commander.RevokeByID(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// ListAssignmentsBySubject 列出主体的角色分配
// @Summary 列出主体的角色分配
// @Tags Authorization-Assignments
// @Produce json
// @Param subject_type query string true "主体类型" Enums(user, group)
// @Param subject_id query string true "主体ID"
// @Success 200 {object} dto.Response{data=[]dto.AssignmentResponse}
// @Router /authz/assignments/subject [get]
func (h *AssignmentHandler) ListAssignmentsBySubject(c *gin.Context) {
	subjectTypeStr := c.Query("subject_type")
	subjectID := c.Query("subject_id")

	if subjectTypeStr == "" || subjectID == "" {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "subject_type 和 subject_id 不能为空"))
		return
	}

	tenantID := getTenantID(c)

	subjectType, err := convertToSubjectType(subjectTypeStr)
	if err != nil {
		handleError(c, err)
		return
	}

	query := driving.ListBySubjectQuery{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		TenantID:    tenantID,
	}

	result, err := h.queryer.ListBySubject(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	assignments := make([]dto.AssignmentResponse, 0, len(result))
	for _, a := range result {
		assignments = append(assignments, h.toAssignmentResponse(a))
	}

	success(c, assignments)
}

// ListAssignmentsByRole 列出角色的分配记录
// @Summary 列出角色的分配记录
// @Tags Authorization-Assignments
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} dto.Response{data=[]dto.AssignmentResponse}
// @Router /authz/roles/{id}/assignments [get]
func (h *AssignmentHandler) ListAssignmentsByRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "角色ID格式错误"))
		return
	}

	tenantID := getTenantID(c)

	query := driving.ListByRoleQuery{
		RoleID:   roleID,
		TenantID: tenantID,
	}

	result, err := h.queryer.ListByRole(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	assignments := make([]dto.AssignmentResponse, 0, len(result))
	for _, a := range result {
		assignments = append(assignments, h.toAssignmentResponse(a))
	}

	success(c, assignments)
}

// toAssignmentResponse 转换为响应对象
func (h *AssignmentHandler) toAssignmentResponse(a *domainAssignment.Assignment) dto.AssignmentResponse {
	return dto.AssignmentResponse{
		ID:          a.ID.Uint64(),
		SubjectType: a.SubjectType.String(),
		SubjectID:   a.SubjectID,
		RoleID:      a.RoleID,
		TenantID:    a.TenantID,
		GrantedBy:   a.GrantedBy,
	}
}
