package register

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= 应用服务接口（Driving Ports）=============

// RegisterApplicationService 注册应用服务 - 完整的用户注册流程
type RegisterApplicationService interface {
	// Register 统一注册接口
	// 完成：1) 创建User 2) 创建Account 3) 绑定Credential 4) 返回用户信息
	Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error)
}

// ============= DTOs =============

// CredentialType 凭据类型（用于注册）
type CredentialType string

const (
	CredTypePassword CredentialType = "password" // 密码
	CredTypePhone    CredentialType = "phone"    // 手机号
	CredTypeWechat   CredentialType = "wechat"   // 微信小程序
	CredTypeWecom    CredentialType = "wecom"    // 企业微信
)

// RegisterRequest 统一注册请求
type RegisterRequest struct {
	// ========== 用户基本信息（必须）==========
	Name  string     // 用户姓名
	Phone meta.Phone // 手机号（E.164格式）
	Email meta.Email // 邮箱（可选）

	// ========== 凭据信息 ==========
	CredentialType CredentialType // 凭据类型

	// 密码凭据
	Password *string // 密码（当 CredentialType = password 时必须）

	// 手机OTP凭据
	// Phone 已在用户基本信息中，无需额外字段

	// 微信凭据
	WechatAppID   *string // 微信AppID（当 CredentialType = wechat 时必须）
	WechatOpenID  *string // 微信OpenID（当 CredentialType = wechat 时必须）
	WechatUnionID *string // 微信UnionID（可选）

	// 企业微信凭据
	WecomCorpID *string // 企业CorpID（当 CredentialType = wecom 时必须）
	WecomUserID *string // 企业微信UserID（当 CredentialType = wecom 时必须）

	// ========== 账户元数据（可选）==========
	Profile    map[string]string // 用户资料（昵称、头像等）
	Meta       map[string]string // 额外元数据
	ParamsJSON []byte            // 第三方平台用户信息JSON
}

// RegisterResult 注册结果
type RegisterResult struct {
	// 用户信息
	UserID     meta.ID               // 用户ID
	UserName   string                // 用户姓名
	Phone      meta.Phone            // 手机号
	Email      meta.Email            // 邮箱
	UserStatus userDomain.UserStatus // 用户状态

	// 账户信息
	AccountID   meta.ID            // 账户ID
	AccountType domain.AccountType // 账户类型
	ExternalID  domain.ExternalID  // 外部标识

	// 凭据信息
	CredentialID int64 // 凭据ID

	// 状态
	IsNewUser    bool // 是否新建用户（true=新建，false=已存在）
	IsNewAccount bool // 是否新建账户（true=新建，false=已存在）
}
