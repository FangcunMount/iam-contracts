package service

import (
	"context"
	"errors"
	"strings"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	drivenPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	drivingPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driving"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// EditorService 领域编辑服务
// 职责：封装账号编辑的业务规则
// 不包含：事务管理（由应用层负责）
type EditorService struct {
	wechat    drivenPort.WeChatRepo
	operation drivenPort.OperationRepo
}

var _ drivingPort.AccountEditor = (*EditorService)(nil)

// NewEditorService 构造编辑服务
func NewEditorService(wx drivenPort.WeChatRepo, op drivenPort.OperationRepo) *EditorService {
	return &EditorService{
		wechat:    wx,
		operation: op,
	}
}

// UpdateWeChatProfile 更新微信资料
// 业务规则：至少提供一个字段，账号必须存在
func (s *EditorService) UpdateWeChatProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string, meta []byte) error {
	nick, nickSet := normalizeOptionalString(nickname)
	ava, avaSet := normalizeOptionalString(avatar)
	metaSet := len(meta) > 0
	if !nickSet && !avaSet && !metaSet {
		return perrors.WithCode(code.ErrInvalidArgument, "no profile fields provided")
	}

	// 验证微信账号存在
	if _, err := s.wechat.FindByAccountID(ctx, accountID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "wechat account for account(%s) not found", accountIDString(accountID))
		}
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to find wechat account")
	}

	// 更新资料
	if err := s.wechat.UpdateProfile(ctx, accountID, nick, ava, meta); err != nil {
		return perrors.WrapC(err, code.ErrInternalServerError, "failed to update wechat profile")
	}
	return nil
}

// SetWeChatUnionID 设置微信 UnionID
// 业务规则：unionID不能为空，账号必须存在
func (s *EditorService) SetWeChatUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error {
	unionID = strings.TrimSpace(unionID)
	if unionID == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "unionId cannot be empty")
	}

	// 验证微信账号存在
	if _, err := s.wechat.FindByAccountID(ctx, accountID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "wechat credential for account(%s) not found", accountIDString(accountID))
		}
		return perrors.WrapC(err, code.ErrDatabase, "load wechat credential for account(%s) failed", accountIDString(accountID))
	}

	// 更新UnionID
	if err := s.wechat.UpdateUnionID(ctx, accountID, unionID); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update unionId for account(%s) failed", accountIDString(accountID))
	}
	return nil
}

// UpdateOperationCredential 更新运营后台凭证信息
// 业务规则：用户名、密码哈希、算法都不能为空
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

	// 验证运营账号存在
	if _, err := s.operation.FindByUsername(ctx, username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) not found", username)
		}
		return perrors.WrapC(err, code.ErrDatabase, "load operation credential(%s) failed", username)
	}

	// 更新密码哈希
	if err := s.operation.UpdateHash(ctx, username, newHash, algo, params); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update operation credential(%s) failed", username)
	}
	return nil
}

// ChangeOperationUsername 修改运营后台账号用户名
// 业务规则：新旧用户名都不能为空，新用户名不能已存在
func (s *EditorService) ChangeOperationUsername(ctx context.Context, oldUsername, newUsername string) error {
	oldUsername = strings.TrimSpace(oldUsername)
	newUsername = strings.TrimSpace(newUsername)
	if oldUsername == "" || newUsername == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}
	if oldUsername == newUsername {
		return nil
	}

	// 验证旧账号存在
	cred, err := s.operation.FindByUsername(ctx, oldUsername)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) not found", oldUsername)
		}
		return perrors.WrapC(err, code.ErrDatabase, "load operation credential(%s) failed", oldUsername)
	}

	// 检查新用户名是否已存在
	if _, err := s.operation.FindByUsername(ctx, newUsername); err == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) already exists", newUsername)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return perrors.WrapC(err, code.ErrDatabase, "check operation credential(%s) failed", newUsername)
	}

	// 更新用户名
	if err := s.operation.UpdateUsername(ctx, cred.AccountID, newUsername); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "change operation username from %s to %s failed", oldUsername, newUsername)
	}

	return nil
}

// ResetOperationFailures 清零失败次数
func (s *EditorService) ResetOperationFailures(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}

	if err := s.operation.ResetFailures(ctx, username); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "reset failures for %s failed", username)
	}
	return nil
}

// UnlockOperationAccount 解锁账号
func (s *EditorService) UnlockOperationAccount(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}

	if err := s.operation.Unlock(ctx, username); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "unlock credential(%s) failed", username)
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
