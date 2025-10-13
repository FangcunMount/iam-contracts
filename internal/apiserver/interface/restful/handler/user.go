package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	requestdto "github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/restful/request"
	responsedto "github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/restful/response"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	"github.com/fangcun-mount/iam-contracts/pkg/core"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserHandler 基础用户 REST 处理器
type UserHandler struct {
	registerSrv port.UserRegister
	statusSrv   port.UserStatusChanger
	profileSrv  port.UserProfileEditor
	querySrv    port.UserQueryer
}

// NewUserHandler 创建用户处理器
func NewUserHandler(
	registerSrv port.UserRegister,
	statusSrv port.UserStatusChanger,
	profileSrv port.UserProfileEditor,
	querySrv port.UserQueryer,
) *UserHandler {
	return &UserHandler{
		registerSrv: registerSrv,
		statusSrv:   statusSrv,
		profileSrv:  profileSrv,
		querySrv:    querySrv,
	}
}

// Register 注册用户
func (h *UserHandler) Register(c *gin.Context) {
	var req requestdto.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrBind, "invalid request payload: %v", err), nil)
		return
	}

	ctx := c.Request.Context()

	created, err := h.registerSrv.Register(ctx, req.Name, meta.NewPhone(req.Phone))
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	if req.Email != "" {
		var emptyPhone meta.Phone
		if err := h.profileSrv.UpdateContact(ctx, created.ID, emptyPhone, meta.NewEmail(req.Email)); err != nil {
			core.WriteResponse(c, err, nil)
			return
		}
	}

	if req.IDCardNumber != "" {
		name := req.IDCardName
		if name == "" {
			name = req.Name
		}
		if err := h.profileSrv.UpdateIDCard(ctx, created.ID, meta.NewIDCard(name, req.IDCardNumber)); err != nil {
			core.WriteResponse(c, err, nil)
			return
		}
	}

	user, err := h.querySrv.FindByID(ctx, created.ID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(user),
	})
}

// GetUser 根据 ID 获取用户
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := parseUserID(c.Param("id"))
	if err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %v", err), nil)
		return
	}

	u, err := h.querySrv.FindByID(c.Request.Context(), userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(u),
	})
}

// GetUserByPhone 根据手机号获取用户
func (h *UserHandler) GetUserByPhone(c *gin.Context) {
	phone := strings.TrimSpace(c.Query("phone"))
	if phone == "" {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "phone query parameter is required"), nil)
		return
	}

	u, err := h.querySrv.FindByPhone(c.Request.Context(), meta.NewPhone(phone))
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(u),
	})
}

// UpdateContact 更新用户联系方式
func (h *UserHandler) UpdateContact(c *gin.Context) {
	userID, err := parseUserID(c.Param("id"))
	if err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %v", err), nil)
		return
	}

	var req requestdto.UpdateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrBind, "invalid request payload: %v", err), nil)
		return
	}

	if req.IsEmpty() {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "either phone or email must be provided"), nil)
		return
	}

	var phone meta.Phone
	if strings.TrimSpace(req.Phone) != "" {
		phone = meta.NewPhone(req.Phone)
	}

	var email meta.Email
	if strings.TrimSpace(req.Email) != "" {
		email = meta.NewEmail(req.Email)
	}

	if err := h.profileSrv.UpdateContact(c.Request.Context(), userID, phone, email); err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	u, err := h.querySrv.FindByID(c.Request.Context(), userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(u),
	})
}

// UpdateIDCard 更新用户身份证信息
func (h *UserHandler) UpdateIDCard(c *gin.Context) {
	userID, err := parseUserID(c.Param("id"))
	if err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %v", err), nil)
		return
	}

	var req requestdto.UpdateIDCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrBind, "invalid request payload: %v", err), nil)
		return
	}

	ctx := c.Request.Context()

	current, err := h.querySrv.FindByID(ctx, userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	name := req.Name
	if name == "" {
		name = current.Name
	}

	if err := h.profileSrv.UpdateIDCard(ctx, userID, meta.NewIDCard(name, req.Number)); err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	u, err := h.querySrv.FindByID(ctx, userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(u),
	})
}

// ChangeStatus 修改用户状态
func (h *UserHandler) ChangeStatus(c *gin.Context) {
	userID, err := parseUserID(c.Param("id"))
	if err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %v", err), nil)
		return
	}

	var req requestdto.ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrBind, "invalid request payload: %v", err), nil)
		return
	}

	ctx := c.Request.Context()

	switch strings.ToLower(strings.TrimSpace(req.Status)) {
	case "active":
		err = h.statusSrv.Activate(ctx, userID)
	case "inactive":
		err = h.statusSrv.Deactivate(ctx, userID)
	case "blocked", "block":
		err = h.statusSrv.Block(ctx, userID)
	default:
		core.WriteResponse(c, perrors.WithCode(code.ErrUserStatusInvalid, "unsupported status: %s", req.Status), nil)
		return
	}

	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	u, err := h.querySrv.FindByID(ctx, userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code":    code.ErrSuccess,
		"message": "user status updated",
		"data":    newUserResponse(u),
	})
}

// GetUserProfile 获取当前用户资料
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	rawUserID, exists := c.Get("user_id")
	if !exists {
		core.WriteResponse(c, perrors.WithCode(code.ErrTokenInvalid, "user id not found in context"), nil)
		return
	}

	userID, err := toUserID(rawUserID)
	if err != nil {
		core.WriteResponse(c, perrors.WithCode(code.ErrInvalidArgument, "invalid user id in context: %v", err), nil)
		return
	}

	u, err := h.querySrv.FindByID(c.Request.Context(), userID)
	if err != nil {
		core.WriteResponse(c, err, nil)
		return
	}

	core.WriteResponse(c, nil, gin.H{
		"code": code.ErrSuccess,
		"data": newUserResponse(u),
	})
}

func newUserResponse(u *domain.User) responsedto.UserResponse {
	if u == nil {
		return responsedto.UserResponse{}
	}
	return responsedto.UserResponse{
		ID:       u.ID.Value(),
		Name:     u.Name,
		Phone:    u.Phone.String(),
		Email:    u.Email.String(),
		IDCard:   u.IDCard.String(),
		Status:   u.Status.String(),
		StatusID: u.Status.Value(),
	}
}

func parseUserID(raw string) (domain.UserID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.UserID{}, fmt.Errorf("empty user id")
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return domain.UserID{}, err
	}

	return domain.NewUserID(id), nil
}

func toUserID(value interface{}) (domain.UserID, error) {
	switch v := value.(type) {
	case domain.UserID:
		return v, nil
	case uint64:
		return domain.NewUserID(v), nil
	case uint32:
		return domain.NewUserID(uint64(v)), nil
	case uint:
		return domain.NewUserID(uint64(v)), nil
	case int64:
		if v < 0 {
			return domain.UserID{}, fmt.Errorf("negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case int32:
		if v < 0 {
			return domain.UserID{}, fmt.Errorf("negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case int:
		if v < 0 {
			return domain.UserID{}, fmt.Errorf("negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case float64:
		if v < 0 {
			return domain.UserID{}, fmt.Errorf("negative id: %f", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case string:
		return parseUserID(v)
	default:
		return domain.UserID{}, fmt.Errorf("unsupported user id type: %T", value)
	}
}
