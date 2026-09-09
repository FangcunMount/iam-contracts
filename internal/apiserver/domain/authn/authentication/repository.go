package authentication

import (
	"context"
	"time"

	credDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/credential"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/loginidentity"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// ================== Repository Interfaces (Driven Ports) ==================
// 定义领域模型所依赖的仓储接口，由基础设施层提供实现

// LoginIdentityRepository 登录身份仓储（查询登录身份）
// 职责：提供登录身份查询能力
type LoginIdentityRepository interface {
	// -- 查询登录身份 ——
	FindUsernameIdentity(ctx context.Context, username string) (*LoginIdentityLookup, error)
	FindLoginIdentityByProviderKey(ctx context.Context, provider loginidentity.Provider, realm, identifier string) (*LoginIdentityLookup, error)
	FindLoginIdentityByGlobalIdentifier(ctx context.Context, provider loginidentity.Provider, globalIdentifier string) (*LoginIdentityLookup, error)

	// -- 判断登录身份是否活动 ——
	IsLoginIdentityActive(ctx context.Context, loginIdentityID meta.ID) (bool, error)
}

// LoginIdentityCredentialRepository 凭据仓储（查询身份核验证明）
// 职责：提供 LoginIdentity 绑定的长期认证材料查询能力
type LoginIdentityCredentialRepository interface {
	// -- 查询密码凭据 ——
	FindPasswordCredentialByLoginIdentity(ctx context.Context, loginIdentityID meta.ID) (*PasswordCredentialLookup, error)
}

// PasswordCredentialLookup 是密码认证所需的长期凭据读模型
type PasswordCredentialLookup struct {
	CredentialID meta.ID                     // 供核验及结果记录使用的凭据 ID
	PasswordHash string                      // 密码哈希，不是明文密码
	Status       credDomain.CredentialStatus // 凭据启用状态
	LockedUntil  *time.Time                  // 凭据锁定截止时间
}

// LoginIdentityLookup 是登录身份的读模型
type LoginIdentityLookup struct {
	LoginIdentityID  meta.ID                // 登录身份 ID
	UserID           meta.ID                // 归属的 IAM 用户 ID
	Provider         loginidentity.Provider // 登录身份提供者
	Realm            string                 // 提供者内的身份命名空间
	Identifier       string                 // 命名空间内的登录标识
	GlobalIdentifier string                 // 可选的跨命名空间标识
	Status           loginidentity.Status   // 登录身份当前状态
}
