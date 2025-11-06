package user

import (
	"context"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// profileEditor 用户资料编辑应用服务
type profileEditor struct {
	repo Repository
}

// 确保 profileEditor 实现了 ProfileEditor 接口
var _ ProfileEditor = (*profileEditor)(nil)

// NewProfileService 创建用户资料服务
func NewProfileService(repo Repository) *profileEditor {
	return &profileEditor{repo: repo}
}

// Rename 更新用户昵称
// 领域逻辑：验证 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *profileEditor) Rename(ctx context.Context, userID UserID, name string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, perrors.WithCode(code.ErrUserBasicInfoInvalid, "nickname cannot be empty")
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：修改实体
	user.Name = name

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

// UpdateContact 更新用户联系方式
// 领域逻辑：验证 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *profileEditor) UpdateContact(ctx context.Context, userID UserID, phone meta.Phone, email meta.Email) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域规则：验证手机号唯一性
	if !phone.IsEmpty() && !user.Phone.Equal(phone) {
		if err := ensurePhoneUnique(ctx, s.repo, phone); err != nil {
			return nil, err
		}
		user.UpdatePhone(phone)
	}

	if !email.IsEmpty() {
		user.UpdateEmail(email)
	}

	// 返回修改后的实体，由应用层持久化
	return user, nil
}

// UpdateIDCard 更新身份证信息
// 领域逻辑：验证 + 修改实体
// 注意：不包括持久化，返回修改后的实体供应用层持久化
func (s *profileEditor) UpdateIDCard(ctx context.Context, userID UserID, idCard meta.IDCard) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 领域逻辑：修改实体
	user.UpdateIDCard(idCard)

	// 返回修改后的实体，由应用层持久化
	return user, nil
}
