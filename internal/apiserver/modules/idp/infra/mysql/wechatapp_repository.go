package mysql

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

// wechatAppRepository 微信应用仓储实现
type wechatAppRepository struct {
	db *gorm.DB
}

// 确保实现了接口
var _ port.WechatAppRepository = (*wechatAppRepository)(nil)

// NewWechatAppRepository 创建微信应用仓储实例
func NewWechatAppRepository(db *gorm.DB) port.WechatAppRepository {
	return &wechatAppRepository{db: db}
}

// Create 创建微信应用
func (r *wechatAppRepository) Create(ctx context.Context, app *domain.WechatApp) error {
	if app == nil {
		return errors.New("app cannot be nil")
	}

	po := &WechatAppPO{}
	po.FromDomain(app)

	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return fmt.Errorf("failed to create wechat app: %w", err)
	}

	return nil
}

// GetByID 根据 ID 查询微信应用
func (r *wechatAppRepository) GetByID(ctx context.Context, id idutil.ID) (*domain.WechatApp, error) {
	po := &WechatAppPO{}

	if err := r.db.WithContext(ctx).Where("id = ?", id.Uint64()).First(po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wechat app by id: %w", err)
	}

	return po.ToDomain(), nil
}

// GetByAppID 根据 AppID 查询微信应用
func (r *wechatAppRepository) GetByAppID(ctx context.Context, appID string) (*domain.WechatApp, error) {
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}

	po := &WechatAppPO{}

	if err := r.db.WithContext(ctx).Where("app_id = ?", appID).First(po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wechat app by appID: %w", err)
	}

	return po.ToDomain(), nil
}

// Update 更新微信应用
func (r *wechatAppRepository) Update(ctx context.Context, app *domain.WechatApp) error {
	if app == nil {
		return errors.New("app cannot be nil")
	}

	po := &WechatAppPO{}
	po.FromDomain(app)

	// 只更新变更的字段
	result := r.db.WithContext(ctx).Model(&WechatAppPO{}).Where("id = ?", po.ID).Updates(map[string]interface{}{
		"name":                   po.Name,
		"type":                   po.Type,
		"status":                 po.Status,
		"auth_secret_cipher":     po.AuthSecretCipher,
		"auth_secret_fp":         po.AuthSecretFP,
		"auth_secret_version":    po.AuthSecretVersion,
		"auth_secret_rotated_at": po.AuthSecretRotatedAt,
		"msg_callback_token":     po.MsgCallbackToken,
		"msg_aes_key_cipher":     po.MsgAESKeyCipher,
		"msg_secret_version":     po.MsgSecretVersion,
		"msg_secret_rotated_at":  po.MsgSecretRotatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update wechat app: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("wechat app not found")
	}

	return nil
}
