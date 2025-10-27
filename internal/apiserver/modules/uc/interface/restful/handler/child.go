package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appchild "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/child"
	appguard "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	requestdto "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful/request"
	responsedto "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// ChildHandler 儿童档案 REST 处理器
type ChildHandler struct {
	*BaseHandler
	childApp   appchild.ChildApplicationService
	profileApp appchild.ChildProfileApplicationService
	guardApp   appguard.GuardianshipApplicationService
	guardQuery appguard.GuardianshipQueryApplicationService
	childQuery appchild.ChildQueryApplicationService
}

// NewChildHandler 创建儿童档案处理器
func NewChildHandler(
	childApp appchild.ChildApplicationService,
	profileApp appchild.ChildProfileApplicationService,
	guardApp appguard.GuardianshipApplicationService,
	guardQuery appguard.GuardianshipQueryApplicationService,
	childQuery appchild.ChildQueryApplicationService,
) *ChildHandler {
	return &ChildHandler{
		BaseHandler: NewBaseHandler(),
		childApp:    childApp,
		profileApp:  profileApp,
		guardApp:    guardApp,
		guardQuery:  guardQuery,
		childQuery:  childQuery,
	}
}

// ListMyChildren 获取当前用户的儿童档案列表
func (h *ChildHandler) ListMyChildren(c *gin.Context) {
	var query requestdto.ChildListQuery
	if err := h.BindQuery(c, &query); err != nil {
		h.Error(c, err)
		return
	}

	rawID, ok := h.GetUserID(c)
	if !ok {
		h.ErrorWithCode(c, code.ErrTokenInvalid, "user id not found in context")
		return
	}

	// 列出用户监护的所有儿童
	guardianships, err := h.guardQuery.ListChildrenByUserID(c.Request.Context(), rawID)
	if err != nil {
		h.Error(c, err)
		return
	}

	var children []responsedto.ChildResponse
	for _, g := range guardianships {
		if g == nil {
			continue
		}
		// 查询儿童详细信息
		child, err := h.childQuery.GetByID(c.Request.Context(), g.ChildID)
		if err != nil {
			h.Error(c, err)
			return
		}
		resp := childResultToResponse(child)
		children = append(children, resp)
	}

	total := len(children)
	sliced := sliceChildren(children, query.Offset, query.Limit)

	h.Success(c, responsedto.ChildPageResponse{
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
		Items:  sliced,
	})
}

// RegisterChild 注册儿童并授予当前用户监护权
func (h *ChildHandler) RegisterChild(c *gin.Context) {
	var req requestdto.ChildRegisterRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	rawUserID, ok := h.GetUserID(c)
	if !ok {
		h.ErrorWithCode(c, code.ErrTokenInvalid, "user id not found in context")
		return
	}

	// 构建注册 DTO
	registerDTO := appguard.RegisterChildWithGuardianDTO{
		ChildName:     strings.TrimSpace(req.LegalName),
		ChildGender:   genderIntToString(req.Gender),
		ChildBirthday: strings.TrimSpace(req.DOB),
		ChildIDCard:   strings.TrimSpace(req.IDNo),
		ChildHeight:   parseHeightCm(req.HeightCm),
		ChildWeight:   parseWeightKg(req.WeightKg),
		UserID:        rawUserID,
		Relation:      req.Relation,
	}

	// 调用应用服务注册儿童并建立监护关系
	result, err := h.guardApp.RegisterChildWithGuardian(c.Request.Context(), registerDTO)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 构建响应
	childResp := responsedto.ChildResponse{
		ID:        result.ChildID,
		LegalName: result.ChildName,
		Gender:    stringGenderToInt(result.ChildGender),
		DOB:       result.ChildBirthday,
		IDType:    req.IDType,
		IDMasked:  maskIDCard(result.ChildID),
	}

	guardResp := responsedto.GuardianshipResponse{
		ID:       result.ID,
		UserID:   result.UserID,
		ChildID:  result.ChildID,
		Relation: result.Relation,
		Since:    parseTime(result.EstablishedAt),
	}

	h.Created(c, responsedto.ChildRegisterResponse{
		Child:        childResp,
		Guardianship: guardResp,
	})
}

// CreateChild 仅创建儿童档案（不建立监护关系）
func (h *ChildHandler) CreateChild(c *gin.Context) {
	var req requestdto.ChildCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 构建注册 DTO
	dto := appchild.RegisterChildDTO{
		Name:     strings.TrimSpace(req.LegalName),
		Gender:   genderIntToString(req.Gender),
		Birthday: strings.TrimSpace(req.DOB),
		IDCard:   strings.TrimSpace(req.IDNo),
		Height:   parseHeightCm(req.HeightCm),
		Weight:   parseWeightKg(req.WeightKg),
	}

	result, err := h.childApp.Register(c.Request.Context(), dto)
	if err != nil {
		h.Error(c, err)
		return
	}

	resp := childResultToResponse(result)
	resp.IDType = req.IDType

	h.Created(c, resp)
}

// GetChild 查询儿童档案
func (h *ChildHandler) GetChild(c *gin.Context) {
	childID := c.Param("id")
	if strings.TrimSpace(childID) == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "child id is required")
		return
	}

	child, err := h.childQuery.GetByID(c.Request.Context(), childID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, childResultToResponse(child))
}

// PatchChild 更新儿童档案
func (h *ChildHandler) PatchChild(c *gin.Context) {
	childID := c.Param("id")
	if strings.TrimSpace(childID) == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "child id is required")
		return
	}

	var req requestdto.ChildUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	ctx := c.Request.Context()

	// 更新姓名
	if req.LegalName != nil && strings.TrimSpace(*req.LegalName) != "" {
		if err := h.profileApp.Rename(ctx, childID, strings.TrimSpace(*req.LegalName)); err != nil {
			h.Error(c, err)
			return
		}
	}

	// 更新性别和生日
	if req.Gender != nil || req.DOB != nil {
		dto := appchild.UpdateChildProfileDTO{
			ChildID: childID,
		}
		if req.Gender != nil {
			dto.Gender = genderIntToString(req.Gender)
		}
		if req.DOB != nil {
			dto.Birthday = strings.TrimSpace(*req.DOB)
		}
		if err := h.profileApp.UpdateProfile(ctx, dto); err != nil {
			h.Error(c, err)
			return
		}
	}

	// 更新身高体重
	if req.HeightCm != nil || req.WeightKg != nil {
		height := uint32(0)
		weight := uint32(0)

		if req.HeightCm != nil {
			height = uint32(*req.HeightCm)
		}
		if req.WeightKg != nil {
			f, _ := strconv.ParseFloat(strings.TrimSpace(*req.WeightKg), 64)
			weight = uint32(f * 1000) // kg转克
		}

		dto := appchild.UpdateHeightWeightDTO{
			ChildID: childID,
			Height:  height,
			Weight:  weight,
		}
		if err := h.profileApp.UpdateHeightWeight(ctx, dto); err != nil {
			h.Error(c, err)
			return
		}
	}

	// 返回更新后的儿童信息
	child, err := h.childQuery.GetByID(ctx, childID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, childResultToResponse(child))
}

// SearchChildren 搜索相似儿童（根据姓名、性别、生日）
func (h *ChildHandler) SearchChildren(c *gin.Context) {
	var query requestdto.ChildSearchQuery
	if err := h.BindQuery(c, &query); err != nil {
		h.Error(c, err)
		return
	}

	name := strings.TrimSpace(query.Name)
	birthday := ""
	if query.DOB != nil {
		birthday = strings.TrimSpace(*query.DOB)
	}

	// SearchQuery 中没有 Gender，这里使用空字符串
	children, err := h.childQuery.FindSimilar(c.Request.Context(), name, "", birthday)
	if err != nil {
		h.Error(c, err)
		return
	}

	var items []responsedto.ChildResponse
	for _, child := range children {
		if child != nil {
			items = append(items, childResultToResponse(child))
		}
	}

	total := len(items)
	sliced := sliceChildren(items, query.Offset, query.Limit)

	h.Success(c, responsedto.ChildPageResponse{
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
		Items:  sliced,
	})
}

// ========== 辅助函数 ==========

// childResultToResponse 将应用服务返回的 ChildResult 转换为响应 DTO
func childResultToResponse(result *appchild.ChildResult) responsedto.ChildResponse {
	if result == nil {
		return responsedto.ChildResponse{}
	}

	resp := responsedto.ChildResponse{
		ID:        result.ID,
		LegalName: result.Name,
		DOB:       result.Birthday,
		IDMasked:  maskIDCard(result.IDCard),
	}

	// 性别转换
	if result.Gender != "" {
		gender := stringGenderToInt(result.Gender)
		resp.Gender = gender
	}

	// 身高（厘米）
	if result.Height > 0 {
		h := int(result.Height)
		resp.HeightCm = &h
	}

	// 体重（千克，字符串格式）
	if result.Weight > 0 {
		kg := float64(result.Weight) / 1000.0
		w := fmt.Sprintf("%.2f", kg)
		resp.WeightKg = &w
	}

	return resp
}

// genderIntToString 将前端的整数性别转换为字符串
func genderIntToString(gender *int) string {
	if gender == nil {
		return ""
	}
	switch *gender {
	case 1:
		return "male"
	case 2:
		return "female"
	default:
		return ""
	}
}

// stringGenderToInt 将字符串性别转换为整数指针
func stringGenderToInt(gender string) *int {
	var g int
	switch strings.ToLower(gender) {
	case "male":
		g = 1
	case "female":
		g = 2
	default:
		return nil
	}
	return &g
}

// parseHeightCm 解析身高（厘米）
func parseHeightCm(heightCm *int) *uint32 {
	if heightCm == nil || *heightCm <= 0 {
		return nil
	}
	h := uint32(*heightCm)
	return &h
}

// parseWeightKg 解析体重（千克字符串转克）
func parseWeightKg(weightKg string) *uint32 {
	if strings.TrimSpace(weightKg) == "" {
		return nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(weightKg), 64)
	if err != nil || f <= 0 {
		return nil
	}
	w := uint32(f * 1000) // 千克转克
	return &w
}

// maskIDCard 脱敏身份证号
func maskIDCard(idCard string) string {
	if len(idCard) < 6 {
		return idCard
	}
	return idCard[:6] + "********" + idCard[len(idCard)-4:]
}

// sliceChildren 分页切片
func sliceChildren(items []responsedto.ChildResponse, offset, limit int) []responsedto.ChildResponse {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []responsedto.ChildResponse{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

// parseTime 解析时间字符串（ISO 8601 格式）
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}
