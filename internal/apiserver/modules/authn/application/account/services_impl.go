package account

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	domainService "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/service"
	authPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// ============= AccountApplicationService 实现 =============

// accountApplicationService 账户应用服务实现
type accountApplicationService struct {
	uow uow.UnitOfWork
}

// accountApplicationService 实现 AccountApplicationService 接口
var _ AccountApplicationService = (*accountApplicationService)(nil)

// NewAccountApplicationService 创建账户应用服务
func NewAccountApplicationService(uow uow.UnitOfWork) AccountApplicationService {
	return &accountApplicationService{uow: uow}
}

// GetAccountByID 根据ID获取账户
func (s *accountApplicationService) GetAccountByID(ctx context.Context, accountID meta.ID) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) FindByExternalRef(
	ctx context.Context,
	accountType domain.AccountType,
	appID domain.AppId,
	externalID domain.ExternalID,
) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByExternalIDAppId(ctx, externalID, appID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) FindByUniqueID(ctx context.Context, uniqueID domain.UnionID) (*AccountResult, error) {
	var result *AccountResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByUniqueID(ctx, uniqueID)
		if err != nil {
			if perrors.Is(err, gorm.ErrRecordNotFound) {
				return perrors.WithCode(code.ErrCredentialNotFound, "account not found")
			}
			return err
		}
		result = toAccountResult(account)
		return nil
	})
	return result, err
}

func (s *accountApplicationService) SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID domain.UnionID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domainService.NewAccountEditor(tx.Accounts)
		_, err := editor.SetUniqueID(ctx, accountID, uniqueID)
		return err
	})
}

func (s *accountApplicationService) UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domainService.NewAccountEditor(tx.Accounts)
		_, err := editor.UpdateProfile(ctx, accountID, profile)
		return err
	})
}

func (s *accountApplicationService) UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		editor := domainService.NewAccountEditor(tx.Accounts)
		_, err := editor.UpdateMeta(ctx, accountID, meta)
		return err
	})
}

func (s *accountApplicationService) EnableAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			return err
		}
		sm, err := domainService.NewAccountStateMachine(account)
		if err != nil {
			return err
		}
		return sm.Activate()
	})
}

func (s *accountApplicationService) DisableAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			return err
		}
		sm, err := domainService.NewAccountStateMachine(account)
		if err != nil {
			return err
		}
		return sm.Disable()
	})
}

func (s *accountApplicationService) ArchiveAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			return err
		}
		sm, err := domainService.NewAccountStateMachine(account)
		if err != nil {
			return err
		}
		return sm.Archive()
	})
}

func (s *accountApplicationService) DeleteAccount(ctx context.Context, accountID meta.ID) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		account, err := tx.Accounts.GetByID(ctx, accountID)
		if err != nil {
			return err
		}
		sm, err := domainService.NewAccountStateMachine(account)
		if err != nil {
			return err
		}
		return sm.Delete()
	})
}

// ============= CredentialApplicationService 实现 =============

type credentialApplicationService struct {
	uow    uow.UnitOfWork
	hasher authPort.PasswordHasher
}

var _ CredentialApplicationService = (*credentialApplicationService)(nil)

func NewCredentialApplicationService(uow uow.UnitOfWork, hasher authPort.PasswordHasher) CredentialApplicationService {
	return &credentialApplicationService{
		uow:    uow,
		hasher: hasher,
	}
}

func (s *credentialApplicationService) BindCredential(ctx context.Context, dto BindCredentialDTO) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		binder := domainService.NewCredentialBinder()
		spec := port.BindSpec{
			AccountID:     int64(dto.AccountID.ToUint64()),
			Type:          dto.Type,
			IDP:           dto.IDP,
			IDPIdentifier: dto.IDPIdentifier,
			AppID:         dto.AppID,
			Material:      dto.Material,
			Algo:          dto.Algo,
			ParamsJSON:    dto.ParamsJSON,
		}
		credential, err := binder.Bind(spec)
		if err != nil {
			return err
		}
		return tx.Credentials.Create(ctx, credential)
	})
}

func (s *credentialApplicationService) UnbindCredential(ctx context.Context, credentialID int64) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		return tx.Credentials.Delete(ctx, meta.NewID(uint64(credentialID)))
	})
}

func (s *credentialApplicationService) RotatePassword(
	ctx context.Context,
	accountID meta.ID,
	oldPassword, newPassword string,
) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		// 获取密码凭据
		credential, err := tx.Credentials.GetByAccountIDAndType(ctx, accountID, domain.CredPassword)
		if err != nil {
			return perrors.WithCode(code.ErrCredentialNotFound, "password credential not found")
		}

		// 验证旧密码
		if !s.verifyPassword(oldPassword, credential) {
			return perrors.WithCode(code.ErrPasswordIncorrect, "old password is incorrect")
		}

		// 使用 PHC 格式哈希新密码
		hashedPassword, err := s.hashPassword(newPassword)
		if err != nil {
			return perrors.WithCode(code.ErrEncrypt, "failed to hash password: %v", err)
		}

		// 使用领域服务轮换
		rotator := domainService.NewCredentialRotator()
		newMaterial := []byte(hashedPassword)
		newAlgo := "argon2id"
		rotator.Rotate(credential, newMaterial, &newAlgo)

		// 持久化
		return tx.Credentials.UpdateMaterial(ctx, meta.NewID(uint64(credential.ID)), newMaterial, newAlgo)
	})
}

// verifyPassword 验证密码（内部辅助方法）
func (s *credentialApplicationService) verifyPassword(plainPassword string, credential *domain.Credential) bool {
	if credential.Material == nil || len(credential.Material) == 0 {
		return false
	}

	// 添加 pepper
	plaintextWithPepper := plainPassword + s.hasher.Pepper()
	storedHash := string(credential.Material)

	// 使用 hasher 验证
	return s.hasher.Verify(storedHash, plaintextWithPepper)
}

// hashPassword 使用 PHC 格式哈希密码（内部辅助方法）
func (s *credentialApplicationService) hashPassword(plainPassword string) (string, error) {
	// 添加 pepper
	plaintextWithPepper := plainPassword + s.hasher.Pepper()

	// 使用 hasher 生成 PHC 格式哈希
	return s.hasher.Hash(plaintextWithPepper)
}

func (s *credentialApplicationService) GetCredentialsByAccountID(ctx context.Context, accountID meta.ID) ([]*CredentialResult, error) {
	var results []*CredentialResult
	err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		credentials, err := tx.Credentials.ListByAccountID(ctx, accountID)
		if err != nil {
			return err
		}
		results = make([]*CredentialResult, len(credentials))
		for i, cred := range credentials {
			results[i] = toCredentialResult(cred)
		}
		return nil
	})
	return results, err
}

func (s *credentialApplicationService) DisableCredential(ctx context.Context, credentialID int64) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		return tx.Credentials.UpdateStatus(ctx, meta.NewID(uint64(credentialID)), domain.CredStatusDisabled)
	})
}

func (s *credentialApplicationService) EnableCredential(ctx context.Context, credentialID int64) error {
	return s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		return tx.Credentials.UpdateStatus(ctx, meta.NewID(uint64(credentialID)), domain.CredStatusEnabled)
	})
}

// ============= Helper Functions =============

func toAccountResult(account *domain.Account) *AccountResult {
	return &AccountResult{
		AccountID:  account.ID,
		UserID:     account.UserID,
		Type:       account.Type,
		AppID:      account.AppID,
		ExternalID: account.ExternalID,
		UniqueID:   account.UniqueID,
		Profile:    account.Profile,
		Meta:       account.Meta,
		Status:     account.Status,
	}
}

func toCredentialResult(cred *domain.Credential) *CredentialResult {
	// 使用 Credential 实体的类型判断方法推断凭据类型
	var credType domain.CredentialType

	// 优先使用实体的类型判断方法
	switch {
	case cred.IsPasswordType():
		credType = domain.CredPassword
	case cred.IsPhoneOTPType():
		credType = domain.CredPhoneOTP
	case cred.IsOAuthType():
		// OAuth 类型需要进一步判断具体的 IDP
		if cred.IDP != nil {
			switch *cred.IDP {
			case "wechat":
				credType = domain.CredOAuthWxMinip
			case "wecom":
				credType = domain.CredOAuthWecom
			default:
				// 未知的 OAuth 类型，使用默认值
				credType = domain.CredentialType(*cred.IDP)
			}
		}
	}

	return &CredentialResult{
		ID:            cred.ID,
		AccountID:     cred.AccountID,
		Type:          credType,
		IDP:           cred.IDP,
		IDPIdentifier: cred.IDPIdentifier,
		AppID:         cred.AppID,
		Status:        cred.Status,
	}
}
