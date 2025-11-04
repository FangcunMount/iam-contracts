package register

import (
	"context"
	"strconv"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	authnDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	authnService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/service"
	authnInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	ucUow "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	ucUserApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"gorm.io/gorm"
)

// WeChatRegisterer 微信注册器
// 实现 Registerer 接口，负责微信账号注册
type WeChatRegisterer struct {
	db *gorm.DB // 数据库连接，用于跨模块事务
}

// NewWeChatRegisterer 创建微信注册器
func NewWeChatRegisterer(db *gorm.DB) *WeChatRegisterer {
	return &WeChatRegisterer{
		db: db,
	}
}

// Type 返回注册器类型
func (r *WeChatRegisterer) Type() string {
	return "wechat"
}

// Register 执行微信注册
// 在单个事务中原子性地创建：User（UC模块）+ Account（Authn模块）+ WeChatAccount（Authn模块）
func (r *WeChatRegisterer) Register(ctx context.Context, request interface{}) (*RegisterResponse, error) {
	// 类型断言
	req, ok := request.(*RegisterWithWeChatRequest)
	if !ok {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "invalid request type for wechat registerer")
	}

	var response *RegisterResponse

	// 使用数据库事务确保原子性
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ========== 步骤1: 创建 User（UC模块）==========
		// 创建 UC 模块的 UnitOfWork
		ucUnitOfWork := ucUow.NewUnitOfWork(tx)

		// 创建用户应用服务
		userAppService := ucUserApp.NewUserApplicationService(ucUnitOfWork)

		// 构建注册 DTO
		registerUserDTO := ucUserApp.RegisterUserDTO{
			Name:  req.Name,
			Phone: req.Phone,
			Email: req.Email,
		}

		// 调用用户注册服务
		userResult, err := userAppService.Register(ctx, registerUserDTO)
		if err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "failed to register user")
		}

		// 解析用户 ID
		userID, err := strconv.ParseUint(userResult.ID, 10, 64)
		if err != nil {
			return perrors.WrapC(err, code.ErrInvalidArgument, "invalid user id")
		}

		// ========== 步骤2: 创建 Account（Authn模块）==========
		// 创建账号仓储
		accountRepo := authnInfra.NewAccountRepository(tx)

		// 准备 appID 指针
		var appIDPtr *string
		if req.AppID != "" {
			appIDPtr = &req.AppID
		}

		// 使用领域服务创建账号实体
		account, err := authnService.CreateAccount(
			authnDomain.UserID(userID),
			authnDomain.ProviderWeChat,
			req.OpenID, // ExternalID 使用 OpenID
			appIDPtr,   // AppID 指针
		)
		if err != nil {
			return err
		}

		// 持久化账号
		if err := accountRepo.Create(ctx, account); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "failed to create account")
		}

		// ========== 步骤3: 创建 WeChatAccount（Authn模块）==========
		// 创建微信账号仓储
		wechatRepo := authnInfra.NewWeChatRepository(tx)

		// 准备微信账号选项
		var wechatOpts []authnDomain.WeChatAccountOption
		if req.UnionID != nil && *req.UnionID != "" {
			wechatOpts = append(wechatOpts, authnDomain.WithWeChatUnionID(*req.UnionID))
		}
		if req.Nickname != nil && *req.Nickname != "" {
			wechatOpts = append(wechatOpts, authnDomain.WithWeChatNickname(*req.Nickname))
		}
		if req.Avatar != nil && *req.Avatar != "" {
			wechatOpts = append(wechatOpts, authnDomain.WithWeChatAvatarURL(*req.Avatar))
		}
		if len(req.Meta) > 0 {
			// 将 map 转换为 JSON bytes
			metaBytes, err := serializeMetaToJSON(req.Meta)
			if err != nil {
				return perrors.WrapC(err, code.ErrEncodingJSON, "failed to serialize wechat meta")
			}
			wechatOpts = append(wechatOpts, authnDomain.WithWeChatMeta(metaBytes))
		}

		// 使用领域服务创建微信账号实体
		wechatAccount, err := authnService.CreateWeChatAccount(
			account.ID,
			req.AppID,
			req.OpenID,
			wechatOpts...,
		)
		if err != nil {
			return err
		}

		// 持久化微信账号
		if err := wechatRepo.Create(ctx, wechatAccount); err != nil {
			return perrors.WrapC(err, code.ErrDatabase, "failed to create wechat account")
		}

		// ========== 构建返回结果 ==========
		response = &RegisterResponse{
			UserID:    userID,
			AccountID: idutil.ID(account.ID).Uint64(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}
