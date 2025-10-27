package account

import (
	"context"
	"errors"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	domainService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/service"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"gorm.io/gorm"
)

// wechatAccountApplicationService 微信账号应用服务实现
type wechatAccountApplicationService struct {
	uow uow.UnitOfWork
}

var _ WeChatAccountApplicationService = (*wechatAccountApplicationService)(nil)

// NewWeChatAccountApplicationService 创建微信账号应用服务
func NewWeChatAccountApplicationService(
	uow uow.UnitOfWork,
) WeChatAccountApplicationService {
	return &wechatAccountApplicationService{
		uow: uow,
	}
}

// BindWeChatAccount 绑定微信账号用例
// 流程：
// 1. 验证账号存在
// 2. 验证微信账号不存在（避免重复绑定）
// 3. 创建微信账号实体
// 4. 如果提供了资料，设置资料
// 5. 持久化微信账号
func (s *wechatAccountApplicationService) BindWeChatAccount(
	ctx context.Context,
	dto BindWeChatAccountDTO,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证账号存在
		account, err := tx.Accounts.FindByID(ctx, dto.AccountID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find account failed")
		}

		// 验证微信账号不存在
		if err := domainService.ValidateWeChatNotExists(
			ctx, tx.WeChats, dto.AppID, dto.ExternalID,
		); err != nil {
			return err
		}

		// 使用领域工厂方法创建微信账号实体
		wechat, err := domainService.CreateWeChatAccountEntity(
			account.ID, dto.ExternalID, dto.AppID,
		)
		if err != nil {
			return err
		}

		// 如果提供了资料，设置到实体
		if dto.Nickname != nil || dto.Avatar != nil || len(dto.Meta) > 0 {
			// 设置昵称
			if dto.Nickname != nil && *dto.Nickname != "" {
				wechat.Nickname = dto.Nickname
			}
			// 设置头像
			if dto.Avatar != nil && *dto.Avatar != "" {
				wechat.AvatarURL = dto.Avatar
			}
			// 设置元数据
			if len(dto.Meta) > 0 {
				wechat.Meta = dto.Meta
			}
		}

		// 持久化微信账号
		if err := tx.WeChats.Create(ctx, wechat); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "create wechat account failed")
		}

		return nil
	})
}

// UpdateProfile 更新微信资料用例
// 流程：
// 1. 查找微信账号
// 2. 验证至少提供一个字段
// 3. 更新资料
func (s *wechatAccountApplicationService) UpdateProfile(
	ctx context.Context,
	dto UpdateWeChatProfileDTO,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证至少提供一个字段
		hasNickname := dto.Nickname != nil && *dto.Nickname != ""
		hasAvatar := dto.Avatar != nil && *dto.Avatar != ""
		hasMeta := len(dto.Meta) > 0

		if !hasNickname && !hasAvatar && !hasMeta {
			return perrors.WithCode(code.ErrInvalidArgument, "no profile fields provided")
		}

		// 验证微信账号存在
		_, err := tx.WeChats.FindByAccountID(ctx, dto.AccountID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "wechat account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find wechat account failed")
		}

		// 更新资料
		if err := tx.WeChats.UpdateProfile(ctx, dto.AccountID, dto.Nickname, dto.Avatar, dto.Meta); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "update wechat profile failed")
		}

		return nil
	})
}

// SetUnionID 设置微信UnionID用例
// 流程：
// 1. 验证微信账号存在
// 2. 更新UnionID
func (s *wechatAccountApplicationService) SetUnionID(
	ctx context.Context,
	accountID domain.AccountID,
	unionID string,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 验证微信账号存在
		_, err := tx.WeChats.FindByAccountID(ctx, accountID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "wechat account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find wechat account failed")
		}

		// 更新UnionID
		if err := tx.WeChats.UpdateUnionID(ctx, accountID, unionID); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "update unionid failed")
		}

		return nil
	})
}

// GetByWeChatRef 根据微信引用查找账号用例
func (s *wechatAccountApplicationService) GetByWeChatRef(
	ctx context.Context,
	externalID, appID string,
) (*AccountResult, error) {
	var result *AccountResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 查找微信账号
		wechat, err := tx.WeChats.FindByAppOpenID(ctx, appID, externalID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "wechat account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find wechat account failed")
		}

		// 查找关联的账号
		account, err := tx.Accounts.FindByID(ctx, wechat.AccountID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrDatabase, "account not found")
			}
			return perrors.WrapC(err, code.ErrDatabase, "find account failed")
		}

		result = &AccountResult{
			Account:    account,
			WeChatData: wechat,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
