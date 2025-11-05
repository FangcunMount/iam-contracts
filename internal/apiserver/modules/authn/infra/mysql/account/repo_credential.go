package account

import (
	"context"
	"fmt"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// CredentialRepository 凭据仓储实现
type CredentialRepository struct {
	db     *gorm.DB
	mapper *Mapper
}

// NewCredentialRepository 创建凭据仓储实例
func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{
		db:     db,
		mapper: NewMapper(),
	}
}

// ==================== 创建 ====================

// Create 创建凭据
func (r *CredentialRepository) Create(ctx context.Context, cred *domain.Credential) error {
	po := r.mapper.ToCredentialPO(cred)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}
	// 回填生成的 ID
	cred.ID = int64(po.ID.Uint64())
	return nil
}

// ==================== 更新 ====================

// UpdateMaterial 更新凭据材料（用于密码重置、轮换等）
func (r *CredentialRepository) UpdateMaterial(ctx context.Context, id meta.ID, material []byte, algo string) error {
	// 使用乐观锁更新
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Updates(map[string]interface{}{
			"material": material,
			"algo":     algo,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update credential material: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateStatus 更新凭据状态
func (r *CredentialRepository) UpdateStatus(ctx context.Context, id meta.ID, status domain.CredentialStatus) error {
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Update("status", int8(status))

	if result.Error != nil {
		return fmt.Errorf("failed to update credential status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateFailedAttempts 更新失败尝试次数（用于账号锁定策略）
func (r *CredentialRepository) UpdateFailedAttempts(ctx context.Context, id meta.ID, attempts int) error {
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Update("failed_attempts", attempts)

	if result.Error != nil {
		return fmt.Errorf("failed to update credential failed_attempts: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateLockedUntil 更新锁定截止时间
func (r *CredentialRepository) UpdateLockedUntil(ctx context.Context, id meta.ID, lockedUntil *time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Update("locked_until", lockedUntil)

	if result.Error != nil {
		return fmt.Errorf("failed to update credential locked_until: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateLastSuccessAt 更新最近成功时间
func (r *CredentialRepository) UpdateLastSuccessAt(ctx context.Context, id meta.ID, lastSuccessAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Update("last_success_at", lastSuccessAt)

	if result.Error != nil {
		return fmt.Errorf("failed to update credential last_success_at: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateLastFailureAt 更新最近失败时间
func (r *CredentialRepository) UpdateLastFailureAt(ctx context.Context, id meta.ID, lastFailureAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&CredentialPO{}).
		Where("id = ?", id.ToUint64()).
		Update("last_failure_at", lastFailureAt)

	if result.Error != nil {
		return fmt.Errorf("failed to update credential last_failure_at: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateExpiresAt 更新过期时间（当前 PO 未定义此字段，返回未实现错误）
func (r *CredentialRepository) UpdateExpiresAt(ctx context.Context, id meta.ID, expiresAt *time.Time) error {
	// TODO: 如果需要支持凭据过期时间，需要在 CredentialPO 中添加 expires_at 字段
	return fmt.Errorf("UpdateExpiresAt not implemented: expires_at field not defined in CredentialPO")
}

// ==================== 查询 ====================

// GetByID 根据ID查询凭据
func (r *CredentialRepository) GetByID(ctx context.Context, id meta.ID) (*domain.Credential, error) {
	var po CredentialPO
	if err := r.db.WithContext(ctx).
		Where("id = ?", id.ToUint64()).
		First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get credential by id: %w", err)
	}
	return r.mapper.ToCredentialDO(&po), nil
}

// GetByAccountIDAndType 根据账号ID和凭据类型查询凭据
func (r *CredentialRepository) GetByAccountIDAndType(ctx context.Context, accountID meta.ID, credType domain.CredentialType) (*domain.Credential, error) {
	var po CredentialPO
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND type = ?", accountID.ToUint64(), string(credType)).
		First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get credential by account_id and type: %w", err)
	}
	return r.mapper.ToCredentialDO(&po), nil
}

// GetByIDPIdentifier 根据外部身份标识查询凭据
func (r *CredentialRepository) GetByIDPIdentifier(ctx context.Context, idpIdentifier string, credType domain.CredentialType) (*domain.Credential, error) {
	var po CredentialPO
	if err := r.db.WithContext(ctx).
		Where("idp_identifier = ? AND type = ?", idpIdentifier, string(credType)).
		First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get credential by idp_identifier: %w", err)
	}
	return r.mapper.ToCredentialDO(&po), nil
}

// ListByAccountID 根据账号ID查询所有凭据
func (r *CredentialRepository) ListByAccountID(ctx context.Context, accountID meta.ID) ([]*domain.Credential, error) {
	var pos []CredentialPO
	if err := r.db.WithContext(ctx).
		Where("account_id = ?", accountID.ToUint64()).
		Find(&pos).Error; err != nil {
		return nil, fmt.Errorf("failed to list credentials by account_id: %w", err)
	}

	credentials := make([]*domain.Credential, 0, len(pos))
	for i := range pos {
		credentials = append(credentials, r.mapper.ToCredentialDO(&pos[i]))
	}
	return credentials, nil
}

// ==================== 删除 ====================

// Delete 删除凭据（物理删除）
func (r *CredentialRepository) Delete(ctx context.Context, id meta.ID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id.ToUint64()).
		Delete(&CredentialPO{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ==================== 认证端口实现 ====================

// FindPasswordCredential 根据账户ID查找密码凭据
// 实现 port.CredentialRepository 接口
func (r *CredentialRepository) FindPasswordCredential(ctx context.Context, accountID int64) (credentialID int64, passwordHash string, err error) {
	var po CredentialPO

	if err := r.db.WithContext(ctx).
		Select("id", "material").
		Where("account_id = ? AND type = ?", accountID, "password").
		First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, "", nil // 凭据不存在，返回空值
		}
		return 0, "", fmt.Errorf("failed to find password credential: %w", err)
	}

	// material 存储的是 PHC 格式的密码哈希
	return int64(po.ID.Uint64()), string(po.Material), nil
}

// FindPhoneOTPCredential 根据手机号查找OTP凭据绑定
// 实现 port.CredentialRepository 接口
func (r *CredentialRepository) FindPhoneOTPCredential(ctx context.Context, phoneE164 string) (accountID, userID, credentialID int64, err error) {
	var result struct {
		CredentialID int64 `gorm:"column:credential_id"`
		AccountID    int64 `gorm:"column:account_id"`
		UserID       int64 `gorm:"column:user_id"`
	}

	// 联表查询获取 account_id 和 user_id
	query := `
		SELECT c.id as credential_id, c.account_id, a.user_id
		FROM iam_auth_credentials c
		JOIN iam_auth_accounts a ON c.account_id = a.id
		WHERE c.type = ? AND c.idp_identifier = ? AND c.deleted_at IS NULL AND a.deleted_at IS NULL
		LIMIT 1
	`

	if err := r.db.WithContext(ctx).Raw(query, "phone_otp", phoneE164).Scan(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, fmt.Errorf("failed to find phone OTP credential: %w", err)
	}

	if result.CredentialID == 0 {
		return 0, 0, 0, nil
	}

	return result.AccountID, result.UserID, result.CredentialID, nil
}

// FindOAuthCredential 根据身份提供商标识查找OAuth凭据绑定
// 实现 port.CredentialRepository 接口
// idpType: "wx_minip" | "wecom" | ...
// appID: 应用ID (微信AppID/企业微信CorpID等)
// idpIdentifier: OpenID/UnionID/UserID等
func (r *CredentialRepository) FindOAuthCredential(ctx context.Context, idpType, appID, idpIdentifier string) (accountID, userID, credentialID int64, err error) {
	var result struct {
		CredentialID int64 `gorm:"column:credential_id"`
		AccountID    int64 `gorm:"column:account_id"`
		UserID       int64 `gorm:"column:user_id"`
	}

	// 联表查询
	query := `
		SELECT c.id as credential_id, c.account_id, a.user_id
		FROM iam_auth_credentials c
		JOIN iam_auth_accounts a ON c.account_id = a.id
		WHERE c.type = ? AND c.app_id = ? AND c.idp_identifier = ? 
		  AND c.deleted_at IS NULL AND a.deleted_at IS NULL
		LIMIT 1
	`

	if err := r.db.WithContext(ctx).Raw(query, idpType, appID, idpIdentifier).Scan(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, fmt.Errorf("failed to find OAuth credential: %w", err)
	}

	if result.CredentialID == 0 {
		return 0, 0, 0, nil
	}

	return result.AccountID, result.UserID, result.CredentialID, nil
}
