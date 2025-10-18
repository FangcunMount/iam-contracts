// Package handler 角色管理处理器
package handler

import (
	"strconv"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/role"
	domainRole "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/interface/restful/dto"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/gin-gonic/gin"
)

// RoleHandler 角色处理器
type RoleHandler struct {
	roleService *role.Service
}

// NewRoleHandler 创建角色处理器
func NewRoleHandler(roleService *role.Service) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

// CreateRole 创建角色
// @Summary 创建角色
// @Tags Role
// @Accept json
// @Produce json
// @Param request body dto.CreateRoleRequest true "创建角色请求"
// @Success 200 {object} dto.Response{data=dto.RoleResponse}
// @Router /authz/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)

	cmd := role.CreateRoleCommand{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		TenantID:    tenantID,
		Description: req.Description,
	}

	createdRole, err := h.roleService.CreateRole(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toRoleResponse(createdRole))
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Tags Role
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body dto.UpdateRoleRequest true "更新角色请求"
// @Success 200 {object} dto.Response{data=dto.RoleResponse}
// @Router /authz/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "角色ID格式错误"))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	cmd := role.UpdateRoleCommand{
		ID:          domainRole.NewRoleID(roleID),
		DisplayName: req.DisplayName,
		Description: req.Description,
	}

	updatedRole, err := h.roleService.UpdateRole(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toRoleResponse(updatedRole))
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Tags Role
// @Param id path string true "角色ID"
// @Success 200 {object} dto.Response
// @Router /authz/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "角色ID格式错误"))
		return
	}

	tenantID := getTenantID(c)

	err = h.roleService.DeleteRole(c.Request.Context(), domainRole.NewRoleID(roleID), tenantID)
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// GetRole 获取角色详情
// @Summary 获取角色详情
// @Tags Role
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} dto.Response{data=dto.RoleResponse}
// @Router /authz/roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "角色ID格式错误"))
		return
	}

	tenantID := getTenantID(c)

	foundRole, err := h.roleService.GetRoleByID(c.Request.Context(), domainRole.NewRoleID(roleID), tenantID)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toRoleResponse(foundRole))
}

// ListRoles 列出角色
// @Summary 列出角色
// @Tags Role
// @Produce json
// @Param offset query int false "偏移量" default(0)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} dto.ListResponse{data=[]dto.RoleResponse}
// @Router /authz/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	var query dto.ListRoleQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	tenantID := getTenantID(c)

	listQuery := role.ListRoleQuery{
		TenantID: tenantID,
		Offset:   query.Offset,
		Limit:    query.Limit,
	}

	result, err := h.roleService.ListRoles(c.Request.Context(), listQuery)
	if err != nil {
		handleError(c, err)
		return
	}

	roles := make([]dto.RoleResponse, 0, len(result.Roles))
	for _, r := range result.Roles {
		roles = append(roles, h.toRoleResponse(r))
	}

	successList(c, roles, result.Total, query.Offset, query.Limit)
}

// toRoleResponse 转换为响应对象
func (h *RoleHandler) toRoleResponse(r *domainRole.Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID:          r.ID.Uint64(),
		Name:        r.Name,
		DisplayName: r.DisplayName,
		TenantID:    r.TenantID,
		Description: r.Description,
	}
}
