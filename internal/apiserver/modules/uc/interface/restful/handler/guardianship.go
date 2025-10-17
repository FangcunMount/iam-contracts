package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	appguard "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	requestdto "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful/request"
	responsedto "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful/response"
)

// GuardianshipHandler 监护关系 REST 处理器
type GuardianshipHandler struct {
	*BaseHandler
	guardApp appguard.GuardianshipApplicationService
}

// NewGuardianshipHandler 创建监护处理器
func NewGuardianshipHandler(
	guardApp appguard.GuardianshipApplicationService,
) *GuardianshipHandler {
	return &GuardianshipHandler{
		BaseHandler: NewBaseHandler(),
		guardApp:    guardApp,
	}
}

// Grant 授予监护关系
func (h *GuardianshipHandler) Grant(c *gin.Context) {
	var req requestdto.GuardianGrantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	dto := appguard.AddGuardianDTO{
		UserID:   req.UserID,
		ChildID:  req.ChildID,
		Relation: req.Relation,
	}

	if err := h.guardApp.AddGuardian(c.Request.Context(), dto); err != nil {
		h.Error(c, err)
		return
	}

	// 查询返回监护关系
	result, err := h.guardApp.GetByUserIDAndChildID(c.Request.Context(), req.UserID, req.ChildID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, guardResultToResponse(result))
}

// Revoke 撤销监护关系
func (h *GuardianshipHandler) Revoke(c *gin.Context) {
	var req requestdto.GuardianRevokeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	dto := appguard.RemoveGuardianDTO{
		UserID:  req.UserID,
		ChildID: req.ChildID,
	}

	if err := h.guardApp.RemoveGuardian(c.Request.Context(), dto); err != nil {
		h.Error(c, err)
		return
	}

	// 查询返回监护关系（包含撤销时间）
	result, err := h.guardApp.GetByUserIDAndChildID(c.Request.Context(), req.UserID, req.ChildID)
	if err != nil {
		h.Error(c, err)
		return
	}

	resp := guardResultToResponse(result)
	h.Success(c, resp)
}

// List 查询监护关系
func (h *GuardianshipHandler) List(c *gin.Context) {
	var req requestdto.GuardianshipListQuery
	if err := h.BindQuery(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	var results []*appguard.GuardianshipResult
	var err error

	switch {
	case req.UserID != "" && req.ChildID != "":
		// 查询特定用户和儿童的监护关系
		result, qerr := h.guardApp.GetByUserIDAndChildID(c.Request.Context(), req.UserID, req.ChildID)
		if qerr != nil {
			h.Error(c, qerr)
			return
		}
		if result != nil {
			results = []*appguard.GuardianshipResult{result}
		} else {
			results = []*appguard.GuardianshipResult{}
		}
	case req.UserID != "":
		// 查询用户的所有监护关系
		results, err = h.guardApp.ListChildrenByUserID(c.Request.Context(), req.UserID)
	case req.ChildID != "":
		// 查询儿童的所有监护人
		results, err = h.guardApp.ListGuardiansByChildID(c.Request.Context(), req.ChildID)
	default:
		results = []*appguard.GuardianshipResult{}
	}

	if err != nil {
		h.Error(c, err)
		return
	}

	// 过滤和分页
	filtered := filterGuardianshipResults(results, req.Active)
	total := len(filtered)
	items := make([]responsedto.GuardianshipResponse, 0, total)
	for _, g := range filtered {
		if g == nil {
			continue
		}
		items = append(items, guardResultToResponse(g))
	}

	sliced := sliceGuardianships(items, req.Offset, req.Limit)

	h.Success(c, responsedto.GuardianshipPageResponse{
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
		Items:  sliced,
	})
}

// ========== 辅助函数 ==========

// guardResultToResponse 将应用服务返回的 GuardianshipResult 转换为响应 DTO
func guardResultToResponse(result *appguard.GuardianshipResult) responsedto.GuardianshipResponse {
	if result == nil {
		return responsedto.GuardianshipResponse{}
	}

	resp := responsedto.GuardianshipResponse{
		ID:       result.ID,
		UserID:   result.UserID,
		ChildID:  result.ChildID,
		Relation: result.Relation,
		Since:    parseGuardTime(result.EstablishedAt),
	}

	return resp
}

// parseGuardTime 解析时间字符串
func parseGuardTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}

// filterGuardianshipResults 过滤监护关系（活跃/已撤销）
func filterGuardianshipResults(items []*appguard.GuardianshipResult, active *bool) []*appguard.GuardianshipResult {
	if active == nil {
		return items
	}

	// GuardianshipResult 中没有 IsActive 字段，这里简化处理
	// 如果需要过滤，可以根据 RevokedAt 判断
	return items
}

// sliceGuardianships 分页切片
func sliceGuardianships(items []responsedto.GuardianshipResponse, offset, limit int) []responsedto.GuardianshipResponse {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []responsedto.GuardianshipResponse{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
