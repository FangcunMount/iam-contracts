// Package interceptors 提供服务级 ACL 权限控制拦截器
package interceptors

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ===== ACL 核心结构 =====

// ServiceACL 服务级访问控制列表
type ServiceACL struct {
	// 服务 -> 允许的方法列表
	rules map[string]*ServicePermissions
	mu    sync.RWMutex

	// 默认策略：deny（拒绝未配置的访问）或 allow（允许未配置的访问）
	defaultPolicy string
}

// ServicePermissions 服务权限配置
type ServicePermissions struct {
	// 服务名称
	ServiceName string `json:"service_name" yaml:"service_name"`
	// 允许调用的方法列表（支持通配符）
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods"`
	// 拒绝调用的方法列表（优先级高于允许）
	DeniedMethods []string `json:"denied_methods" yaml:"denied_methods"`
	// 方法级权限（更细粒度）
	MethodPermissions map[string][]string `json:"method_permissions" yaml:"method_permissions"`
	// 是否启用
	Enabled bool `json:"enabled" yaml:"enabled"`
	// 描述
	Description string `json:"description" yaml:"description"`
}

// ACLConfig ACL 配置
type ACLConfig struct {
	// 默认策略
	DefaultPolicy string `json:"default_policy" yaml:"default_policy"` // "deny" or "allow"
	// 服务权限列表
	Services []*ServicePermissions `json:"services" yaml:"services"`
}

// NewServiceACL 创建服务 ACL
func NewServiceACL(cfg *ACLConfig) *ServiceACL {
	policy := cfg.DefaultPolicy
	if policy == "" {
		policy = "deny"
	}

	acl := &ServiceACL{
		rules:         make(map[string]*ServicePermissions),
		defaultPolicy: policy,
	}

	for _, svc := range cfg.Services {
		if svc.Enabled {
			acl.rules[svc.ServiceName] = svc
		}
	}

	return acl
}

// CheckAccess 检查服务是否有权访问指定方法
func (a *ServiceACL) CheckAccess(serviceName, method string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	perms, ok := a.rules[serviceName]
	if !ok {
		if a.defaultPolicy == "allow" {
			return nil
		}
		return fmt.Errorf("service %q not configured in ACL", serviceName)
	}

	// 检查拒绝列表（优先级最高）
	for _, denied := range perms.DeniedMethods {
		if MatchMethod(method, denied) {
			return fmt.Errorf("method %q is denied for service %q", method, serviceName)
		}
	}

	// 检查允许列表
	for _, allowed := range perms.AllowedMethods {
		if MatchMethod(method, allowed) {
			return nil
		}
	}

	// 默认拒绝
	return fmt.Errorf("method %q not allowed for service %q", method, serviceName)
}

// MatchMethod 方法匹配（支持通配符）
func MatchMethod(method, pattern string) bool {
	// 完全匹配
	if method == pattern {
		return true
	}

	// 通配符匹配 /service/* 或 /service/Method*
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}

	// 服务级通配符 /service.v1.Service/*
	if strings.HasSuffix(pattern, "/*") {
		servicePrefix := strings.TrimSuffix(pattern, "/*")
		// 提取 method 的服务部分
		if idx := strings.LastIndex(method, "/"); idx > 0 {
			methodService := method[:idx]
			if methodService == servicePrefix {
				return true
			}
		}
	}

	return false
}

// AddServicePermission 添加服务权限
func (a *ServiceACL) AddServicePermission(perms *ServicePermissions) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules[perms.ServiceName] = perms
}

// RemoveServicePermission 移除服务权限
func (a *ServiceACL) RemoveServicePermission(serviceName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.rules, serviceName)
}

// UpdateServiceMethods 更新服务允许的方法
func (a *ServiceACL) UpdateServiceMethods(serviceName string, methods []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	perms, ok := a.rules[serviceName]
	if !ok {
		return fmt.Errorf("service %q not found", serviceName)
	}

	perms.AllowedMethods = methods
	return nil
}

// ListServices 列出所有配置的服务
func (a *ServiceACL) ListServices() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	services := make([]string, 0, len(a.rules))
	for svc := range a.rules {
		services = append(services, svc)
	}
	return services
}

// GetServicePermissions 获取服务权限配置
func (a *ServiceACL) GetServicePermissions(serviceName string) (*ServicePermissions, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	perms, ok := a.rules[serviceName]
	return perms, ok
}

// ===== ACL 拦截器 =====

// ACLInterceptor 服务 ACL 授权拦截器
func ACLInterceptor(acl AccessChecker, opts ...ACLOption) grpc.UnaryServerInterceptor {
	options := defaultACLOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 检查是否跳过
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(ctx, req)
		}

		// 获取服务身份
		serviceName := ""

		// 优先从 mTLS 身份获取
		if identity, ok := ServiceIdentityFromContext(ctx); ok {
			serviceName = identity.ServiceName
		}

		// 如果启用了凭证身份，从凭证获取
		if serviceName == "" && options.useCredentialIdentity {
			if cred, ok := CredentialFromContext(ctx); ok && cred.Subject != "" {
				serviceName = cred.Subject
			}
		}

		if serviceName == "" {
			if options.logger != nil {
				options.logger.LogError("ACL check failed: no service identity",
					map[string]interface{}{
						"method": info.FullMethod,
					})
			}
			return nil, status.Error(codes.Unauthenticated, "service identity required")
		}

		// 检查访问权限
		if err := acl.CheckAccess(serviceName, info.FullMethod); err != nil {
			if options.logger != nil {
				options.logger.LogError("ACL check failed: permission denied",
					map[string]interface{}{
						"method":  info.FullMethod,
						"service": serviceName,
						"error":   err.Error(),
					})
			}
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		if options.logger != nil {
			options.logger.LogInfo("ACL check passed",
				map[string]interface{}{
					"method":  info.FullMethod,
					"service": serviceName,
				})
		}

		return handler(ctx, req)
	}
}

// ACLStreamInterceptor 流式 ACL 授权拦截器
func ACLStreamInterceptor(acl AccessChecker, opts ...ACLOption) grpc.StreamServerInterceptor {
	options := defaultACLOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		serviceName := ""

		if identity, ok := ServiceIdentityFromContext(ctx); ok {
			serviceName = identity.ServiceName
		}

		if serviceName == "" && options.useCredentialIdentity {
			if cred, ok := CredentialFromContext(ctx); ok && cred.Subject != "" {
				serviceName = cred.Subject
			}
		}

		if serviceName == "" {
			return status.Error(codes.Unauthenticated, "service identity required")
		}

		if err := acl.CheckAccess(serviceName, info.FullMethod); err != nil {
			return status.Error(codes.PermissionDenied, err.Error())
		}

		return handler(srv, ss)
	}
}

// ===== ACL 选项 =====

type aclOptions struct {
	skipMatcher           *SkipMethodMatcher
	useCredentialIdentity bool
	logger                InterceptorLogger
}

func defaultACLOptions() *aclOptions {
	return &aclOptions{
		skipMatcher:           NewSkipMethodMatcher(DefaultSkipMethods()...),
		useCredentialIdentity: true,
	}
}

// ACLOption ACL 拦截器选项函数
type ACLOption func(*aclOptions)

// WithACLSkipMethods 设置跳过检查的方法
func WithACLSkipMethods(methods ...string) ACLOption {
	return func(o *aclOptions) {
		o.skipMatcher.Add(methods...)
	}
}

// WithoutCredentialIdentity 禁用凭证身份
func WithoutCredentialIdentity() ACLOption {
	return func(o *aclOptions) {
		o.useCredentialIdentity = false
	}
}

// WithACLLogger 设置日志记录器
func WithACLLogger(logger InterceptorLogger) ACLOption {
	return func(o *aclOptions) {
		o.logger = logger
	}
}

// ===== 方法级权限检查 =====

// MethodPermission 方法级权限
type MethodPermission struct {
	Method      string   `json:"method" yaml:"method"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	Description string   `json:"description" yaml:"description"`
}

// MethodPermissionChecker 方法级权限检查器
type MethodPermissionChecker struct {
	permissions map[string][]string // method -> required permissions
	mu          sync.RWMutex
}

// NewMethodPermissionChecker 创建方法级权限检查器
func NewMethodPermissionChecker() *MethodPermissionChecker {
	return &MethodPermissionChecker{
		permissions: make(map[string][]string),
	}
}

// Register 注册方法所需的权限
func (c *MethodPermissionChecker) Register(method string, permissions ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissions[method] = permissions
}

// Check 检查是否有足够的权限
func (c *MethodPermissionChecker) Check(method string, userPermissions []string) error {
	c.mu.RLock()
	requiredPerms, ok := c.permissions[method]
	c.mu.RUnlock()

	if !ok {
		return nil // 方法没有配置权限要求，默认允许
	}

	userPermSet := make(map[string]bool)
	for _, p := range userPermissions {
		userPermSet[p] = true
	}

	for _, required := range requiredPerms {
		if !userPermSet[required] {
			return fmt.Errorf("missing permission: %s", required)
		}
	}

	return nil
}

// PermissionInterceptor 权限检查拦截器（用于更细粒度的权限控制）
func PermissionInterceptor(checker *MethodPermissionChecker) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var userPermissions []string

		if cred, ok := CredentialFromContext(ctx); ok {
			userPermissions = cred.Permissions
		}

		if err := checker.Check(info.FullMethod, userPermissions); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		return handler(ctx, req)
	}
}
