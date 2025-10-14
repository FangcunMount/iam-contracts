package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	requestdto "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful/request"
	responsedto "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful/response"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserHandler 基础用户 REST 处理器
type UserHandler struct {
	*BaseHandler
	registerSrv port.UserRegister
	profileSrv  port.UserProfileEditor
	querySrv    port.UserQueryer
}

// NewUserHandler 创建用户处理器
func NewUserHandler(
	registerSrv port.UserRegister,
	profileSrv port.UserProfileEditor,
	querySrv port.UserQueryer,
) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler(),
		registerSrv: registerSrv,
		profileSrv:  profileSrv,
		querySrv:    querySrv,
	}
}

// CreateUser 创建用户
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

	created, err := h.registerSrv.Register(ctx, name, meta.NewPhone(phoneValue))
	if err != nil {
		h.Error(c, err)
		return
	}

	if emailValue != "" {
		var emptyPhone meta.Phone
		if err := h.profileSrv.UpdateContact(ctx, created.ID, emptyPhone, meta.NewEmail(emailValue)); err != nil {
			h.Error(c, err)
			return
		}
	}

	user, err := h.querySrv.FindByID(ctx, created.ID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Created(c, newUserResponse(user))
}

// GetUser 根据 ID 查询用户
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := parseUserID(c.Param("userId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	u, err := h.querySrv.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newUserResponse(u))
}

// PatchUser 更新用户信息（昵称 / 联系方式）
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
		if err := h.profileSrv.Rename(ctx, userID, strings.TrimSpace(*req.Nickname)); err != nil {
			h.Error(c, err)
			return
		}
	}

	if len(req.Contacts) > 0 {
		phoneValue, emailValue := extractContactValues(req.Contacts)

		var phone meta.Phone
		var email meta.Email
		updateContact := false

		if phoneValue != "" {
			phone = meta.NewPhone(phoneValue)
			updateContact = true
		}
		if emailValue != "" {
			email = meta.NewEmail(emailValue)
			updateContact = true
		}

		if updateContact {
			if err := h.profileSrv.UpdateContact(ctx, userID, phone, email); err != nil {
				h.Error(c, err)
				return
			}
		}
	}

	u, err := h.querySrv.FindByID(ctx, userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newUserResponse(u))
}

// GetUserProfile 获取当前用户资料
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

	u, err := h.querySrv.FindByID(c.Request.Context(), userID)
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

func newUserResponse(u *domain.User) responsedto.UserResponse {
	if u == nil {
		return responsedto.UserResponse{}
	}

	resp := responsedto.UserResponse{
		ID:       u.ID.String(),
		Status:   u.Status.String(),
		Nickname: u.Name,
	}

	if !u.Phone.IsEmpty() {
		resp.Contacts = append(resp.Contacts, responsedto.VerifiedContactResponse{
			Type:  "phone",
			Value: u.Phone.String(),
		})
	}

	if !u.Email.IsEmpty() {
		resp.Contacts = append(resp.Contacts, responsedto.VerifiedContactResponse{
			Type:  "email",
			Value: u.Email.String(),
		})
	}

	return resp
}

func parseUserID(raw string) (domain.UserID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "user id cannot be empty")
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "invalid user id: %s", raw)
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
			return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case int32:
		if v < 0 {
			return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case int:
		if v < 0 {
			return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "negative id: %d", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case float64:
		if v < 0 {
			return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "negative id: %f", v)
		}
		return domain.NewUserID(uint64(v)), nil
	case string:
		return parseUserID(v)
	default:
		return domain.UserID{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported user id type: %T", value)
	}
}
