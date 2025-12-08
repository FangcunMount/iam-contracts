// Package interceptors 提供通用 gRPC 拦截器类型和工具
//
// 本包定义了可复用的认证授权类型，用于构建 gRPC 安全拦截器链。
// 这些类型与具体业务逻辑解耦，可被不同的服务项目引用。
package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ===== 服务身份相关类型 =====

// ServiceIdentity 服务身份信息（从 mTLS 证书或应用层凭证提取）
type ServiceIdentity struct {
	// 服务名称（从证书 CN 或 SAN 中的服务标识提取）
	ServiceName string `json:"service_name"`
	// 服务命名空间（如 production, staging）
	ServiceNamespace string `json:"service_namespace,omitempty"`
	// 证书通用名称
	CommonName string `json:"common_name,omitempty"`
	// 组织单元列表
	OrganizationalUnits []string `json:"organizational_units,omitempty"`
	// 证书序列号（用于追踪和撤销）
	CertSerialNumber string `json:"cert_serial_number,omitempty"`
	// DNS SANs
	DNSSANs []string `json:"dns_sans,omitempty"`
	// URI SANs
	URISANs []string `json:"uri_sans,omitempty"`
	// 证书有效期
	NotBefore time.Time `json:"not_before,omitempty"`
	NotAfter  time.Time `json:"not_after,omitempty"`
}

// ===== 凭证相关类型 =====

// CredentialType 凭证类型
type CredentialType string

const (
	CredentialTypeBearer CredentialType = "bearer"
	CredentialTypeHMAC   CredentialType = "hmac"
	CredentialTypeAPIKey CredentialType = "api_key"
)

// ServiceCredential 服务凭证（用于应用层鉴权）
type ServiceCredential struct {
	// 凭证类型
	Type CredentialType `json:"type"`

	// Bearer Token 相关
	Token     string `json:"token,omitempty"`
	TokenType string `json:"token_type,omitempty"` // jwt, opaque

	// HMAC 相关
	AccessKey string `json:"access_key,omitempty"`
	Signature string `json:"signature,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`

	// 解析后的信息
	Subject     string            `json:"subject,omitempty"`     // 凭证主体（服务名）
	Permissions []string          `json:"permissions,omitempty"` // 权限列表
	ExpiresAt   time.Time         `json:"expires_at,omitempty"`  // 过期时间
	Claims      map[string]string `json:"claims,omitempty"`      // 额外声明
}

// ===== 上下文键定义 =====

// 上下文键类型（避免冲突）
type contextKey string

const (
	// ServiceIdentityKey 服务身份上下文键
	ServiceIdentityKey contextKey = "grpc_service_identity"
	// ServiceCredentialKey 服务凭证上下文键
	ServiceCredentialKey contextKey = "grpc_service_credential"
)

// ContextWithServiceIdentity 将服务身份注入上下文
func ContextWithServiceIdentity(ctx context.Context, identity *ServiceIdentity) context.Context {
	return context.WithValue(ctx, ServiceIdentityKey, identity)
}

// ServiceIdentityFromContext 从上下文获取服务身份
func ServiceIdentityFromContext(ctx context.Context) (*ServiceIdentity, bool) {
	identity, ok := ctx.Value(ServiceIdentityKey).(*ServiceIdentity)
	return identity, ok
}

// ContextWithCredential 将凭证注入上下文
func ContextWithCredential(ctx context.Context, cred *ServiceCredential) context.Context {
	return context.WithValue(ctx, ServiceCredentialKey, cred)
}

// CredentialFromContext 从上下文获取服务凭证
func CredentialFromContext(ctx context.Context) (*ServiceCredential, bool) {
	cred, ok := ctx.Value(ServiceCredentialKey).(*ServiceCredential)
	return cred, ok
}

// ===== 接口定义 =====

// IdentityExtractor 服务身份提取器接口
type IdentityExtractor interface {
	// Extract 从上下文提取服务身份
	Extract(ctx context.Context) (*ServiceIdentity, error)
}

// IdentityValidator 服务身份验证器接口
type IdentityValidator interface {
	// Validate 验证服务身份
	Validate(identity *ServiceIdentity) error
}

// CredentialExtractor 凭证提取器接口
type CredentialExtractor interface {
	// Extract 从上下文提取凭证
	Extract(ctx context.Context) (*ServiceCredential, error)
}

// CredentialValidator 凭证验证器接口
type CredentialValidator interface {
	// Validate 验证凭证，返回验证后的凭证信息
	Validate(ctx context.Context, cred *ServiceCredential) (*ServiceCredential, error)
}

// AccessChecker 访问权限检查器接口
type AccessChecker interface {
	// CheckAccess 检查服务是否有权访问指定方法
	CheckAccess(serviceName, method string) error
}

// AuditLogger 审计日志记录器接口
type AuditLogger interface {
	// Log 记录审计事件
	Log(event *AuditEvent)
}

// ===== 审计事件 =====

// AuditEvent 审计事件
type AuditEvent struct {
	// 时间信息
	Timestamp time.Time `json:"timestamp"`

	// 请求信息
	Method    string `json:"method"`
	RequestID string `json:"request_id,omitempty"`

	// 调用方信息
	ServiceName      string   `json:"service_name"`
	ServiceNamespace string   `json:"service_namespace,omitempty"`
	CertCN           string   `json:"cert_cn,omitempty"`
	CertOU           []string `json:"cert_ou,omitempty"`

	// 凭证信息
	CredentialType    string `json:"credential_type,omitempty"`
	CredentialSubject string `json:"credential_subject,omitempty"`

	// 结果信息
	StatusCode string        `json:"status_code"`
	StatusMsg  string        `json:"status_msg,omitempty"`
	Duration   time.Duration `json:"duration"`

	// 客户端信息
	ClientAddr string `json:"client_addr,omitempty"`

	// 错误信息
	Error string `json:"error,omitempty"`
}

// ===== 通用工具 =====

// WrappedServerStream 包装 ServerStream 以注入自定义上下文
type WrappedServerStream struct {
	grpc.ServerStream
	Ctx context.Context
}

// Context 返回自定义上下文
func (w *WrappedServerStream) Context() context.Context {
	return w.Ctx
}

// MethodMatcher 方法匹配接口
type MethodMatcher interface {
	// Match 检查方法是否匹配
	Match(method string) bool
}

// SkipMethodMatcher 跳过方法匹配器
type SkipMethodMatcher struct {
	methods []string
}

// NewSkipMethodMatcher 创建跳过方法匹配器
func NewSkipMethodMatcher(methods ...string) *SkipMethodMatcher {
	return &SkipMethodMatcher{methods: methods}
}

// Match 检查方法是否应该被跳过
func (m *SkipMethodMatcher) Match(method string) bool {
	for _, skip := range m.methods {
		if skip == method {
			return true
		}
	}
	return false
}

// Add 添加要跳过的方法
func (m *SkipMethodMatcher) Add(methods ...string) {
	m.methods = append(m.methods, methods...)
}

// DefaultSkipMethods 返回默认应跳过认证的方法列表
func DefaultSkipMethods() []string {
	return []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}
}
