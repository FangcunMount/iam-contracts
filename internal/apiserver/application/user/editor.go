package user

import (
	"context"
	"strings"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserProfileEditor 用户资料编辑应用服务
type UserProfileEditor struct {
	repo port.UserRepository
}

// 确保 UserProfileEditor 实现了 port.UserProfileEditor 接口
var _ port.UserProfileEditor = (*UserProfileEditor)(nil)

// NewProfileService 创建用户资料服务
func NewProfileService(repo port.UserRepository) *UserProfileEditor {
	return &UserProfileEditor{repo: repo}
}

// Rename 更新用户昵称
func (s *UserProfileEditor) Rename(ctx context.Context, userID domain.UserID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return perrors.WithCode(code.ErrUserBasicInfoInvalid, "nickname cannot be empty")
	}

	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.Name = name

	if err := s.repo.Update(ctx, u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "rename user(%s) failed", userID.String())
	}

	return nil
}

// UpdateContact 更新用户联系方式
func (s *UserProfileEditor) UpdateContact(ctx context.Context, userID domain.UserID, phone meta.Phone, email meta.Email) error {
	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if !phone.IsEmpty() && !u.Phone.Equal(phone) {
		if err := ensurePhoneUnique(ctx, s.repo, phone); err != nil {
			return err
		}
		u.UpdatePhone(phone)
	}

	if !email.IsEmpty() {
		u.UpdateEmail(email)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update contact for user(%s) failed", userID.String())
	}

	return nil
}

// UpdateIDCard 更新身份证信息
func (s *UserProfileEditor) UpdateIDCard(ctx context.Context, userID domain.UserID, idCard meta.IDCard) error {
	u, err := NewQueryService(s.repo).FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.UpdateIDCard(idCard)

	if err := s.repo.Update(ctx, u); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update id card for user(%s) failed", userID.String())
	}

	return nil
}
