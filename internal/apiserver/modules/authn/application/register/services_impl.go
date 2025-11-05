package register

import (
	"context"
	"fmt"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	domainService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/service"
	authPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	userPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// ============= RegisterApplicationService 实现 =============

type registerApplicationService struct {
	uow      uow.UnitOfWork
	hasher   authPort.PasswordHasher
	userRepo userPort.UserRepository
}

var _ RegisterApplicationService = (*registerApplicationService)(nil)

func NewRegisterApplicationService(
	uow uow.UnitOfWork,
	hasher authPort.PasswordHasher,
	userRepo userPort.UserRepository,
) RegisterApplicationService {
	return &registerApplicationService{
		uow:      uow,
		hasher:   hasher,
		userRepo: userRepo,
	}
}

// Register 统一注册接口
func (s *registerApplicationService) Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
	var result *RegisterResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// ========== 步骤1: 创建或获取 User ==========
		user, isNewUser, err := s.createOrGetUser(ctx, req)
		if err != nil {
			return err
		}

		// ========== 步骤2: 创建或获取 Account ==========
		account, isNewAccount, err := s.createOrGetAccount(ctx, tx, req, user.ID)
		if err != nil {
			return err
		}

		// ========== 步骤3: 创建并绑定 Credential ==========
		credential, err := s.createCredential(ctx, tx, req, account)
		if err != nil {
			return err
		}

		// ========== 步骤4: 构造返回结果 ==========
		result = &RegisterResult{
			// 用户信息
			UserID:     meta.NewID(user.ID.Uint64()),
			UserName:   user.Name,
			Phone:      user.Phone,
			Email:      user.Email,
			UserStatus: user.Status,

			// 账户信息
			AccountID:   account.ID,
			AccountType: account.Type,
			ExternalID:  account.ExternalID,

			// 凭据信息
			CredentialID: credential.ID,

			// 状态
			IsNewUser:    isNewUser,
			IsNewAccount: isNewAccount,
		}

		return nil
	})

	return result, err
}

// ============= 内部辅助方法 =============

// createOrGetUser 创建或获取用户（步骤1）
func (s *registerApplicationService) createOrGetUser(ctx context.Context, req RegisterRequest) (*userDomain.User, bool, error) {
	// 通过手机号查找现有用户
	existingUser, err := s.userRepo.FindByPhone(ctx, req.Phone)
	if err != nil && !perrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if existingUser != nil {
		// 用户已存在
		return existingUser, false, nil
	}

	// 创建新用户
	user, err := userDomain.NewUser(req.Name, req.Phone, func(u *userDomain.User) {
		if !req.Email.IsEmpty() {
			u.Email = req.Email
		}
	})
	if err != nil {
		return nil, false, perrors.WithCode(code.ErrUserBasicInfoInvalid, "failed to create user: %v", err)
	}

	// 持久化用户
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, false, perrors.WithCode(code.ErrDatabase, "failed to save user: %v", err)
	}

	return user, true, nil
}

// createOrGetAccount 创建或获取账户（步骤2）
func (s *registerApplicationService) createOrGetAccount(
	ctx context.Context,
	tx uow.TxRepositories,
	req RegisterRequest,
	userID userDomain.UserID,
) (*domain.Account, bool, error) {
	// 根据凭据类型确定账户类型和外部ID
	accountType, appID, externalID := s.determineAccountInfo(req)

	// 幂等性检查：通过 ExternalID + AppID 查找是否已存在账户
	existingAccount, err := tx.Accounts.GetByExternalIDAppId(ctx, externalID, appID)
	if err != nil && !perrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if existingAccount != nil {
		// 账户已存在，验证是否属于同一用户
		if existingAccount.UserID != meta.NewID(userID.Uint64()) {
			return nil, false, perrors.WithCode(code.ErrExternalExists, "account already belongs to another user")
		}
		return existingAccount, false, nil
	}

	// 创建新账户
	creater := domainService.NewAccountCreater(tx.Accounts)
	account, err := creater.Create(ctx, port.CreateAccountDTO{
		UserID:      meta.NewID(userID.Uint64()),
		AccountType: accountType,
		AppID:       appID,
		ExternalID:  externalID,
	})
	if err != nil {
		return nil, false, err
	}

	// 设置资料和元数据
	if len(req.Profile) > 0 {
		account.Profile = req.Profile
	}
	if len(req.Meta) > 0 {
		account.Meta = req.Meta
	}

	// 持久化账户
	if err := tx.Accounts.Create(ctx, account); err != nil {
		return nil, false, err
	}

	return account, true, nil
}

// createCredential 创建并绑定凭据（步骤3）
func (s *registerApplicationService) createCredential(
	ctx context.Context,
	tx uow.TxRepositories,
	req RegisterRequest,
	account *domain.Account,
) (*domain.Credential, error) {
	binder := domainService.NewCredentialBinder()

	var credential *domain.Credential
	var err error

	switch req.CredentialType {
	case CredTypePassword:
		credential, err = s.createPasswordCredential(binder, req, account)
	case CredTypePhone:
		credential, err = s.createPhoneCredential(binder, req, account)
	case CredTypeWechat:
		credential, err = s.createWechatCredential(binder, req, account)
	case CredTypeWecom:
		credential, err = s.createWecomCredential(binder, req, account)
	default:
		return nil, perrors.WithCode(code.ErrInvalidArgument, "unsupported credential type: %s", req.CredentialType)
	}

	if err != nil {
		return nil, err
	}

	// 持久化凭据
	if err := tx.Credentials.Create(ctx, credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// createPasswordCredential 创建密码凭据
func (s *registerApplicationService) createPasswordCredential(
	binder *domainService.CredentialBinder,
	req RegisterRequest,
	account *domain.Account,
) (*domain.Credential, error) {
	if req.Password == nil || *req.Password == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "password is required for password credential")
	}

	// 哈希密码（PHC 格式）
	hashedPassword, err := s.hashPassword(*req.Password)
	if err != nil {
		return nil, perrors.WithCode(code.ErrEncrypt, "failed to hash password: %v", err)
	}

	algo := "argon2id"
	return binder.Bind(port.BindSpec{
		AccountID: int64(account.ID.ToUint64()),
		Type:      domain.CredPassword,
		Material:  []byte(hashedPassword),
		Algo:      &algo,
	})
}

// createPhoneCredential 创建手机凭据
func (s *registerApplicationService) createPhoneCredential(
	binder *domainService.CredentialBinder,
	req RegisterRequest,
	account *domain.Account,
) (*domain.Credential, error) {
	idp := "phone"
	return binder.Bind(port.BindSpec{
		AccountID:     int64(account.ID.ToUint64()),
		Type:          domain.CredPhoneOTP,
		IDP:           &idp,
		IDPIdentifier: req.Phone.String(),
	})
}

// createWechatCredential 创建微信凭据
func (s *registerApplicationService) createWechatCredential(
	binder *domainService.CredentialBinder,
	req RegisterRequest,
	account *domain.Account,
) (*domain.Credential, error) {
	if req.WechatAppID == nil || *req.WechatAppID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat appid is required")
	}
	if req.WechatOpenID == nil || *req.WechatOpenID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat openid is required")
	}

	idp := "wechat"
	idpIdentifier := *req.WechatOpenID
	if req.WechatUnionID != nil && *req.WechatUnionID != "" {
		idpIdentifier = *req.WechatUnionID // 优先使用 UnionID
	}

	return binder.Bind(port.BindSpec{
		AccountID:     int64(account.ID.ToUint64()),
		Type:          domain.CredOAuthWxMinip,
		IDP:           &idp,
		IDPIdentifier: idpIdentifier,
		AppID:         req.WechatAppID,
		ParamsJSON:    req.ParamsJSON,
	})
}

// createWecomCredential 创建企业微信凭据
func (s *registerApplicationService) createWecomCredential(
	binder *domainService.CredentialBinder,
	req RegisterRequest,
	account *domain.Account,
) (*domain.Credential, error) {
	if req.WecomCorpID == nil || *req.WecomCorpID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wecom corpid is required")
	}
	if req.WecomUserID == nil || *req.WecomUserID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wecom userid is required")
	}

	idp := "wecom"
	return binder.Bind(port.BindSpec{
		AccountID:     int64(account.ID.ToUint64()),
		Type:          domain.CredOAuthWecom,
		IDP:           &idp,
		IDPIdentifier: *req.WecomUserID,
		AppID:         req.WecomCorpID,
		ParamsJSON:    req.ParamsJSON,
	})
}

// determineAccountInfo 根据凭据类型确定账户类型、AppID 和 ExternalID
func (s *registerApplicationService) determineAccountInfo(req RegisterRequest) (domain.AccountType, domain.AppId, domain.ExternalID) {
	switch req.CredentialType {
	case CredTypePassword:
		// 密码账户：运营账户
		return domain.TypeOpera, "opera", domain.ExternalID(req.Phone.String())

	case CredTypePhone:
		// 手机账户：归类为运营账户
		return domain.TypeOpera, "phone", domain.ExternalID(req.Phone.String())

	case CredTypeWechat:
		// 微信账户：使用 OpenID@AppID 作为 ExternalID
		if req.WechatAppID != nil && req.WechatOpenID != nil {
			externalID := fmt.Sprintf("%s@%s", *req.WechatOpenID, *req.WechatAppID)
			return domain.TypeWcMinip, domain.AppId(*req.WechatAppID), domain.ExternalID(externalID)
		}
		// 默认值
		return domain.TypeWcMinip, "wechat", domain.ExternalID(req.Phone.String())

	case CredTypeWecom:
		// 企业微信账户
		if req.WecomCorpID != nil && req.WecomUserID != nil {
			return domain.TypeWcCom, domain.AppId(*req.WecomCorpID), domain.ExternalID(*req.WecomUserID)
		}
		// 默认值
		return domain.TypeWcCom, "wecom", domain.ExternalID(req.Phone.String())

	default:
		// 默认为运营账户
		return domain.TypeOpera, "default", domain.ExternalID(req.Phone.String())
	}
}

// hashPassword 使用 PHC 格式哈希密码
func (s *registerApplicationService) hashPassword(plainPassword string) (string, error) {
	plaintextWithPepper := plainPassword + s.hasher.Pepper()
	return s.hasher.Hash(plaintextWithPepper)
}
