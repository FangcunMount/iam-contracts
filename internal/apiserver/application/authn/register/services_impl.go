package register

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	credDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/credential"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// ============= RegisterApplicationService 实现 =============

type registerApplicationService struct {
	uow      uow.UnitOfWork
	userRepo userDomain.Repository
	hasher   authentication.PasswordHasher
	idp      authentication.IdentityProvider
}

var _ RegisterApplicationService = (*registerApplicationService)(nil)

func NewRegisterApplicationService(
	uow uow.UnitOfWork,
	hasher authentication.PasswordHasher,
	idp authentication.IdentityProvider,
	userRepo userDomain.Repository,
) RegisterApplicationService {
	return &registerApplicationService{
		uow:      uow,
		userRepo: userRepo,
		hasher:   hasher,
		idp:      idp,
	}
}

// Register 统一注册接口（使用领域层策略模式 + 凭据绑定分离）
func (s *registerApplicationService) Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
	var result *RegisterResult

	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// ========== 步骤1: 创建或获取 User ==========
		user, isNewUser, err := s.createOrGetUser(ctx, req)
		if err != nil {
			return err
		}

		// ========== 步骤2: 根据 AccountType 创建账户 ==========
		// 构造领域层输入
		domainInput := s.toDomainInput(req, user.ID)

		// 创建账户创建器（策略选择在领域层内部完成）
		accountCreator := domain.NewAccountCreator(tx.Accounts, s.idp)

		// 创建账户实体（不包含持久化）
		account, creationParams, err := accountCreator.CreateAccount(ctx, domainInput)
		if err != nil {
			return err
		}

		isNewAccount := false
		// 如果账户是新创建的（不是从数据库查到的），需要持久化
		if account.ID.IsZero() {
			if err := tx.Accounts.Create(ctx, account); err != nil {
				return perrors.WithCode(code.ErrDatabase, "failed to save account: %v", err)
			}
			isNewAccount = true
		}

		// ========== 步骤3: 根据 CredentialType 颁发凭据 ==========
		credIssuer := credDomain.NewIssuer(s.hasher)
		credential, err := s.issueCredential(ctx, credIssuer, account.ID, creationParams, req)
		if err != nil {
			return err
		}

		// 持久化凭据
		if err := tx.Credentials.Create(ctx, credential); err != nil {
			return perrors.WithCode(code.ErrDatabase, "failed to save credential: %v", err)
		}

		// ========== 步骤4: 构造返回结果 ==========
		result = &RegisterResult{
			// 用户信息
			UserID:     user.ID,
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

// issueCredential 根据凭据类型颁发凭据
func (s *registerApplicationService) issueCredential(
	ctx context.Context,
	issuer credDomain.Issuer,
	accountID meta.ID,
	creationParams *domain.CreationParams,
	req RegisterRequest,
) (*credDomain.Credential, error) {
	switch req.CredentialType {
	case CredTypePassword:
		// 颁发密码凭据
		if req.Password == nil || *req.Password == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "password is required")
		}
		return issuer.IssuePassword(ctx, credDomain.IssuePasswordRequest{
			AccountID:     accountID,
			PlainPassword: *req.Password,
		})

	case CredTypePhone:
		// 颁发手机OTP凭据
		return issuer.IssuePhoneOTP(ctx, credDomain.IssuePhoneOTPRequest{
			AccountID: accountID,
			Phone:     req.Phone,
		})

	case CredTypeWechat:
		// 颁发微信凭据
		if creationParams == nil || creationParams.OpenID == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "openid is required for wechat credential")
		}
		// 优先使用 UnionID 作为标识符
		idpIdentifier := creationParams.OpenID
		if creationParams.UnionID != "" {
			idpIdentifier = creationParams.UnionID
		}
		appID := ""
		if req.WechatAppID != nil {
			appID = *req.WechatAppID
		}
		return issuer.IssueWechatMinip(ctx, credDomain.IssueOAuthRequest{
			AccountID:     accountID,
			IDPIdentifier: idpIdentifier,
			AppID:         appID,
			ParamsJSON:    req.ParamsJSON,
		})

	case CredTypeWecom:
		// 颁发企业微信凭据
		if req.WecomUserID == nil || *req.WecomUserID == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "wecom userid is required")
		}
		appID := ""
		if req.WecomCorpID != nil {
			appID = *req.WecomCorpID
		}
		return issuer.IssueWecom(ctx, credDomain.IssueOAuthRequest{
			AccountID:     accountID,
			IDPIdentifier: *req.WecomUserID,
			AppID:         appID,
			ParamsJSON:    req.ParamsJSON,
		})

	default:
		return nil, perrors.WithCode(code.ErrInvalidArgument, "unsupported credential type: %s", req.CredentialType)
	}
} // toDomainInput 将应用层DTO转换为领域层输入
func (s *registerApplicationService) toDomainInput(req RegisterRequest, userID meta.ID) domain.CreationInput {
	return domain.CreationInput{
		UserID:          userID,
		Phone:           req.Phone,
		Email:           req.Email,
		AccountType:     req.AccountType,
		WechatAppID:     req.WechatAppID,
		WechatAppSecret: req.WechatAppSecret,
		WechatJsCode:    req.WechatJsCode,
		WechatOpenID:    req.WechatOpenID,
		WechatUnionID:   req.WechatUnionID,
		WecomCorpID:     req.WecomCorpID,
		WecomUserID:     req.WecomUserID,
		Profile:         req.Profile,
		Meta:            req.Meta,
		ParamsJSON:      req.ParamsJSON,
	}
}

// ============= 内部辅助方法 =============

// createOrGetUser 创建或获取用户（步骤1）
func (s *registerApplicationService) createOrGetUser(ctx context.Context, req RegisterRequest) (*userDomain.User, bool, error) {
	// 通过手机号查找现有用户
	existingUser, err := s.userRepo.FindByPhone(ctx, req.Phone)
	if err != nil && !perrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// 用户已存在
	if existingUser != nil {
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
