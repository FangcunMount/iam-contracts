// Package handler 资源管理处理器
package handler

import (
	"strconv"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/resource"
	domainResource "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/interface/restful/dto"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/gin-gonic/gin"
)

// ResourceHandler 资源处理器
type ResourceHandler struct {
	resourceService *resource.Service
}

// NewResourceHandler 创建资源处理器
func NewResourceHandler(resourceService *resource.Service) *ResourceHandler {
	return &ResourceHandler{
		resourceService: resourceService,
	}
}

// CreateResource 创建资源
// @Summary 创建资源
// @Tags Resource
// @Accept json
// @Produce json
// @Param request body dto.CreateResourceRequest true "创建资源请求"
// @Success 200 {object} dto.Response{data=dto.ResourceResponse}
// @Router /authz/resources [post]
func (h *ResourceHandler) CreateResource(c *gin.Context) {
	var req dto.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	cmd := resource.CreateResourceCommand{
		Key:         req.Key,
		DisplayName: req.DisplayName,
		AppName:     req.AppName,
		Domain:      req.Domain,
		Type:        req.Type,
		Actions:     req.Actions,
		Description: req.Description,
	}

	createdResource, err := h.resourceService.CreateResource(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toResourceResponse(createdResource))
}

// UpdateResource 更新资源
// @Summary 更新资源
// @Tags Resource
// @Accept json
// @Produce json
// @Param id path string true "资源ID"
// @Param request body dto.UpdateResourceRequest true "更新资源请求"
// @Success 200 {object} dto.Response{data=dto.ResourceResponse}
// @Router /authz/resources/{id} [put]
func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "资源ID格式错误"))
		return
	}

	var req dto.UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	cmd := resource.UpdateResourceCommand{
		ID:          domainResource.NewResourceID(resourceID),
		DisplayName: req.DisplayName,
		Actions:     req.Actions,
		Description: req.Description,
	}

	updatedResource, err := h.resourceService.UpdateResource(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toResourceResponse(updatedResource))
}

// DeleteResource 删除资源
// @Summary 删除资源
// @Tags Resource
// @Param id path string true "资源ID"
// @Success 200 {object} dto.Response
// @Router /authz/resources/{id} [delete]
func (h *ResourceHandler) DeleteResource(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "资源ID格式错误"))
		return
	}

	err = h.resourceService.DeleteResource(c.Request.Context(), domainResource.NewResourceID(resourceID))
	if err != nil {
		handleError(c, err)
		return
	}

	successNoContent(c)
}

// GetResource 获取资源详情
// @Summary 获取资源详情
// @Tags Resource
// @Produce json
// @Param id path string true "资源ID"
// @Success 200 {object} dto.Response{data=dto.ResourceResponse}
// @Router /authz/resources/{id} [get]
func (h *ResourceHandler) GetResource(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handleError(c, errors.WithCode(code.ErrInvalidArgument, "资源ID格式错误"))
		return
	}

	foundResource, err := h.resourceService.GetResourceByID(c.Request.Context(), domainResource.NewResourceID(resourceID))
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toResourceResponse(foundResource))
}

// GetResourceByKey 根据键获取资源
// @Summary 根据键获取资源
// @Tags Resource
// @Produce json
// @Param key path string true "资源键"
// @Success 200 {object} dto.Response{data=dto.ResourceResponse}
// @Router /authz/resources/key/{key} [get]
func (h *ResourceHandler) GetResourceByKey(c *gin.Context) {
	key := c.Param("key")

	foundResource, err := h.resourceService.GetResourceByKey(c.Request.Context(), key)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, h.toResourceResponse(foundResource))
}

// ListResources 列出资源
// @Summary 列出资源
// @Tags Resource
// @Produce json
// @Param app_name query string false "应用名称"
// @Param domain query string false "域"
// @Param type query string false "类型"
// @Param offset query int false "偏移量" default(0)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} dto.ListResponse{data=[]dto.ResourceResponse}
// @Router /authz/resources [get]
func (h *ResourceHandler) ListResources(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	query := resource.ListResourceQuery{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.resourceService.ListResources(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	resources := make([]dto.ResourceResponse, 0, len(result.Resources))
	for _, r := range result.Resources {
		resources = append(resources, h.toResourceResponse(r))
	}

	successList(c, resources, result.Total, offset, limit)
}

// ValidateAction 验证资源动作
// @Summary 验证资源动作
// @Tags Resource
// @Accept json
// @Produce json
// @Param request body dto.ValidateActionRequest true "验证动作请求"
// @Success 200 {object} dto.Response{data=dto.ValidateActionResponse}
// @Router /authz/resources/validate-action [post]
func (h *ResourceHandler) ValidateAction(c *gin.Context) {
	var req dto.ValidateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, errors.WithCode(code.ErrBind, "请求参数错误: %v", err))
		return
	}

	query := resource.ValidateActionQuery{
		ResourceKey: req.ResourceKey,
		Action:      req.Action,
	}

	valid, err := h.resourceService.ValidateAction(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	success(c, dto.ValidateActionResponse{
		Valid: valid,
	})
}

// toResourceResponse 转换为响应对象
func (h *ResourceHandler) toResourceResponse(r *domainResource.Resource) dto.ResourceResponse {
	return dto.ResourceResponse{
		ID:          r.ID.Uint64(),
		Key:         r.Key,
		DisplayName: r.DisplayName,
		AppName:     r.AppName,
		Domain:      r.Domain,
		Type:        r.Type,
		Actions:     r.Actions,
		Description: r.Description,
	}
}
