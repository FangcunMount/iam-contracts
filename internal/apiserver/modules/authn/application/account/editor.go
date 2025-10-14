package account

import (
	"context"
	"errors"
	"strings"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// EditorService 负责账号相关的编辑能力。
type EditorService struct {
	wechat    port.WeChatRepo
	operation port.OperationRepo
}

var _ port.AccountEditor = (*EditorService)(nil)

// NewEditorService 构造编辑服务。
func NewEditorService(wx port.WeChatRepo, op port.OperationRepo) *EditorService {
	return &EditorService{
		wechat:    wx,
		operation: op,
	}
}

// UpdateWeChatProfile 更新微信昵称与头像。
func (s *EditorService) UpdateWeChatProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string) error {
	nick, nickSet := normalizeOptionalString(nickname)
	ava, avaSet := normalizeOptionalString(avatar)
	if !nickSet && !avaSet {
		return perrors.WithCode(code.ErrInvalidArgument, "no profile fields provided")
	}

	if _, err := s.wechat.FindByAccountID(ctx, accountID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "wechat credential for account(%s) not found", accountIDString(accountID))
		}
		return perrors.WrapC(err, code.ErrDatabase, "load wechat credential for account(%s) failed", accountIDString(accountID))
	}

	type profileUpdater interface {
		UpdateProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string) error
	}

	if updater, ok := s.wechat.(profileUpdater); ok {
		if err := updater.UpdateProfile(ctx, accountID, nick, ava); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "update wechat profile for account(%s) failed", accountIDString(accountID))
		}
		return nil
	}

	return perrors.WithCode(code.ErrInternalServerError, "wechat repository does not support profile update")
}

// UpdateOperationCredential 更新运营后台凭证信息。
func (s *EditorService) UpdateOperationCredential(ctx context.Context, username string, newHash []byte, algo string, params []byte) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}
	if len(newHash) == 0 {
		return perrors.WithCode(code.ErrInvalidArgument, "password hash cannot be empty")
	}
	algo = strings.TrimSpace(algo)
	if algo == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "hash algorithm cannot be empty")
	}

	if _, err := s.operation.FindByUsername(ctx, username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) not found", username)
		}
		return perrors.WrapC(err, code.ErrDatabase, "load operation credential(%s) failed", username)
	}

	if err := s.operation.UpdateHash(ctx, username, newHash, algo, params); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update operation credential(%s) failed", username)
	}

	return nil
}

// ChangeOperationUsername 修改运营后台账号用户名。
func (s *EditorService) ChangeOperationUsername(ctx context.Context, oldUsername, newUsername string) error {
	oldUsername = strings.TrimSpace(oldUsername)
	newUsername = strings.TrimSpace(newUsername)
	if oldUsername == "" || newUsername == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}
	if oldUsername == newUsername {
		return nil
	}

	cred, err := s.operation.FindByUsername(ctx, oldUsername)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) not found", oldUsername)
		}
		return perrors.WrapC(err, code.ErrDatabase, "load operation credential(%s) failed", oldUsername)
	}

	if _, err := s.operation.FindByUsername(ctx, newUsername); err == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) already exists", newUsername)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return perrors.WrapC(err, code.ErrDatabase, "check operation credential(%s) failed", newUsername)
	}

	if err := s.operation.UpdateUsername(ctx, cred.AccountID, newUsername); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "change operation username from %s to %s failed", oldUsername, newUsername)
	}

	return nil
}

func normalizeOptionalString(input *string) (*string, bool) {
	if input == nil {
		return nil, false
	}
	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil, true
	}
	result := trimmed
	return &result, true
}

// ensure imported time used? currently only used in register, not here, but we imported time? yes at top time used? only in UpdateOperationCredential? not; we have time import but not used. We should remove time from imports? It's there but not used -> compile error. but in this file we only need time? we do not. remove time.
