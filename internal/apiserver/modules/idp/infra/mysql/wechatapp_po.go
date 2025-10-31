package mysql

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

// WechatAppPO 微信应用持久化对象
type WechatAppPO struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement:false" json:"id"`
	AppID     string    `gorm:"column:app_id;type:varchar(64);uniqueIndex;not null" json:"app_id"`
	Name      string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Type      string    `gorm:"column:type;type:varchar(32);not null" json:"type"`
	Status    string    `gorm:"column:status;type:varchar(32);not null;default:'Enabled'" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`

	// 凭据字段（加密存储）
	AuthSecretCipher    []byte     `gorm:"column:auth_secret_cipher;type:blob" json:"-"`
	AuthSecretFP        string     `gorm:"column:auth_secret_fp;type:varchar(128)" json:"-"`
	AuthSecretVersion   int        `gorm:"column:auth_secret_version;default:0" json:"-"`
	AuthSecretRotatedAt *time.Time `gorm:"column:auth_secret_rotated_at" json:"-"`
	MsgCallbackToken    string     `gorm:"column:msg_callback_token;type:varchar(128)" json:"-"`
	MsgAESKeyCipher     []byte     `gorm:"column:msg_aes_key_cipher;type:blob" json:"-"`
	MsgSecretVersion    int        `gorm:"column:msg_secret_version;default:0" json:"-"`
	MsgSecretRotatedAt  *time.Time `gorm:"column:msg_secret_rotated_at" json:"-"`
}

// TableName 指定表名
func (WechatAppPO) TableName() string {
	return "iam_idp_wechat_apps"
}

// ToDomain 转换为领域对象
func (po *WechatAppPO) ToDomain() *domain.WechatApp {
	if po == nil {
		return nil
	}

	app := &domain.WechatApp{
		ID:     idutil.NewID(po.ID),
		AppID:  po.AppID,
		Name:   po.Name,
		Type:   domain.AppType(po.Type),
		Status: domain.Status(po.Status),
	}

	// 转换凭据
	app.Cred = &domain.Credentials{}

	if len(po.AuthSecretCipher) > 0 {
		app.Cred.Auth = &domain.AuthSecret{
			AppSecretCipher: po.AuthSecretCipher,
			Fingerprint:     po.AuthSecretFP,
			Version:         po.AuthSecretVersion,
			LastRotatedAt:   po.AuthSecretRotatedAt,
		}
	}

	if len(po.MsgAESKeyCipher) > 0 {
		app.Cred.Msg = &domain.MsgSecret{
			CallbackToken:        po.MsgCallbackToken,
			EncodingAESKeyCipher: po.MsgAESKeyCipher,
			Version:              po.MsgSecretVersion,
			LastRotatedAt:        po.MsgSecretRotatedAt,
		}
	}

	return app
}

// FromDomain 从领域对象转换
func (po *WechatAppPO) FromDomain(app *domain.WechatApp) {
	if app == nil {
		return
	}

	po.ID = app.ID.Uint64()
	po.AppID = app.AppID
	po.Name = app.Name
	po.Type = string(app.Type)
	po.Status = string(app.Status)

	// 转换凭据
	if app.Cred != nil {
		if app.Cred.Auth != nil {
			po.AuthSecretCipher = app.Cred.Auth.AppSecretCipher
			po.AuthSecretFP = app.Cred.Auth.Fingerprint
			po.AuthSecretVersion = app.Cred.Auth.Version
			po.AuthSecretRotatedAt = app.Cred.Auth.LastRotatedAt
		}

		if app.Cred.Msg != nil {
			po.MsgCallbackToken = app.Cred.Msg.CallbackToken
			po.MsgAESKeyCipher = app.Cred.Msg.EncodingAESKeyCipher
			po.MsgSecretVersion = app.Cred.Msg.Version
			po.MsgSecretRotatedAt = app.Cred.Msg.LastRotatedAt
		}
	}
}
