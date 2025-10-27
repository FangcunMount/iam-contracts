package authentication

import (
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// Authentication 认证结果实体，表示一次成功的认证
type Authentication struct {
	UserID          account.UserID    // 认证成功的用户 ID
	AccountID       account.AccountID // 使用的账号 ID
	Provider        account.Provider  // 认证提供者（如 operation, wechat）
	AuthenticatedAt time.Time         // 认证时间
	Metadata        map[string]string // 认证元数据（如 IP、设备信息等）
}

// NewAuthentication 创建认证结果
func NewAuthentication(
	userID account.UserID,
	accountID account.AccountID,
	provider account.Provider,
	metadata map[string]string,
) *Authentication {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	return &Authentication{
		UserID:          userID,
		AccountID:       accountID,
		Provider:        provider,
		AuthenticatedAt: time.Now(),
		Metadata:        metadata,
	}
}

// WithMetadata 添加元数据
func (a *Authentication) WithMetadata(key, value string) {
	if a.Metadata == nil {
		a.Metadata = make(map[string]string)
	}
	a.Metadata[key] = value
}

// GetMetadata 获取元数据
func (a *Authentication) GetMetadata(key string) (string, bool) {
	if a.Metadata == nil {
		return "", false
	}
	value, ok := a.Metadata[key]
	return value, ok
}
