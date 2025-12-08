// Package interceptors 提供审计日志和监控指标拦截器
package interceptors

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ===== 默认审计日志记录器 =====

// DefaultAuditLogger 默认审计日志记录器（使用 InterceptorLogger 接口）
type DefaultAuditLogger struct {
	logger InterceptorLogger
}

// NewDefaultAuditLogger 创建默认审计日志记录器
func NewDefaultAuditLogger(logger InterceptorLogger) *DefaultAuditLogger {
	return &DefaultAuditLogger{logger: logger}
}

// Log 记录审计事件
func (l *DefaultAuditLogger) Log(event *AuditEvent) {
	fields := map[string]interface{}{
		"method":   event.Method,
		"service":  event.ServiceName,
		"status":   event.StatusCode,
		"duration": event.Duration.String(),
	}

	if event.ServiceNamespace != "" {
		fields["namespace"] = event.ServiceNamespace
	}
	if event.CredentialType != "" {
		fields["credential_type"] = event.CredentialType
	}
	if event.CredentialSubject != "" {
		fields["credential_subject"] = event.CredentialSubject
	}
	if event.ClientAddr != "" {
		fields["client_addr"] = event.ClientAddr
	}
	if event.Error != "" {
		fields["error"] = event.Error
	}
	if event.RequestID != "" {
		fields["request_id"] = event.RequestID
	}

	if l.logger != nil {
		if event.StatusCode == "OK" {
			l.logger.LogInfo("[AUDIT] gRPC call succeeded", fields)
		} else {
			l.logger.LogError("[AUDIT] gRPC call failed", fields)
		}
	}
}

// ===== 审计拦截器 =====

// AuditInterceptor 审计日志拦截器
func AuditInterceptor(logger AuditLogger, opts ...AuditOption) grpc.UnaryServerInterceptor {
	options := defaultAuditOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过不需要审计的方法
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(ctx, req)
		}

		start := time.Now()

		// 执行请求
		resp, err := handler(ctx, req)

		// 构建审计事件
		event := &AuditEvent{
			Timestamp: start,
			Method:    info.FullMethod,
			Duration:  time.Since(start),
		}

		// 提取服务身份
		if identity, ok := ServiceIdentityFromContext(ctx); ok {
			event.ServiceName = identity.ServiceName
			event.ServiceNamespace = identity.ServiceNamespace
			event.CertCN = identity.CommonName
			event.CertOU = identity.OrganizationalUnits
		}

		// 提取凭证信息
		if cred, ok := CredentialFromContext(ctx); ok {
			event.CredentialType = string(cred.Type)
			event.CredentialSubject = cred.Subject
		}

		// 提取请求 ID（如果提供了 RequestIDExtractor）
		if options.requestIDExtractor != nil {
			event.RequestID = options.requestIDExtractor(ctx)
		}

		// 提取状态码
		if err != nil {
			if st, ok := status.FromError(err); ok {
				event.StatusCode = st.Code().String()
				event.StatusMsg = st.Message()
			} else {
				event.StatusCode = codes.Internal.String()
				event.Error = err.Error()
			}
		} else {
			event.StatusCode = codes.OK.String()
		}

		// 记录审计日志
		if logger != nil {
			logger.Log(event)
		}

		return resp, err
	}
}

// AuditStreamInterceptor 流式审计日志拦截器
func AuditStreamInterceptor(logger AuditLogger, opts ...AuditOption) grpc.StreamServerInterceptor {
	options := defaultAuditOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if options.skipMatcher.Match(info.FullMethod) {
			return handler(srv, ss)
		}

		start := time.Now()
		err := handler(srv, ss)

		ctx := ss.Context()
		event := &AuditEvent{
			Timestamp: start,
			Method:    info.FullMethod,
			Duration:  time.Since(start),
		}

		if identity, ok := ServiceIdentityFromContext(ctx); ok {
			event.ServiceName = identity.ServiceName
			event.ServiceNamespace = identity.ServiceNamespace
		}

		if err != nil {
			if st, ok := status.FromError(err); ok {
				event.StatusCode = st.Code().String()
				event.StatusMsg = st.Message()
			} else {
				event.StatusCode = codes.Internal.String()
				event.Error = err.Error()
			}
		} else {
			event.StatusCode = codes.OK.String()
		}

		if logger != nil {
			logger.Log(event)
		}

		return err
	}
}

// ===== 审计选项 =====

type auditOptions struct {
	skipMatcher        *SkipMethodMatcher
	requestIDExtractor func(context.Context) string
}

func defaultAuditOptions() *auditOptions {
	return &auditOptions{
		skipMatcher: NewSkipMethodMatcher(DefaultSkipMethods()...),
	}
}

// AuditOption 审计选项函数
type AuditOption func(*auditOptions)

// WithAuditSkipMethods 设置跳过审计的方法
func WithAuditSkipMethods(methods ...string) AuditOption {
	return func(o *auditOptions) {
		o.skipMatcher.Add(methods...)
	}
}

// WithRequestIDExtractor 设置请求 ID 提取器
func WithRequestIDExtractor(extractor func(context.Context) string) AuditOption {
	return func(o *auditOptions) {
		o.requestIDExtractor = extractor
	}
}

// ===== 监控指标 =====

// AuthMetrics 认证授权监控指标
type AuthMetrics struct {
	// 认证统计
	AuthSuccess  uint64 // 认证成功次数
	AuthFailure  uint64 // 认证失败次数
	MTLSSuccess  uint64 // mTLS 认证成功
	MTLSFailure  uint64 // mTLS 认证失败
	TokenSuccess uint64 // Token 验证成功
	TokenFailure uint64 // Token 验证失败

	// 授权统计
	ACLAllowed uint64 // ACL 允许次数
	ACLDenied  uint64 // ACL 拒绝次数

	// 按服务统计
	serviceMetrics map[string]*ServiceMetrics
	mu             sync.RWMutex
}

// ServiceMetrics 服务级指标
type ServiceMetrics struct {
	ServiceName   string
	RequestCount  uint64
	SuccessCount  uint64
	FailureCount  uint64
	TotalDuration int64 // 纳秒
	LastRequestAt int64 // Unix 时间戳
}

// NewAuthMetrics 创建认证授权监控指标
func NewAuthMetrics() *AuthMetrics {
	return &AuthMetrics{
		serviceMetrics: make(map[string]*ServiceMetrics),
	}
}

// RecordAuthSuccess 记录认证成功
func (m *AuthMetrics) RecordAuthSuccess(authType string) {
	atomic.AddUint64(&m.AuthSuccess, 1)
	switch authType {
	case "mtls":
		atomic.AddUint64(&m.MTLSSuccess, 1)
	case "token", "bearer", "hmac", "api_key":
		atomic.AddUint64(&m.TokenSuccess, 1)
	}
}

// RecordAuthFailure 记录认证失败
func (m *AuthMetrics) RecordAuthFailure(authType string) {
	atomic.AddUint64(&m.AuthFailure, 1)
	switch authType {
	case "mtls":
		atomic.AddUint64(&m.MTLSFailure, 1)
	case "token", "bearer", "hmac", "api_key":
		atomic.AddUint64(&m.TokenFailure, 1)
	}
}

// RecordACLResult 记录 ACL 结果
func (m *AuthMetrics) RecordACLResult(allowed bool) {
	if allowed {
		atomic.AddUint64(&m.ACLAllowed, 1)
	} else {
		atomic.AddUint64(&m.ACLDenied, 1)
	}
}

// RecordServiceRequest 记录服务请求
func (m *AuthMetrics) RecordServiceRequest(serviceName string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, ok := m.serviceMetrics[serviceName]
	if !ok {
		metrics = &ServiceMetrics{ServiceName: serviceName}
		m.serviceMetrics[serviceName] = metrics
	}

	atomic.AddUint64(&metrics.RequestCount, 1)
	if success {
		atomic.AddUint64(&metrics.SuccessCount, 1)
	} else {
		atomic.AddUint64(&metrics.FailureCount, 1)
	}
	atomic.AddInt64(&metrics.TotalDuration, int64(duration))
	atomic.StoreInt64(&metrics.LastRequestAt, time.Now().Unix())
}

// GetMetrics 获取指标快照
func (m *AuthMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services := make(map[string]interface{})
	for name, sm := range m.serviceMetrics {
		reqCount := atomic.LoadUint64(&sm.RequestCount)
		avgDuration := float64(0)
		if reqCount > 0 {
			avgDuration = float64(atomic.LoadInt64(&sm.TotalDuration)) / float64(reqCount) / 1e6
		}

		services[name] = map[string]interface{}{
			"request_count":   reqCount,
			"success_count":   atomic.LoadUint64(&sm.SuccessCount),
			"failure_count":   atomic.LoadUint64(&sm.FailureCount),
			"avg_duration_ms": avgDuration,
			"last_request_at": time.Unix(atomic.LoadInt64(&sm.LastRequestAt), 0),
		}
	}

	return map[string]interface{}{
		"auth": map[string]interface{}{
			"success":       atomic.LoadUint64(&m.AuthSuccess),
			"failure":       atomic.LoadUint64(&m.AuthFailure),
			"mtls_success":  atomic.LoadUint64(&m.MTLSSuccess),
			"mtls_failure":  atomic.LoadUint64(&m.MTLSFailure),
			"token_success": atomic.LoadUint64(&m.TokenSuccess),
			"token_failure": atomic.LoadUint64(&m.TokenFailure),
		},
		"acl": map[string]interface{}{
			"allowed": atomic.LoadUint64(&m.ACLAllowed),
			"denied":  atomic.LoadUint64(&m.ACLDenied),
		},
		"services": services,
	}
}

// MetricsInterceptor 监控指标拦截器
func MetricsInterceptor(metrics *AuthMetrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// 获取服务名
		serviceName := "unknown"
		if identity, ok := ServiceIdentityFromContext(ctx); ok {
			serviceName = identity.ServiceName
		}

		// 记录指标
		success := err == nil
		metrics.RecordServiceRequest(serviceName, success, duration)

		return resp, err
	}
}

// ===== 告警规则 =====

// AlertRule 告警规则
type AlertRule struct {
	Name        string
	Description string
	Condition   func(metrics *AuthMetrics) bool
	Severity    string // critical, warning, info
}

// DefaultAlertRules 默认告警规则
func DefaultAlertRules() []*AlertRule {
	return []*AlertRule{
		{
			Name:        "high_auth_failure_rate",
			Description: "认证失败率过高",
			Severity:    "critical",
			Condition: func(m *AuthMetrics) bool {
				total := atomic.LoadUint64(&m.AuthSuccess) + atomic.LoadUint64(&m.AuthFailure)
				if total < 100 {
					return false
				}
				failureRate := float64(atomic.LoadUint64(&m.AuthFailure)) / float64(total)
				return failureRate > 0.1 // 10% 失败率告警
			},
		},
		{
			Name:        "high_acl_deny_rate",
			Description: "ACL 拒绝率过高",
			Severity:    "warning",
			Condition: func(m *AuthMetrics) bool {
				total := atomic.LoadUint64(&m.ACLAllowed) + atomic.LoadUint64(&m.ACLDenied)
				if total < 100 {
					return false
				}
				denyRate := float64(atomic.LoadUint64(&m.ACLDenied)) / float64(total)
				return denyRate > 0.05 // 5% 拒绝率告警
			},
		},
		{
			Name:        "mtls_failure_spike",
			Description: "mTLS 认证失败激增",
			Severity:    "critical",
			Condition: func(m *AuthMetrics) bool {
				return atomic.LoadUint64(&m.MTLSFailure) > 10 // 绝对数量告警
			},
		},
	}
}

// AlertChecker 告警检查器
type AlertChecker struct {
	rules   []*AlertRule
	metrics *AuthMetrics
	handler func(rule *AlertRule)
}

// NewAlertChecker 创建告警检查器
func NewAlertChecker(metrics *AuthMetrics, handler func(rule *AlertRule)) *AlertChecker {
	return &AlertChecker{
		rules:   DefaultAlertRules(),
		metrics: metrics,
		handler: handler,
	}
}

// AddRule 添加告警规则
func (c *AlertChecker) AddRule(rule *AlertRule) {
	c.rules = append(c.rules, rule)
}

// Check 检查告警
func (c *AlertChecker) Check() {
	for _, rule := range c.rules {
		if rule.Condition(c.metrics) {
			c.handler(rule)
		}
	}
}

// StartPeriodicCheck 启动定期检查
func (c *AlertChecker) StartPeriodicCheck(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Check()
		case <-stopCh:
			return
		}
	}
}
