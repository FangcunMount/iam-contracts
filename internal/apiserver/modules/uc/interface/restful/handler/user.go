package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	appuser "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/user"
	requestdto "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful/request"
	responsedto "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	_ "github.com/FangcunMount/iam-contracts/pkg/core" // imported for swagger
)

// UserHandler 基础用户 REST 处理器
type UserHandler struct {
	*BaseHandler
	userApp    appuser.UserApplicationService
	profileApp appuser.UserProfileApplicationService
	userQuery  appuser.UserQueryApplicationService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(
	userApp appuser.UserApplicationService,
	profileApp appuser.UserProfileApplicationService,
	userQuery appuser.UserQueryApplicationService,
) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler(),
		userApp:     userApp,
		profileApp:  profileApp,
		userQuery:   userQuery,
	}
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户，至少需要提供昵称或联系方式
// @Tags Identity-User
// @Accept json
// @Produce json
// @Param request body requestdto.UserCreateRequest true "创建用户请求"
// @Success 201 {object} responsedto.UserResponse "创建成功"
// @Failure 400 {object} core.ErrResponse "请求参数错误"
// @Failure 500 {object} core.ErrResponse "服务器内部错误"
// @Router /users [post]
// @Security BearerAuth
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req requestdto.UserCreateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	name := strings.TrimSpace(req.Nickname)
	phoneValue, emailValue := extractContactValues(req.Contacts)

	if name == "" {
		if phoneValue != "" {
			name = phoneValue
		} else if emailValue != "" {
			name = emailValue
		}
	}
	if name == "" {
		h.ErrorWithCode(c, code.ErrUserBasicInfoInvalid, "nickname or contact must be provided")
		return
	}
	if phoneValue == "" {
		h.ErrorWithCode(c, code.ErrUserBasicInfoInvalid, "phone contact is required")
		return
	}

	ctx := c.Request.Context()

	result, err := h.userApp.Register(ctx, appuser.RegisterUserDTO{
		Name:  name,
		Phone: phoneValue,
		Email: emailValue,
	})
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Created(c, newUserResponse(result))
}

// GetUser 根据 ID 查询用户
// @Summary 查询用户
// @Description 根据用户 ID 查询用户详细信息
// @Tags Identity-User
// @Accept json
// @Produce json
// @Param userId path string true "用户 ID"
// @Success 200 {object} responsedto.UserResponse "查询成功"
// @Failure 400 {object} core.ErrResponse "参数错误"
// @Failure 404 {object} core.ErrResponse "用户不存在"
// @Failure 500 {object} core.ErrResponse "服务器内部错误"
// @Router /users/{userId} [get]
// @Security BearerAuth
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := parseUserID(c.Param("userId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	u, err := h.userQuery.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newUserResponse(u))
}

// PatchUser 更新用户信息（昵称 / 联系方式）
// @Summary 更新用户信息
// @Description 部分更新用户信息，支持更新昵称和联系方式
// @Tags Identity-User
// @Accept json
// @Produce json
// @Param userId path string true "用户 ID"
// @Param request body requestdto.UserUpdateRequest true "更新用户请求"
// @Success 200 {object} responsedto.UserResponse "更新成功"
// @Failure 400 {object} core.ErrResponse "参数错误"
// @Failure 404 {object} core.ErrResponse "用户不存在"
// @Failure 500 {object} core.ErrResponse "服务器内部错误"
// @Router /users/{userId} [patch]
// @Security BearerAuth
func (h *UserHandler) PatchUser(c *gin.Context) {
	userID, err := parseUserID(c.Param("userId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	var req requestdto.UserUpdateRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	ctx := c.Request.Context()

	if req.Nickname != nil {
		if err := h.profileApp.Rename(ctx, userID, strings.TrimSpace(*req.Nickname)); err != nil {
			h.Error(c, err)
			return
		}
	}

	if len(req.Contacts) > 0 {
		phoneValue, emailValue := extractContactValues(req.Contacts)

		if phoneValue != "" || emailValue != "" {
			if err := h.profileApp.UpdateContact(ctx, appuser.UpdateContactDTO{
				UserID: userID,
				Phone:  phoneValue,
				Email:  emailValue,
			}); err != nil {
				h.Error(c, err)
				return
			}
		}
	}

	u, err := h.userQuery.GetByID(ctx, userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newUserResponse(u))
}

// GetUserProfile 获取当前用户资料
// @Summary 获取当前用户资料
// @Description 获取当前登录用户的资料信息
// @Tags Identity-User
// @Accept json
// @Produce json
// @Success 200 {object} responsedto.UserResponse "查询成功"
// @Failure 401 {object} core.ErrResponse "未授权"
// @Failure 500 {object} core.ErrResponse "服务器内部错误"
// @Router /users/profile [get]
// @Security BearerAuth
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	rawUserID, exists := c.Get("user_id")
	if !exists {
		h.ErrorWithCode(c, code.ErrTokenInvalid, "user id not found in context")
		return
	}

	userID, err := toUserID(rawUserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	u, err := h.userQuery.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newUserResponse(u))
}

func extractContactValues(contacts []requestdto.UserContactUpsert) (phone string, email string) {
	for _, contact := range contacts {
		switch strings.ToLower(contact.Type) {
		case "phone":
			if phone == "" {
				phone = strings.TrimSpace(contact.Value)
			}
		case "email":
			if email == "" {
				email = strings.TrimSpace(contact.Value)
			}
		}
	}
	return
}

func newUserResponse(u *appuser.UserResult) responsedto.UserResponse {
	if u == nil {
		return responsedto.UserResponse{}
	}

	resp := responsedto.UserResponse{
		ID:       u.ID,
		Status:   u.Status.String(),
		Nickname: u.Name,
	}

	if strings.TrimSpace(u.Phone) != "" {
		resp.Contacts = append(resp.Contacts, responsedto.VerifiedContactResponse{
			Type:  "phone",
			Value: strings.TrimSpace(u.Phone),
		})
	}

	if strings.TrimSpace(u.Email) != "" {
		resp.Contacts = append(resp.Contacts, responsedto.VerifiedContactResponse{
			Type:  "email",
			Value: strings.TrimSpace(u.Email),
		})
	}

	return resp
}

func parseUserID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", perrors.WithCode(code.ErrInvalidArgument, "user id cannot be empty")
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return "", perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %s", raw)
	}

	return strconv.FormatUint(id, 10), nil
}

func toUserID(value interface{}) (string, error) {
	switch v := value.(type) {
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case int64:
		if v < 0 {
			return "", perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return strconv.FormatUint(uint64(v), 10), nil
	case int32:
		if v < 0 {
			return "", perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return strconv.FormatUint(uint64(v), 10), nil
	case int:
		if v < 0 {
			return "", perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return strconv.FormatUint(uint64(v), 10), nil
	case float64:
		if v < 0 {
			return "", perrors.WithCode(code.ErrInvalidArgument, "negative id: %f", v)
		}
		return strconv.FormatUint(uint64(v), 10), nil
	case string:
		return parseUserID(v)
	default:
		return "", perrors.WithCode(code.ErrInvalidArgument, "unsupported user id type: %T", value)
	}
}
