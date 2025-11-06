package service

import (
	"context"
	"errors"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// ensurePhoneUnique 确保手机号在系统中唯一
func ensurePhoneUnique(ctx context.Context, repo port.UserRepository, phone meta.Phone) error {
	if phone.IsEmpty() {
		return perrors.WithCode(code.ErrUserBasicInfoInvalid, "phone cannot be empty")
	}

	_, err := repo.FindByPhone(ctx, phone)
	if err == nil {
		return perrors.WithCode(code.ErrUserAlreadyExists, "user with phone(%s) already exists", phone.String())
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return perrors.WrapC(err, code.ErrDatabase, "check user phone(%s) failed", phone.String())
}
